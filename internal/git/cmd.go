package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"os/exec"
	"slices"
)

type SubprocessErr struct {
	ExitCode int
	Stderr   string
	Err      error
}

func (err SubprocessErr) Error() string {
	if err.Stderr != "" {
		return fmt.Sprintf(
			"Git subprocess exited with code %d. Error output:\n%s",
			err.ExitCode,
			err.Stderr,
		)
	}

	return fmt.Sprintf("Git subprocess exited with code %d", err.ExitCode)
}

func (err SubprocessErr) Unwrap() error {
	return err.Err
}

type Subprocess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
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
	logger().Debug("Waiting for subprocess...")

	stderr, err := io.ReadAll(s.stderr)
	if err != nil {
		return fmt.Errorf("could not read stderr: %w", err)
	}

	err = s.cmd.Wait()
	logger().Debug(
		"subprocess exited",
		"code",
		s.cmd.ProcessState.ExitCode(),
	)

	if err != nil {
		return SubprocessErr{
			ExitCode: s.cmd.ProcessState.ExitCode(),
			Stderr:   string(stderr),
			Err:      err,
		}
	}

	return nil
}

func Run(ctx context.Context, args []string) (*Subprocess, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	logger().Debug("running subprocess", "cmd", cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stderr pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start subprocess: %w", err)
	}

	return &Subprocess{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// Runs git log
func RunLog(
	ctx context.Context,
	revs []string,
	paths []string,
	since string,
) (*Subprocess, error) {
	var baseArgs = []string{
		"log",
		"--pretty=format:%H%n%h%n%an%n%ae%n%ad%n%s",
		"--numstat",
		"--summary",
		"--date=unix",
		"--no-merges", // Ensures every commit has file diffs
		"--reverse",   // Needed to handle file renaming
	}

	var args []string
	if since != "" {
		args = slices.Concat(
			baseArgs,
			[]string{"--since", since},
			revs,
			[]string{"--"},
			paths,
		)
	} else {
		args = slices.Concat(baseArgs, revs, []string{"--"}, paths)
	}

	subprocess, err := Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	return subprocess, nil
}

// Runs git rev-parse
func RunRevParse(ctx context.Context, args []string) (*Subprocess, error) {
	var baseArgs = []string{
		"rev-parse",
		"--no-flags",
	}

	subprocess, err := Run(ctx, slices.Concat(baseArgs, args))
	if err != nil {
		return nil, fmt.Errorf("failed to run git rev-parse: %w", err)
	}

	return subprocess, nil
}
