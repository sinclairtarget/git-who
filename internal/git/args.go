package git

import (
	"context"
	"fmt"
)

// Handles splitting the Git revisions from the paths given a list of args.
//
// We call git rev-parse to disambiguate.
func ParseArgs(args []string) (revs []string, paths []string, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subprocess, err := RunRevParse(ctx, args)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse args: %w", err)
	}

	lines := subprocess.StdoutLines()
	revs = []string{}
	paths = []string{}

	finishedRevs := false
	for line, err := range lines {
		if err != nil {
			return nil, nil, fmt.Errorf(
				"failed reading output of rev-parse: %w",
				err,
			)
		}

		if !finishedRevs && isRev(line) {
			revs = append(revs, line)
		} else {
			finishedRevs = true

			if line != "--" {
				paths = append(paths, line)
			}
		}
	}

	err = subprocess.Wait()
	if err != nil {
		return nil, nil, err
	}

	if len(revs) == 0 {
		// Default rev
		revs = append(revs, "HEAD")
	}

	return revs, paths, nil
}
