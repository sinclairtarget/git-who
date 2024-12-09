package main

import (
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
)

// The "table" subcommand summarizes the authorship history of the given
// commits and path in a table printed to stdout.
func table(revs []string, path string, useCsv bool) error {
	fmt.Printf("table() revs: %v, path: %s, useCsv: %t\n", revs, path, useCsv)

	output, err := git.LogLines(revs, path)
	if err != nil {
		return fmt.Errorf("failed to run git log: %w", err)
	}

	for l := range output.Lines {
		fmt.Printf("%s\n", l)
	}

	if output.Err != nil {
		return fmt.Errorf(
			"encountered error while iterating git log output: %w",
			output.Err,
		)
	}

	return nil
}
