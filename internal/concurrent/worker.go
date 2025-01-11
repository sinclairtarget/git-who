package concurrent

import (
	"context"
	"errors"
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
)

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
func runWaiter(
	ctx context.Context,
	workers <-chan worker,
	errs chan<- error,
) {
	logger().Debug("waiter started")
	defer logger().Debug("waiter exited")

	for {
		var w worker
		var ok bool

		select {
		case <-ctx.Done():
			return
		case w, ok = <-workers:
			if !ok {
				// Channel closed, no more workers
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		case err, ok := <-w.err:
			if ok && err != nil {
				errs <- err
				return // Exit on the first error
			}
		}
	}
}

// A tally worker that runs git log for each chunk of work.
func runWorker[T combinable[T]](
	ctx context.Context,
	id int,
	whop whoperation[T],
	in <-chan []string,
	results chan<- T,
) (err error) {
	logger := logger().With("workerId", id)
	logger.Debug("worker started")

	defer func() {
		if err != nil {
			err = fmt.Errorf("error in worker %d: %w", id, err)
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

			subprocess, err := git.RunStdinLog(ctx, whop.paths, true)
			if err != nil {
				return err
			}

			w, stdinCloser := subprocess.StdinWriter()

			// Write to git log stdin
			for _, rev := range revs {
				fmt.Fprintln(w, rev)
			}
			w.Flush()

			err = stdinCloser()
			if err != nil {
				return err
			}

			// Read parsed commits
			lines := subprocess.StdoutLines()
			commits := git.ParseCommits(lines)
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
