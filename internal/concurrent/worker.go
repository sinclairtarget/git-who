package concurrent

import (
	"context"
	"errors"
	"fmt"

	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/git"
)

type worker struct {
	id  int
	err chan error
}

// Write chunks of work to our work queue to be handled by workers downstream.
func runWriter(ctx context.Context, revs []string, q chan<- []string) {
	logger().Debug("writer started")
	defer logger().Debug("writer exited")

	i := 0
	for i < len(revs) {
		select {
		case <-ctx.Done():
			return
		case q <- revs[i:min(i+chunkSize, len(revs))]:
			i += chunkSize
		}
	}
}

// Spawner. Creates new workers while we have free CPUs and work to do.
func runSpawner[T combinable[T]](
	ctx context.Context,
	whop whoperation[T],
	q <-chan []string,
	q2 chan []string,
	workers chan<- worker,
	results chan<- T,
	toCache chan<- []git.Commit,
) {
	logger().Debug("spawner started")
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
				// Channel closed, no more work
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

				err := runWorker[T](
					ctx,
					w.id,
					whop,
					q2,
					results,
					toCache,
				)
				if err != nil {
					w.err <- err
				}
			}()

			workers <- w
		}

		select {
		case <-ctx.Done():
			return
		case q2 <- revs: // Forward work to workers
		}
	}
}

// Waiter. Waits for done or error for each one in turn. Forwards
// errors to errs channel.
func runWaiter(workers <-chan worker, errs chan<- error) {
	logger().Debug("waiter started")
	defer logger().Debug("waiter exited")

	for w := range workers {
		logger().Debug("waiting on worker", "workerId", w.id)

		err, ok := <-w.err
		if ok && err != nil {
			errs <- err
		}
	}
}

// Cacher. Writes parsed commits to the cache.
func runCacher(
	ctx context.Context,
	cache *cache.Cache,
	toCache <-chan []git.Commit,
) (err error) {
	logger().Debug("cacher started")

	defer func() {
		if err != nil {
			err = fmt.Errorf("error in cacher: %w", err)
		}

		logger().Debug("cacher exited")
	}()

loop:
	for {
		select {
		case <-ctx.Done():
			return errors.New("cacher cancelled")
		case commits, ok := <-toCache:
			if !ok {
				break loop
			}

			err := cache.Add(commits)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// A tally worker that runs git log for each chunk of work.
func runWorker[T combinable[T]](
	ctx context.Context,
	id int,
	whop whoperation[T],
	in <-chan []string,
	results chan<- T,
	toCache chan<- []git.Commit,
) (err error) {
	logger := logger().With("workerId", id)
	logger.Debug("worker started")

	defer func() {
		if err != nil {
			err = fmt.Errorf("error in worker %d: %w", id, err)
			logger.Debug("worker exiting with error")
		}

		logger.Debug("worker exited")
	}()

loop:
	for {
		select {
		case <-ctx.Done():
			return errors.New("worker cancelled")
		case revs, ok := <-in:
			if !ok {
				if err != nil {
					return err
				}

				break loop // We're done, input channel is closed
			}

			// We pass an empty array of paths here. Even if we are only
			// tallying commits that affected certain paths, we want to make
			// sure that the diffs we get include ALL paths touched by each
			// commit. Otherwise when we cache the commits we would be caching
			// only a part of the commit
			nopaths := []string{}
			subprocess, err := git.RunStdinLog(
				ctx,
				nopaths,
				true,
				whop.useMailmap,
			)
			if err != nil {
				return err
			}

			w, stdinCloser := subprocess.StdinWriter()

			// Write revs to git log stdin
			for _, rev := range revs {
				fmt.Fprintln(w, rev)
			}
			w.Flush()

			err = stdinCloser()
			if err != nil {
				return err
			}

			// Read parsed commits and enqueue for caching
			lines := subprocess.StdoutNullDelimitedLines()
			commits := git.ParseCommits(lines)
			commits = cacheTee(commits, toCache)

			// Now that we're tallying, we DO care to only look at the file
			// diffs under the given paths
			commits = git.LimitDiffsByPathspec(commits, whop.pathspecs)

			result, err := whop.tally(commits, whop.opts)
			if err != nil {
				return err
			}

			err = subprocess.Wait()
			if err != nil {
				return err
			}

			results <- result
		}
	}

	return nil
}
