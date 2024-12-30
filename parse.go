package main

import (
	"fmt"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out a simple representation of the commits parsed from `git log`
// for debugging.
func parse(revs []string, paths []string, since string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running \"parse\": %w", err)
		}
	}()

	logger().Debug(
		"called parse()",
		"revs",
		revs,
		"paths",
		paths,
		"since",
		since,
	)

	start := time.Now()

	commits, closer, err := git.CommitsSince(revs, paths, since)
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = closer()
		}
	}()

	numCommits := 0
	for commit, err := range commits {
		if err != nil {
			return fmt.Errorf("Error iterating commits: %w", err)
		}

		fmt.Printf("%s\n", commit)
		for _, diff := range commit.FileDiffs {
			fmt.Printf("  %s\n", diff)
		}

		fmt.Println()

		numCommits += 1
	}

	fmt.Printf("Parsed %d commits.\n", numCommits)

	elapsed := time.Now().Sub(start)
	logger().Debug("finished parse", "duration_ms", elapsed.Milliseconds())

	return nil
}
