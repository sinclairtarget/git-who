package git

import (
	"bufio"
	"fmt"
	"io"
	"iter"
	"os/exec"
	"slices"
)

type Subprocess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

// Returns a single-use iterator over the output of the command, line by line.
func (s Subprocess) StdoutLines() iter.Seq2[string, error] {
	scanner := bufio.NewScanner(s.stdout)

	return func(yield func(string, error) bool) {
		for scanner.Scan() {
			if !yield(scanner.Text(), nil) {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			yield("", fmt.Errorf("error while scanning: %w", err))
		}
	}
}

func (s Subprocess) Wait() error {
	err := s.cmd.Wait()
	if err != nil {
		// TODO: Can we log stderr as well here to help diagnose?
		return fmt.Errorf("error after waiting for subprocess: %w", err)
	}

	logger().Debug(
		"subprocess exited",
		"code",
		s.cmd.ProcessState.ExitCode(),
	)
	return nil
}

func Run(args []string) (*Subprocess, error) {
	cmd := exec.Command("git", args...)
	logger().Debug("running subprocess", "cmd", cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start subprocess: %w", err)
	}

	return &Subprocess{
		cmd:    cmd,
		stdout: stdout,
	}, nil
}

// Runs git log
func RunLog(revs []string, path string) (*Subprocess, error) {
	var baseArgs = []string{
		"log",
		"--pretty=format:%H%n%h%n%an%n%ae%n%ad%n%s",
		"--numstat",
		"--date=unix",
	}
	args := slices.Concat(baseArgs, revs, []string{path})

	subprocess, err := Run(args)
	if err != nil {
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	return subprocess, nil
}
