package backends

import (
	"iter"
	"slices"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/utils/iterutils"
)

type NoopBackend struct{}

func (b NoopBackend) Name() string {
	return "noop"
}

func (b NoopBackend) Size() int {
	return 0
}

func (b NoopBackend) Get(revs []string) (
	iter.Seq2[git.Commit, error], bool, error,
) {
	return iterutils.WithoutErrors(slices.Values([]git.Commit{})), false, nil
}

func (b NoopBackend) Add(commits []git.Commit) error {
	return nil
}

func (b NoopBackend) Clear() error {
	return nil
}
