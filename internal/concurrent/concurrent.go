// Try to get some speed up on large repos by running git log in parallel.
package concurrent

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const chunkSize = 5000

var nCPU int

func init() {
	nCPU = runtime.GOMAXPROCS(0)
}

type worker struct {
	id  int
	err chan error
}

func tallyCommitsByPath(
	ctx context.Context,
	revspec []string,
	paths []string,
	filters git.LogFilters,
	opts tally.TallyOpts,
) (_ tally.TalliesByPath, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running concurrent tally: %w", err)
		}
	}()

	// -- Get rev list --
	revs, err := git.RevList(ctx, revspec, paths, filters)
	if err != nil {
		return nil, err
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

	// q is our work queue. We will write batches of work to the queue to be
	// handled by our workers.
	q := func() <-chan []string {
		q := make(chan []string)
		go func() {
			defer close(q)
			defer logger().Debug("work enqueuer exited")

			i := 0
			for i < len(revs) {
				select {
				case <-ctx.Done():
					return
				case q <- revs[i:min(i+chunkSize, len(revs))]:
					i += chunkSize
				}
			}
		}()

		return q
	}()

	// Launches workers that consume from q and write to results and errors
	// that can be read by the main coroutine.
	results, errs := func() (<-chan tally.TalliesByPath, <-chan error) {
		in := make(chan []string)
		workers := make(chan worker, nCPU)
		results := make(chan tally.TalliesByPath)
		errs := make(chan error, 1)

		// Spawner. Creates new workers while we have free CPUs and work to do.
		go func() {
			defer close(in)
			defer close(workers)
			defer logger().Debug("spawner exited")

			nWorkers := 0

			for {
				var revs []string
				var ok bool

				select {
				case <-ctx.Done():
					return
				case revs, ok = <-q:
					if !ok {
						return
					}
				}

				// Spawn worker if we are still under count
				if nWorkers < nCPU {
					nWorkers += 1

					w := worker{
						id:  nWorkers,
						err: make(chan error, 1),
					}
					go func() {
						defer close(w.err)

						err := runWorker(ctx, w.id, in, results, paths, opts)
						if err != nil {
							w.err <- err
						}
					}()

					workers <- w
				}

				select {
				case <-ctx.Done():
					return
				case in <- revs: // Forward work to workers
				}
			}
		}()

		// Waiter. Waits for done or error for each one in turn. Forwards
		// errors to errs channel.
		go func() {
			defer close(results)
			defer close(errs)
			defer logger().Debug("waiter exited")

			for {
				var w worker
				var ok bool

				select {
				case <-ctx.Done():
					return
				case w, ok = <-workers:
					if !ok {
						return
					}
				}

				select {
				case <-ctx.Done():
					return
				case err, ok := <-w.err:
					if ok && err != nil {
						errs <- err
					}
					return
				}
			}
		}()

		return results, errs
	}()

	talliesByPath := tally.TalliesByPath{}

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("concurrent tally cancelled")
		case result, ok := <-results:
			if !ok {
				break loop
			}

			talliesByPath = talliesByPath.Combine(result)
		case err, ok := <-errs:
			if ok && err != nil {
				return nil, fmt.Errorf("concurrent tally failed: %w", err)
			}
		}
	}

	return talliesByPath, nil
}

func TallyCommits(
	ctx context.Context,
	revspec []string,
	paths []string,
	filters git.LogFilters,
	opts tally.TallyOpts,
) (_ map[string]tally.Tally, err error) {
	talliesByPath, err := tallyCommitsByPath(
		ctx,
		revspec,
		paths,
		filters,
		opts,
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
) (*tally.TreeNode, error) {
	talliesByPath, err := tallyCommitsByPath(
		ctx,
		revspec,
		paths,
		filters,
		opts,
	)
	if err != nil {
		return nil, err
	}

	return tally.TallyCommitsTreeFromPaths(talliesByPath, worktreePaths)
}
