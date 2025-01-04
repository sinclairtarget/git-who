// Try to get some speed up on large repos by running git log in parallel
package concurrent

import (
	"context"
	"fmt"
	"iter"
	"runtime"

	"github.com/sinclairtarget/git-who/internal/git"
)

// A tally operation over an unrealized set of commits that we can divide
// among workers.
//
// The Merge() function is used to merge the tallies returned by each worker.
type Whoperation[T any] struct {
	Revs          []string
	Paths         []string
	Filters       git.LogFilters
	PopulateDiffs bool
	TallyFunc     func(commits iter.Seq2[git.Commit, error]) (T, error)
	Merge         func(a, b T) T
}

func getNWorkers(nCPU int, nCommits int, populateDiffs bool) int {
	var targetPerWorker int
	if populateDiffs {
		targetPerWorker = 10_000
	} else {
		targetPerWorker = 100_000
	}

	maxWorkers := nCPU*2 - 1
	return max(1, min(maxWorkers, nCommits/targetPerWorker+1))
}

func Tally[T any](ctx context.Context, whop Whoperation[T]) (_ T, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running concurrent tally: %w", err)
		}
	}()

	var t T

	nCommits, err := git.NumCommits(whop.Revs, whop.Paths, whop.Filters)
	if err != nil {
		return t, err
	}
	logger().Debug("got commit count with rev-list", "value", nCommits)

	nCPU := runtime.GOMAXPROCS(0)
	logger().Debug("cpus available", "value", nCPU)

	nWorkers := getNWorkers(nCPU, nCommits, whop.PopulateDiffs)
	logger().Debug("decided to use n workers", "value", nWorkers)

	return t, nil
}
