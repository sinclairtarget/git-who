package git

import (
	"context"
	"fmt"
	"path/filepath"
)

// Handles splitting the Git revisions from the pathspecs given a list of args.
//
// We call git rev-parse to disambiguate.
func ParseArgs(args []string) (revs []string, pathspecs []string, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subprocess, err := RunRevParse(ctx, args)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse args: %w", err)
	}

	lines, finish := subprocess.StdoutLines()

	revs = []string{}
	pathspecs = []string{}

	pastRevs := false
	for line := range lines {
		if !pastRevs && isRev(line) {
			revs = append(revs, line)
		} else {
			pastRevs = true

			if line != "--" {
				// If user used backslashes as path separator on windows,
				// we want to turn into forward slashes
				pathspecs = append(pathspecs, filepath.ToSlash(line))
			}
		}
	}

	err = finish()
	if err != nil {
		err = fmt.Errorf("failed reading output of rev-parse: %w", err)
		return nil, nil, err
	}

	err = subprocess.Wait()
	if err != nil {
		return nil, nil, err
	}

	if len(revs) == 0 {
		// Default rev
		revs = append(revs, "HEAD")
	}

	return revs, pathspecs, nil
}
