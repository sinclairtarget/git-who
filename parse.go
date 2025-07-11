package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out a simple representation of the commits parsed from `git log`
// for debugging.
func parse(
	revs []string,
	pathspecs []string,
	short bool,
	since string,
	until string,
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
		"pathspecs",
		pathspecs,
		"short",
		short,
		"since",
		since,
		"until",
		until,
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
		Until:    until,
		Authors:  authors,
		Nauthors: nauthors,
	}

	gitRootPath, err := git.GetRoot()
	if err != nil {
		return err
	}

	repoFiles, err := git.CheckRepoConfigFiles(gitRootPath)
	if err != nil {
		return err
	}

	commits, finish := git.CommitsWithOpts(
		ctx,
		revs,
		pathspecs,
		filters,
		!short,
		repoFiles,
	)

	w := bufio.NewWriter(os.Stdout)

	numCommits := 0
	for commit := range commits {
		fmt.Fprintf(w, "%s\n", commit)
		for _, diff := range commit.FileDiffs {
			fmt.Fprintf(w, "  %s\n", diff)
		}

		fmt.Fprintln(w)

		numCommits += 1
	}

	w.Flush()

	err = finish()
	if err != nil {
		return err
	}

	fmt.Printf("Parsed %d commits.\n", numCommits)

	elapsed := time.Now().Sub(start)
	logger().Debug("finished parse", "duration_ms", elapsed.Milliseconds())

	return nil
}
