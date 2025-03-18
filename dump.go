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
	paths []string,
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
		"paths",
		paths,
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

	var subprocess *git.Subprocess
	if short {
		subprocess, err = git.RunLog(ctx, revs, paths, filters, false)
	} else {
		subprocess, err = git.RunLog(ctx, revs, paths, filters, true)
	}
	if err != nil {
		return err
	}

	w := bufio.NewWriter(os.Stdout)

	lines := subprocess.StdoutLogLines()
	for line, err := range lines {
		if err != nil {
			w.Flush()
			return err
		}

		lineWithNulls := strings.ReplaceAll(line, "\x00", "^@")
		fmt.Fprintln(w, lineWithNulls)
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
