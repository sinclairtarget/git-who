package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out the output of git log as seen by git who.
func dump(
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
			err = fmt.Errorf("error running \"dump\": %w", err)
		}
	}()

	logger().Debug(
		"called revs()",
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

	subprocess, err := git.RunLog(
		ctx,
		revs,
		pathspecs,
		filters,
		!short,
		repoFiles.HasMailmap(),
	)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(os.Stdout)

	lines, finish := subprocess.StdoutNullDelimitedLines()
	for line := range lines {
		lineWithReplaced := strings.ReplaceAll(line, "\n", "\\n")
		fmt.Fprintln(w, lineWithReplaced)
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	err = finish()
	if err != nil {
		return err
	}

	err = subprocess.Wait()
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug("finished dump", "duration_ms", elapsed.Milliseconds())

	return nil
}
