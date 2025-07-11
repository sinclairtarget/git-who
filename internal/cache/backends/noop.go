package backends

import (
	"iter"
	"slices"

	"github.com/sinclairtarget/git-who/internal/git"
)

type NoopBackend struct{}

func (b NoopBackend) Name() string {
	return "noop"
}

func (b NoopBackend) Open() error {
	return nil
}

func (b NoopBackend) Close() error {
	return nil
}

func (b NoopBackend) Get(revs []string) (iter.Seq[git.Commit], func() error) {
	return slices.Values([]git.Commit{}), func() error { return nil }
}

func (b NoopBackend) Add(commits []git.Commit) error {
	return nil
}

func (b NoopBackend) Clear() error {
	return nil
}
