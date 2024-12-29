package main

import (
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out a simple representation of the parsed commits for debugging.
func parse(revs []string, paths []string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running \"parse\": %w", err)
		}
	}()

	logger().Debug("called parse()", "revs", revs, "paths", paths)

	commits, closer, err := git.Commits(revs, paths)
	if err != nil {
		return err
	}

	defer func() {
		errClose := closer()
		if err == nil {
			err = errClose
		}
	}()

	for commit := range commits {
		fmt.Printf("%s\n", commit)
		for _, diff := range commit.FileDiffs {
			fmt.Printf("  %s\n", diff)
		}

		fmt.Println()
	}

	return nil
}
