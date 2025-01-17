// Try to get some speed up on large repos by running git log in parallel.
//
// Concurrency graph is something like:
//
//	      rev writer
//	          v
//	         ~q~
//	          v
//	       spawner
//	          v
//	         ~q2~
//	   v      v      v
//	worker  worker  worker ...
//	      v       v v v
//	  ~results~   waiter
//	        |       v
//	        |     ~errs~
//	        v    v
//	         main
package concurrent

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"runtime"
	"time"

	"github.com/sinclairtarget/git-who/internal/format"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/pretty"
	"github.com/sinclairtarget/git-who/internal/tally"
)

// We run one git log process for each chuck of this many revisions.
const chunkSize = 1024

var nCPU int

func init() {
	nCPU = runtime.GOMAXPROCS(0)
}

type tallyFunc[T any] func(
	commits iter.Seq2[git.Commit, error],
	opts tally.TallyOpts,
) (T, error)

type combinable[T any] interface {
	Combine(other T) T
}

// tally job we can do concurrently
type whoperation[T combinable[T]] struct {
	revspec []string
	paths   []string
	filters git.LogFilters
	tally   tallyFunc[T]
	opts    tally.TallyOpts
}

type worker struct {
	id  int
	err chan error
}

func calcTotalChunks(revCount int) int {
	return revCount/chunkSize + 1
}

func shouldShowProgress(chunks int) bool {
	return chunks > nCPU
}

func tallyFanOutFanIn[T combinable[T]](
	ctx context.Context,
	whop whoperation[T],
	allowProgressBar bool,
) (_ T, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running concurrent tally: %w", err)
		}
	}()

	var accumulator T

	// -- Get rev list ---------------------------------------------------------
	revs, err := git.RevList(ctx, whop.revspec, whop.paths, whop.filters)
	if err != nil {
		return accumulator, err
	}

	logger().Debug(
		"running concurrent tally",
		"revCount",
		len(revs),
		"nCPU",
		nCPU,
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// -- Fork -----------------------------------------------------------------
	q := func() <-chan []string {
		q := make(chan []string) // q is our work queue
		go func() {
			defer close(q)

			runWriter(ctx, revs, q)
		}()

		return q
	}()

	// Launches workers that consume from q and write to results and errors
	// that can be read by the main coroutine.
	results, errs := func() (<-chan T, <-chan error) {
		q2 := make(chan []string) // Intermediate work queue
		workers := make(chan worker, nCPU)
		results := make(chan T)
		errs := make(chan error, 1)

		go func() {
			defer close(q2)
			defer close(workers)

			runSpawner[T](ctx, whop, q, q2, workers, results)
		}()

		go func() {
			defer close(results)
			defer close(errs)

			runWaiter(ctx, workers, errs)
		}()

		return results, errs
	}()

	// -- Join -----------------------------------------------------------------
	// Read and combine results until results channel is closed, context is
	// cancelled, or we get a worker error
	totalChunks := calcTotalChunks(len(revs))
	chunksComplete := 0
	showProgress := allowProgressBar && shouldShowProgress(totalChunks)

	if showProgress {
		fmt.Printf("  0%% (0/%s commits)", format.Number(len(revs)))
	}

loop:
	for {
		select {
		case <-ctx.Done():
			return accumulator, errors.New("concurrent tally cancelled")
		case result, ok := <-results:
			if !ok {
				break loop
			}

			accumulator = accumulator.Combine(result)
			chunksComplete += 1

			if showProgress {
				fmt.Printf("%s\r", pretty.EraseLine)
				fmt.Printf(
					"%3.0f%% (%s/%s commits)",
					float32(chunksComplete)/float32(totalChunks)*100,
					format.Number(min(len(revs), chunksComplete*chunkSize)),
					format.Number(len(revs)),
				)
			}
		case err, ok := <-errs:
			if ok && err != nil {
				return accumulator, fmt.Errorf(
					"concurrent tally failed: %w",
					err,
				)
			}
		}
	}

	if showProgress {
		fmt.Printf("%s\r", pretty.EraseLine)
	}
	return accumulator, nil
}

func TallyCommits(
	ctx context.Context,
	revspec []string,
	paths []string,
	filters git.LogFilters,
	opts tally.TallyOpts,
	allowProgressBar bool,
) (_ map[string]tally.Tally, err error) {
	whop := whoperation[tally.TalliesByPath]{
		revspec: revspec,
		paths:   paths,
		filters: filters,
		tally:   tally.TallyCommitsByPath,
		opts:    opts,
	}

	talliesByPath, err := tallyFanOutFanIn[tally.TalliesByPath](
		ctx,
		whop,
		allowProgressBar,
	)
	if err != nil {
		return nil, err
	}

	return talliesByPath.Reduce(), nil
}

func TallyCommitsTree(
	ctx context.Context,
	revspec []string,
	paths []string,
	filters git.LogFilters,
	opts tally.TallyOpts,
	worktreePaths map[string]bool,
	gitRootPath string,
	allowProgressBar bool,
) (*tally.TreeNode, error) {
	whop := whoperation[tally.TalliesByPath]{
		revspec: revspec,
		paths:   paths,
		filters: filters,
		tally:   tally.TallyCommitsByPath,
		opts:    opts,
	}

	talliesByPath, err := tallyFanOutFanIn[tally.TalliesByPath](
		ctx,
		whop,
		allowProgressBar,
	)
	if err != nil {
		return nil, err
	}

	return tally.TallyCommitsTreeFromPaths(
		talliesByPath,
		worktreePaths,
		gitRootPath,
	)
}

func TallyCommitsTimeline(
	ctx context.Context,
	revspec []string,
	paths []string,
	filters git.LogFilters,
	opts tally.TallyOpts,
	end time.Time,
	allowProgressBar bool,
) ([]tally.TimeBucket, error) {
	f := func(
		commits iter.Seq2[git.Commit, error],
		opts tally.TallyOpts,
	) (tally.TimeSeries, error) {
		return tally.TallyCommitsByDate(commits, opts, end)
	}

	whop := whoperation[tally.TimeSeries]{
		revspec: revspec,
		paths:   paths,
		filters: filters,
		tally:   f,
		opts:    opts,
	}

	buckets, err := tallyFanOutFanIn[tally.TimeSeries](
		ctx,
		whop,
		allowProgressBar,
	)
	if err != nil {
		return nil, err
	}

	resolution := tally.CalcResolution(
		buckets[0].Time,
		buckets[len(buckets)-1].Time,
	)
	rebuckets := tally.Rebucket(buckets, resolution, end)
	return rebuckets, nil
}
