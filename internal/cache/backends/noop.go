package backends

import (
	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/git"
)

type NoopBackend struct{}

func (b NoopBackend) Name() string {
	return "noop"
}

func (b NoopBackend) Size() int {
	return 0
}

func (b NoopBackend) Get(revs []string) (cache.Result, error) {
	return cache.Result{Revs: []string{}}, nil
}

func (b NoopBackend) Add(commits []git.Commit) error {
	return nil
}

func (b NoopBackend) Clear() error {
	return nil
}
