package main

import (
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

// The "table" subcommand summarizes the authorship history of the given
// commits and path in a table printed to stdout.
func table(revs []string, path string, useCsv bool) error {
	fmt.Printf("table() revs: %v, path: %s, useCsv: %t\n", revs, path, useCsv)

	lines, err := git.LogLines(revs, path)
	if err != nil {
		return fmt.Errorf("failed to run git log: %w", err)
	}

	tallies, err := tally.TallyCommits(git.ParseCommits(lines))
	if err != nil {
		return fmt.Errorf("failed to tally commits: %w", err)
	}

	for _, tally := range tallies {
		fmt.Println(tally)
	}

	return nil
}
