package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out a simple representation of the commits parsed from `git log`
// for debugging.
func parse(
	revs []string,
	paths []string,
	short bool,
	since string,
	authors []string,
	nauthors []string,
) (err error) {
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
		"authors",
		authors,
		"nauthors",
		nauthors,
	)

	start := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	filters := git.LogFilters{
		Since:    since,
		Authors:  authors,
		Nauthors: nauthors,
	}
	commits, closer, err := git.CommitsWithOpts(ctx, revs, paths, filters, short)
	if err != nil {
		return err
	}

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

	err = closer()
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug("finished parse", "duration_ms", elapsed.Milliseconds())

	return nil
}
