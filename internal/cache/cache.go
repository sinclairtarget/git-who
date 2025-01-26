package cache

import (
	"iter"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

type CacheBackend interface {
	Name() string
	Size() int
	Get(revs []string) (iter.Seq2[git.Commit, error], bool, error)
	Add(commits []git.Commit) error
	Clear() error
}

type Cache struct {
	backend CacheBackend
}

func NewCache(backend CacheBackend) Cache {
	return Cache{
		backend: backend,
	}
}

func (c *Cache) Name() string {
	return c.backend.Name()
}

func (c *Cache) Size() int {
	return c.backend.Size()
}

func (c *Cache) Get(revs []string) (iter.Seq2[git.Commit, error], error) {
	start := time.Now()

	commits, wasHit, err := c.backend.Get(revs)
	if err != nil {
		return nil, err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache get",
		"duration_ms",
		elapsed.Milliseconds(),
		"hit",
		wasHit,
	)

	return commits, nil
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
