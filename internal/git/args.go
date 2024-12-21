package git

import (
	"fmt"
	"regexp"
)

// Handles splitting the Git revisions from the paths given a list of args.
//
// We call git rev-parse to disambiguate.
func ParseArgs(args []string) (revs []string, paths []string, err error) {
	subprocess, err := RunRevParse(args)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse args: %w", err)
	}

	defer func() {
		if err == nil {
			err = subprocess.Wait()
		} else {
			subprocess.Wait()
		}
	}()

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

	if len(revs) == 0 {
		// Default rev
		revs = append(revs, "HEAD")
	}

	if len(paths) == 0 {
		// Default path
		paths = append(paths, ".")
	}

	return revs, paths, nil
}

// Returns true if this is a (full-length) Git revision hash, false otherwise.
//
// We also need to handle a hash with "^" in front.
func isRev(s string) bool {
	matched, err := regexp.MatchString(`[\^a-f0-9]+`, s)
	if err != nil {
		logger().Error("Bad regexp!")
	}

	return matched && (len(s) == 40 || len(s) == 41)
}
