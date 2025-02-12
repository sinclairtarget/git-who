package backends

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"

	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/utils/iterutils"
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

func (b JSONBackend) Get(revs []string) (cache.Result, error) {
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
			return result, err
		}

		hit, _ := lookingFor[c.Hash]
		if hit {
			if isDup, _ := seen[c.Hash]; isDup {
				return result, fmt.Errorf(
					"duplicate commit in cache: %s",
					c.Hash,
				)
			}

			seen[c.Hash] = true
			commits = append(commits, c)
		}
	}

	return cache.Result{
		Commits: iterutils.WithoutErrors(slices.Values(commits)),
	}, nil
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
