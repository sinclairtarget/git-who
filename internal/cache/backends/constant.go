package backends

import (
	"errors"
	"slices"
	"time"

	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/utils/iterutils"
)

type ConstantBackend struct{}

var commits []git.Commit = []git.Commit{
	{
		ShortHash:   "9e9ea7662b1",
		Hash:        "9e9ea7662b1001d860471a4cece5e2f1de8062fb",
		AuthorName:  "Sinclair Target",
		AuthorEmail: "sinclair@chartbeat.com",
		Date: time.Date(
			2025, 1, 31, 16, 35, 26, 0, time.UTC,
		),
		Subject: "Use apikey header.",
		FileDiffs: []git.FileDiff{
			{
				Path:         "reactually/cb3po/dashboard/src/api/gsc.ts",
				LinesAdded:   3,
				LinesRemoved: 5,
			},
		},
	},
}

func (b ConstantBackend) Name() string {
	return "constant"
}

func (b ConstantBackend) Size() int {
	return len(commits)
}

func (b ConstantBackend) Get(revs []string) (cache.Result, error) {
	cached := map[string]git.Commit{}
	for _, commit := range commits {
		cached[commit.Hash] = commit
	}

	hits := []git.Commit{}
	for _, r := range revs {
		commit, ok := cached[r]
		if ok {
			hits = append(hits, commit)
		}
	}

	hitRevs := []string{}
	for _, c := range hits {
		hitRevs = append(hitRevs, c.Hash)
	}

	return cache.Result{
		Revs:    hitRevs,
		Commits: iterutils.WithoutErrors(slices.Values(hits)),
	}, nil
}

func (b ConstantBackend) Add(c []git.Commit) error {
	return errors.New("not implemented")
}

func (b ConstantBackend) Clear() error {
	return errors.New("not implemented")
}
