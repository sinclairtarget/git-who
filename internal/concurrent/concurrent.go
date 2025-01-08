// Try to get some speed up on large repos by running git log in parallel.
//
// We get the commit history as a list of revisions from git rev-list. We split
// the commit history up into roughly equal-sized sublists/workloads.
//
// We then launch, for each workload, a git log subprocess invoked with
// --stdin. We also launch one goroutine responsible for writing the sublist of
// revisions to the subprocess, and another goroutine responsible for reading
// from the log and tallying the results.
//
// Finally, when all workers are done, we merge the tallies computed for each
// workload in the main goroutine and return the result.
package concurrent

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

// A tally operation over an unrealized set of commits that we can divide
// among workers.
//
// The Merge() function is used to merge the tallies returned by each worker.
type Whoperation[T any, R any] struct {
	Revs          []string
	Paths         []string
	Filters       git.LogFilters
	PopulateDiffs bool
	Apply         tally.TallyFunc[T]
	Merge         tally.MergeFunc[T]
	Finalize      tally.FinalizeFunc[T, R]
	NWorkers      int
}

type workload[R any] struct {
	revs       []string
	subprocess *git.Subprocess
	writeError chan error
	tallyError chan error
	result     chan R
}

func newWorkload[R any]() workload[R] {
	return workload[R]{
		writeError: make(chan error, 1),
		tallyError: make(chan error, 1),
		result:     make(chan R, 1),
	}
}

func GetNWorkers(
	revs []string,
	paths []string,
	filters git.LogFilters,
	populateDiffs bool,
) (int, error) {
	nCommits, err := git.NumCommits(revs, paths, filters)
	if err != nil {
		return nCommits, fmt.Errorf("error computing n workers: %w", err)
	}
	logger().Debug("got commit count with rev-list", "value", nCommits)

	nCPU := runtime.GOMAXPROCS(0)
	logger().Debug("cpus available", "value", nCPU)

	var targetPerWorker int
	if populateDiffs {
		targetPerWorker = 10_000
	} else {
		targetPerWorker = 100_000
	}

	maxWorkers := nCPU*2 - 1
	nWorkers := max(1, min(maxWorkers, nCommits/targetPerWorker+1))
	logger().Debug("decided to use n workers", "value", nWorkers)

	return nWorkers, nil
}

func splitWork[T any](revs []string, nWorkers int) []workload[T] {
	workloads := make([]workload[T], nWorkers)
	for i := range nWorkers {
		workloads[i] = newWorkload[T]()
	}

	revsPerworkload := len(revs) / nWorkers
	for i, rev := range revs {
		workload_i := min(nWorkers-1, i/revsPerworkload)
		workloads[workload_i].revs = append(workloads[workload_i].revs, rev)
	}

	return workloads
}

func Tally[T any, R any](
	ctx context.Context,
	whop Whoperation[T, R],
) (_ R, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running concurrent tally: %w", err)
		}
	}()

	var noResult R

	// -- Get revs --
	revs, err := git.RevList(ctx, whop.Revs, whop.Paths, whop.Filters)
	if err != nil {
		return noResult, err
	}

	// -- Launch workers --
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workloads := splitWork[T](revs, whop.NWorkers)
	for i, workload := range workloads {
		logger().Debug(
			"launching workers for workload",
			"id",
			i,
			"firstRev",
			workload.revs[0],
			"nRevs",
			len(workload.revs),
		)

		subprocess, err := git.RunStdinLog(ctx, whop.PopulateDiffs)
		if err != nil {
			return noResult, err
		}

		go writeWorker(workload, subprocess)
		go tallyWorker(workload, subprocess, whop)

		workload.subprocess = subprocess
		workloads[i] = workload
	}

	// -- Join --
	var mergeVal T
	for i, workload := range workloads {
		logger().Debug("waiting for workload", "id", i)

	loop:
		for {
			select {
			case <-ctx.Done():
				return noResult, errors.New("concurrent tally cancelled")
			case err, ok := <-workload.writeError:
				if !ok {
					workload.writeError = nil
				}

				if err != nil {
					return noResult, fmt.Errorf(
						"error from writer goroutine for workload %d: %w",
						err,
						i,
					)
				}
			case err, ok := <-workload.tallyError:
				if !ok {
					workload.tallyError = nil
				}

				if err != nil {
					return noResult, fmt.Errorf(
						"error from writer goroutine for workload %d: %w",
						err,
						i,
					)
				}
			case r := <-workload.result:
				mergeVal = whop.Merge(mergeVal, r)
				break loop
			}
		}

		err = workload.subprocess.Wait()
		if err != nil {
			return noResult, err
		}
	}

	return whop.Finalize(mergeVal), nil
}
