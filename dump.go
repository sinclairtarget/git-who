package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out the output of git log as seen by git who.
func dump(
	revs []string,
	paths []string,
	short bool,
	since string,
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
		"paths",
		paths,
		"short",
		short,
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

	var subprocess *git.Subprocess
	if short {
		subprocess, err = git.RunLogDiffless(ctx, revs, paths, filters)
	} else {
		subprocess, err = git.RunLog(ctx, revs, paths, filters)
	}
	if err != nil {
		return err
	}

	w := bufio.NewWriter(os.Stdout)

	lines := subprocess.StdoutLines()
	for line, err := range lines {
		if err != nil {
			w.Flush()
			return err
		}

		fmt.Fprintln(w, line)
	}

	w.Flush()

	err = subprocess.Wait()
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug("finished dump", "duration_ms", elapsed.Milliseconds())

	return nil
}
