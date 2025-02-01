package cache

import (
	"iter"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

type Result struct {
	Revs    []string                     // All commit hashes in the sequence
	Commits iter.Seq2[git.Commit, error] // The sequence of commits
}

func (r Result) WasHit() bool {
	return len(r.Revs) > 0
}

type Backend interface {
	Name() string
	Size() int
	Get(revs []string) (Result, error)
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

func (c *Cache) Size() int {
	return c.backend.Size()
}

func (c *Cache) Get(revs []string) (Result, error) {
	start := time.Now()

	var result Result

	result, err := c.backend.Get(revs)
	if err != nil {
		return result, err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug(
		"cache get",
		"duration_ms",
		elapsed.Milliseconds(),
		"hit",
		result.WasHit(),
	)

	return result, nil
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
