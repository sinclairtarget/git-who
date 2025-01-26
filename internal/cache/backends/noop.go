package backends

import (
	"github.com/sinclairtarget/git-who/internal/git"
)

type NoopBackend struct{}

func (b NoopBackend) Name() string {
	return "noop"
}

func (b NoopBackend) Get(revs []string) ([]git.Commit, error) {
	return []git.Commit{}, nil
}

func (b NoopBackend) Add(commits []git.Commit) error {
	return nil
}

func (b NoopBackend) Wipe() error {
	return nil
}
