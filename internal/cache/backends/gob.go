package backends

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"

	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/git"
)

// Stores commits on disk at a particular filepath.
//
// Commits are stored in Gob format. The file stored on disk is a series of
// Gob-encoded arrays, each prefixed with a four-byte value indicating the
// number of bytes in the next array. This framing creates redundancy (since
// the Gob type metadata is repeated for each array) but allows us to append to
// the file on disk instead of replacing the whole file when we want to cache
// new commits.
//
// The Gob backend produces a cache file roughly half the size of the JSON
// backend on disk. It's also SIGNIFICANTLY faster to read the cache from disk.
type GobBackend struct {
	Path      string
	wasOpened bool
	isDirty   bool
}

func (b *GobBackend) Name() string {
	return "gob"
}

func (b *GobBackend) compressedPath() string {
	return b.Path + ".gz"
}

func (b *GobBackend) Open() (err error) {
	b.wasOpened = true

	// Uncompress gzipped file to regular location if it exists
	f, err := os.Open(b.compressedPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()

	fout, err := os.OpenFile(b.Path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fout.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(fout)
	_, err = io.Copy(w, zr)
	if err != nil {
		return err
	}

	err = zr.Close()
	if err != nil {
		return err
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (b *GobBackend) Close() (err error) {
	if b.isDirty {
		// Compress file and save to gzipped location
		f, err := os.Open(b.Path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		defer f.Close()

		fout, err := os.OpenFile(b.compressedPath(), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer fout.Close()

		r := bufio.NewReader(f)
		zw := gzip.NewWriter(fout)

		_, err = io.Copy(zw, r)
		if err != nil {
			return err
		}

		err = zw.Close()
		if err != nil {
			return err
		}
	}

	// Remove uncompressed file
	err = os.RemoveAll(b.Path)
	if err != nil {
		return err
	}

	return nil
}

func (b *GobBackend) Get(revs []string) (_ cache.Result, err error) {
	if !b.wasOpened {
		panic("cache not yet open. Did you forget to call Open()?")
	}

	result := cache.EmptyResult()

	lookingFor := map[string]bool{}
	for _, rev := range revs {
		lookingFor[rev] = true
	}

	f, err := os.Open(b.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return result, nil
	} else if err != nil {
		return result, err
	}

	// In theory we shouldn't get any duplicates into the cache if we're
	// careful about what we write to it. But let's make sure by detecting dups
	// and throwing an error if we see one.
	seen := map[string]bool{}

	it := func(yield func(git.Commit, error) bool) {
		defer f.Close() // Don't care about error closing when reading

		for {
			var commit git.Commit

			// -- Find length of next gob in bytes --
			prefix := make([]byte, 4)
			_, err = f.Read(prefix)
			if err == io.EOF {
				return
			} else if err != nil {
				yield(commit, err)
				return
			}

			var size uint32
			err = binary.Read(
				bytes.NewReader(prefix),
				binary.LittleEndian,
				&size,
			)
			if err != nil {
				yield(commit, err)
				return
			}

			// -- Decode next gob --
			var commits []git.Commit

			data := make([]byte, size)
			_, err = f.Read(data)

			dec := gob.NewDecoder(bytes.NewReader(data))
			err = dec.Decode(&commits)
			if err != nil {
				yield(commit, err)
				return
			}

			// -- Yield matching commits --
			for _, c := range commits {
				hit, _ := lookingFor[c.Hash]
				if hit {
					if isDup, _ := seen[c.Hash]; isDup {
						yield(c, fmt.Errorf(
							"duplicate commit in cache: %s",
							c.Hash,
						))
						return
					}

					seen[c.Hash] = true
					if !yield(c, nil) {
						return
					}
				}
			}
		}
	}

	return cache.Result{Commits: it}, nil
}

func (b *GobBackend) Add(commits []git.Commit) (err error) {
	if !b.wasOpened {
		panic("cache not yet open. Did you forget to call Open()?")
	}

	b.isDirty = true

	f, err := os.OpenFile(
		b.Path,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0644,
	)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
	}()

	var data bytes.Buffer

	enc := gob.NewEncoder(&data)
	err = enc.Encode(&commits)
	if err != nil {
		return err
	}

	if data.Len() > 0xFFFF_FFFF {
		return errors.New(
			"cannot add more than 4,294,976,296 bytes to cache at once", // lol
		)
	}

	err = binary.Write(f, binary.LittleEndian, uint32(data.Len()))
	if err != nil {
		return err
	}

	_, err = f.Write(data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (b *GobBackend) Clear() error {
	err := os.RemoveAll(b.Path)
	if err != nil {
		return err
	}

	err = os.RemoveAll(b.compressedPath())
	if err != nil {
		return err
	}

	return nil
}

// Returns the absolute path at which we should store the Gob data.
//
// Tries to store it under the XDG_CACHE_HOME dir.
func GobCachePathXDG(gitRootPath string) (string, error) {
	// Filename includes hash of path to repo so we don't collide with other
	// git-who caches for other repos.
	h := fnv.New32()
	h.Write([]byte(gitRootPath))
	filename := fmt.Sprintf("%x.gobs", h.Sum32())

	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	cacheHome := filepath.Join(usr.HomeDir, ".cache")
	if len(os.Getenv("XDG_CACHE_HOME")) > 0 {
		cacheHome = os.Getenv("XDG_CACHE_HOME")
	}

	p := filepath.Join(cacheHome, "git-who", filename)
	absP, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	return absP, nil
}
