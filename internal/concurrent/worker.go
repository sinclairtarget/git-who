package concurrent

import (
	"context"
	"errors"
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

// A tally worker that manages its own git log subprocess.
func runWorker(
	ctx context.Context,
	id int,
	in <-chan []string,
	results chan<- tally.TalliesByPath,
	opts tally.TallyOpts,
) (err error) {
	logger := logger().With("workerId", id)
	defer func() {
		if err != nil {
			err = fmt.Errorf("error in worker %d: %w", id, err)
		}

		logger.Debug("worker exited")
	}()

	logger.Debug("worker started")

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

			nRevs := len(revs)
			logger.Debug("read revs", "count", nRevs)

			subprocess, err := git.RunStdinLog(ctx, true)
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
			result, err := tally.TallyCommitsByPath(commits, opts)
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
