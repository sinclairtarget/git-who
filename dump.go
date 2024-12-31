package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out the output of git log as seen by git who.
func dump(revs []string, paths []string, since string) (err error) {
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
		"since",
		since,
	)

	start := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subprocess, err := git.RunLog(ctx, revs, paths, since)
	if err != nil {
		return err
	}

	lines := subprocess.StdoutLines()
	for line, err := range lines {
		if err != nil {
			return err
		}

		fmt.Println(line)
	}

	err = subprocess.Wait()
	if err != nil {
		return err
	}

	elapsed := time.Now().Sub(start)
	logger().Debug("finished dump", "duration_ms", elapsed.Milliseconds())

	return nil
}
