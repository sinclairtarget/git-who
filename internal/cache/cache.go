// Cache for storing commits we've already diff-ed and parsed.
package cache

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"iter"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

func IsCachingEnabled() bool {
	if len(os.Getenv("GIT_WHO_DISABLE_CACHE")) > 0 {
		return false
	}

	return true
}

type Backend interface {
	Name() string
	Open() error
	Close() error
	Get(revs []string) (iter.Seq[git.Commit], func() error)
	Add(commits []git.Commit) error
	Clear() error
}

type Cache struct {
	backend Backend
}

func NewCache(backend Backend) Cache {
	return Cache{
		backend: backend,
	}
}

func (c *Cache) Name() string {
	return c.backend.Name()
}

func (c *Cache) Open() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error opening cache: %w", err)
		}
	}()

	start := time.Now()

	err = c.backend.Open()
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache open",
		"duration_ms",
		elapsed.Milliseconds(),
	)

	return nil
}

func (c *Cache) Close() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error closing cache: %w", err)
		}
	}()

	start := time.Now()

	err = c.backend.Close()
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache close",
		"duration_ms",
		elapsed.Milliseconds(),
	)

	return nil
}

func (c *Cache) Get(revs []string) (iter.Seq[git.Commit], func() error) {
	start := time.Now()

	commits, finish := c.backend.Get(revs)

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache get",
		"duration_ms",
		elapsed.Milliseconds(),
	)

	return commits, func() error {
		err := finish()
		if err != nil {
			err = fmt.Errorf("failed to retrieve from cache: %w", err)
		}

		return err
	}
}

func (c *Cache) Add(commits []git.Commit) error {
	start := time.Now()

	err := c.backend.Add(commits)
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache add",
		"duration_ms",
		elapsed.Milliseconds(),
	)

	return nil
}

func (c *Cache) Clear() error {
	err := c.backend.Clear()
	if err != nil {
		return err
	}

	logger().Debug("cache clear")
	return nil
}

// Returns the absolute path at which we should store data for a given cache
// backend.
//
// Tries to store it under the XDG_CACHE_HOME dir.
func CacheStorageDir(name string) (_ string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to determine cache storage path: %w", err)
		}
	}()

	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	cacheHome := filepath.Join(usr.HomeDir, ".cache")
	if len(os.Getenv("XDG_CACHE_HOME")) > 0 {
		cacheHome = os.Getenv("XDG_CACHE_HOME")
	}

	p := filepath.Join(cacheHome, "git-who", name)
	absP, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	return absP, nil
}

// Hash of all the state in the repo that affects the validity of our cache
func RepoStateHash(rf git.RepoConfigFiles) (string, error) {
	h := fnv.New32()
	err := rf.MailmapHash(h)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
