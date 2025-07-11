package backends

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"slices"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Stores commits on disk at a particular filepath.
//
// Commits are stored as newline-delimited JSON. For now, all commits that match
// the revs being searched for are loaded into memory before being returned.
type JSONBackend struct {
	Path string
}

func (b JSONBackend) Name() string {
	return "json"
}

func (b JSONBackend) Open() error {
	return nil
}

func (b JSONBackend) Close() error {
	return nil
}

// Reads all commits into memory!
func (b JSONBackend) Get(revs []string) (iter.Seq[git.Commit], func() error) {
	lookingFor := map[string]bool{}
	for _, rev := range revs {
		lookingFor[rev] = true
	}

	var iterErr error
	empty := slices.Values([]git.Commit{})
	finish := func() error {
		return iterErr
	}

	f, err := os.Open(b.Path)
	if errors.Is(err, fs.ErrNotExist) {
		// If file doesn't exist, don't treat as an error
		return empty, finish
	} else if err != nil {
		iterErr = err
		return empty, finish
	}
	defer f.Close() // Don't care about error closing when reading

	dec := json.NewDecoder(f)

	var commits []git.Commit

	// In theory we shouldn't get any duplicates into the cache if we're
	// careful about what we write to it. But let's make sure by detecting dups
	// and throwing an error if we see one.
	seen := map[string]bool{}

	for {
		var c git.Commit

		err = dec.Decode(&c)
		if err == io.EOF {
			break
		} else if err != nil {
			iterErr = err
			return slices.Values(commits), finish
		}

		hit, _ := lookingFor[c.Hash]
		if hit {
			if isDup, _ := seen[c.Hash]; isDup {
				iterErr = fmt.Errorf(
					"duplicate commit in cache: %s",
					c.Hash,
				)
				return slices.Values(commits), finish
			}

			seen[c.Hash] = true
			commits = append(commits, c)
		}
	}

	return slices.Values(commits), finish
}

func (b JSONBackend) Add(commits []git.Commit) (err error) {
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

	enc := json.NewEncoder(f)

	for _, c := range commits {
		err = enc.Encode(&c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b JSONBackend) Clear() error {
	return os.Remove(b.Path)
}
