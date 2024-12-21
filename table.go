package main

import (
	"encoding/csv"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

// The "table" subcommand summarizes the authorship history of the given
// commits and paths in a table printed to stdout.
func table(revs []string, paths []string, useCsv bool) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running \"table\": %w", err)
		}
	}()

	logger().Debug(
		"called table()",
		"revs",
		revs,
		"paths",
		paths,
		"useCsv",
		useCsv,
	)

	tallies, err := func() (_ map[string]tally.Tally, err error) {
		commits, closer, err := git.Commits(revs, paths)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err == nil {
				err = closer()
			}
		}()

		tallies, err := tally.TallyCommits(commits)
		if err != nil {
			return nil, err
		}

		return tallies, nil
	}()
	if err != nil {
		return fmt.Errorf("failed to tally commits: %w", err)
	}

	sorted := slices.SortedFunc(
		maps.Values(tallies),
		func(a, b tally.Tally) int {
			if a.Commits > b.Commits {
				return -1
			} else if a.Commits == b.Commits {
				return 0
			} else {
				return 1
			}
		},
	)

	if useCsv {
		err := writeCsv(sorted)
		if err != nil {
			return err
		}
	} else {
		writeTable(sorted)
	}

	return nil
}

func toRecord(t tally.Tally) []string {
	record := make([]string, 0)
	return append(
		record,
		t.AuthorName,
		t.AuthorEmail,
		strconv.Itoa(t.Commits),
		strconv.Itoa(t.LinesAdded),
		strconv.Itoa(t.LinesRemoved),
		strconv.Itoa(t.FileCount),
	)
}

func writeCsv(tallies []tally.Tally) error {
	w := csv.NewWriter(os.Stdout)

	// Write header
	w.Write([]string{
		"name",
		"email",
		"commits",
		"lines added",
		"lines removed",
		"files",
	})

	for _, tally := range tallies {
		record := toRecord(tally)
		if err := w.Write(record); err != nil {
			return fmt.Errorf("error writing CSV record to stdout: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	return nil
}

func writeTable(tallies []tally.Tally) {
	fmt.Printf("%s\t%s\t%s\n", "Email", "Author", "Commits")
	for _, tally := range tallies {
		fmt.Printf("%s\t%s\t%d\n",
			tally.AuthorEmail,
			tally.AuthorName,
			tally.Commits,
		)
	}
}
