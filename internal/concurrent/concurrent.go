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
//	      v       v v v          v
//	  ~results~   waiter       cacher
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

	"github.com/sinclairtarget/git-who/internal/cache"
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

func calcTotalChunks(revCount int) int {
	return revCount/chunkSize + 1
}

func shouldShowProgress(chunks int) bool {
	return chunks > nCPU
}

// All the strings in the first array minus the strings in the second array
func setDiff(a []string, b []string) []string {
	m := map[string]bool{}
	for _, rev := range b {
		m[rev] = true
	}

	ret := []string{}
	for _, rev := range a {
		if _, ok := m[rev]; !ok {
			ret = append(ret, rev)
		}
	}

	return ret
}

func accumulateCached[T combinable[T]](
	whop whoperation[T],
	c cache.Cache,
	revs []string,
) (T, []string, error) {
	var none T

	result, err := c.Get(revs)
	if err != nil {
		return none, revs, err
	}

	commits := git.LimitDiffsByPath(result.Commits, whop.paths)

	foundRevs := []string{}
	accumulator, err := whop.tally(revTee(commits, &foundRevs), whop.opts)
	if err != nil {
		return none, revs, err
	}

	logger().Debug("commits found in cache", "num", len(foundRevs))
	return accumulator, setDiff(revs, foundRevs), nil
}

func handleCacheFailure(c cache.Cache, err error) error {
	// Graceful handling of cache error. Wipe cache and move on without it
	logger().Warn(
		fmt.Sprintf("error reading from cache (maybe corrupt?): %v", err),
	)
	logger().Warn("wiping cache and moving on")
	return c.Clear()
}

func tallyFanOutFanIn[T combinable[T]](
	ctx context.Context,
	whop whoperation[T],
	cache cache.Cache,
	allowProgressBar bool,
) (_ T, _err error) {
	defer func() {
		if _err != nil {
			_err = fmt.Errorf("error running concurrent tally: %w", _err)
		}
	}()

	var accumulator T

	// -- Get rev list ---------------------------------------------------------
	revs, err := git.RevList(ctx, whop.revspec, whop.paths, whop.filters)
	if err != nil {
		return accumulator, err
	}

	if len(revs) == 0 {
		logger().Debug("no commits found; no work to do")
		return accumulator, nil
	}

	// -- Use cached commits if there are any ----------------------------------
	remainingRevs := revs

	err = cache.Open()
	defer func() {
		err = cache.Close()
	}()

	if err == nil {
		accumulator, remainingRevs, err = accumulateCached(whop, cache, revs)
		if err != nil {
			err = handleCacheFailure(cache, err)
			if err != nil {
				return accumulator, err
			}
		} else if len(remainingRevs) == 0 {
			logger().Debug("all commits read from cache")
			return accumulator, nil
		}
	} else {
		err = handleCacheFailure(cache, err)
		if err != nil {
			return accumulator, err
		}
	}

	// -- Fork -----------------------------------------------------------------
	logger().Debug(
		"running concurrent tally",
		"revCount",
		len(remainingRevs),
		"nCPU",
		nCPU,
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	q := func() <-chan []string {
		q := make(chan []string) // q is our work queue
		go func() {
			defer close(q)

			runWriter(ctx, remainingRevs, q)
		}()

		return q
	}()

	// Launches workers that consume from q and write to results and errors
	// that can be read by the main coroutine.
	results, errs, cacheErr := func() (<-chan T, <-chan error, <-chan error) {
		q2 := make(chan []string) // Intermediate work queue
		workers := make(chan worker, nCPU)
		toCache := make(chan []git.Commit)
		results := make(chan T)
		errs := make(chan error, 1)

		go func() {
			defer close(q2)
			defer close(workers)

			runSpawner[T](ctx, whop, q, q2, workers, results, toCache)
		}()

		go func() {
			defer close(toCache)
			defer close(results)
			defer close(errs)

			runWaiter(workers, errs)
		}()

		cacheErr := make(chan error, 1)
		go func() {
			defer close(cacheErr)

			err := runCacher(ctx, &cache, toCache)
			if err != nil {
				cacheErr <- err
			}
		}()

		return results, errs, cacheErr
	}()

	// -- Join -----------------------------------------------------------------
	// Read and combine results until results channel is closed, context is
	// cancelled, or we get a worker error
	totalChunks := calcTotalChunks(len(remainingRevs))
	chunksComplete := 0
	showProgress := allowProgressBar && shouldShowProgress(totalChunks)

	if showProgress {
		fmt.Printf("  0%% (0/%s commits)", format.Number(len(remainingRevs)))
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
					format.Number(min(len(remainingRevs), chunksComplete*chunkSize)),
					format.Number(len(remainingRevs)),
				)
			}
		case err, ok := <-errs:
			if ok && err != nil {
				logger().Debug("error in concurrent tally; cancelling")
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

	// Check if there was a caching error (and wait for cacher to exit)
	select {
	case <-ctx.Done():
		return accumulator, errors.New("concurrent tally cancelled")
	case err, ok := <-cacheErr:
		if ok && err != nil {
			return accumulator, err
		}
	}

	return accumulator, nil
}

func TallyCommits(
	ctx context.Context,
	revspec []string,
	paths []string,
	filters git.LogFilters,
	opts tally.TallyOpts,
	cache cache.Cache,
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
		cache,
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
	cache cache.Cache,
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
		cache,
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
	cache cache.Cache,
	allowProgressBar bool,
) ([]tally.TimeBucket, error) {
	f := func(
		commits iter.Seq2[git.Commit, error],
		opts tally.TallyOpts,
	) (tally.TimeSeries, error) {
		return tally.TallyCommitsByDate(commits, opts)
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
		cache,
		allowProgressBar,
	)
	if err != nil {
		return nil, err
	}

	if end.IsZero() {
		end = buckets[len(buckets)-1].Time
	}
	resolution := tally.CalcResolution(buckets[0].Time, end)
	rebuckets := tally.Rebucket(buckets, resolution, end)
	return rebuckets, nil
}
