package cache

import (
	"fmt"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

type Backend interface {
	Name() string
	Get(revs []string) ([]git.Commit, error)
	Add(commits []git.Commit) error
	Wipe() error
}

var backend Backend

func UseBackend(b Backend) {
	logger().Debug(fmt.Sprintf("using backend %s", b.Name()))
	backend = b
}

func Get(revs []string) ([]git.Commit, error) {
	start := time.Now()

	commits, err := backend.Get(revs)
	if err != nil {
		return nil, err
	}

	elapsed := time.Now().Sub(start)
	wasHit := len(commits) > 0
	logger().Debug(
		"cache get",
		"duration_ms",
		elapsed.Milliseconds(),
		"hit",
		wasHit,
	)

	return commits, nil
}

func Add(commits []git.Commit) error {
	start := time.Now()

	err := backend.Add(commits)
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

func Wipe() error {
	return backend.Wipe()
}
