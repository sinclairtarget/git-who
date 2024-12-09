/*
* Wraps access to data needed from Git.
*
* We invoke Git directly as a subprocess and parse the output rather than using
* git2go/libgit2.
*/
package git

import (
	"bufio"
	"fmt"
	"iter"
	"os/exec"
	"slices"
)

// Whether we rank authors by commit, lines, or files.
type TallyMode int

const (
    CommitMode TallyMode = iota
    LinesMode 
    FilesMode
)

// Output from a Git CLI command
type CmdOutput struct {
	Lines iter.Seq[string]
	Err error
}

var baseArgs = []string {
	"log",
	"--pretty=format:%H:%h%n%an%n%ae%n%ad%n%s",
	"--numstat",
	"--date=unix",
}

// Runs git log and returns an iterator over each line of the output
func LogLines(revs []string, path string) (*CmdOutput, error) {
	args := slices.Concat(baseArgs, revs, []string { path })

	cmd := exec.Command("git", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start subprocess: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	var it CmdOutput = CmdOutput {}

	it.Lines = func(yield func(string) bool) {
		for scanner.Scan() {
			if !yield(scanner.Text()) {
				break
			}
		}

		err := scanner.Err()
		if err != nil {
			it.Err = fmt.Errorf("failed to scan stdout: %w", err)
		}

		err = cmd.Wait()
		if err != nil && it.Err == nil {
			// TODO: Can we log stderr as well here to help diagnose?
			it.Err = fmt.Errorf("error after waiting for subprocess: %w", err)
		}
	}

	return &it, nil
}
