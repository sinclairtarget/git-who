package main

import (
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Just prints out the output of git log as seen by git who.
func dump(revs []string, paths []string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running \"dump\": %w", err)
		}
	}()

	subprocess, err := git.RunLog(revs, paths)
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
	return err
}
