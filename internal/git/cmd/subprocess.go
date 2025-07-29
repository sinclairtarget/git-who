package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"os/exec"
	"strings"
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
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (s Subprocess) StdinWriter() (_ *bufio.Writer, closer func() error) {
	return bufio.NewWriter(s.stdin), func() error {
		return s.stdin.Close()
	}
}

func (s Subprocess) StdoutText() (string, error) {
	b, err := io.ReadAll(s.stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(b)), nil
}

// Returns a single-use iterator over the output of the command, line by line.
func (s Subprocess) StdoutLines() (iter.Seq[string], func() error) {
	var iterErr error

	seq := func(yield func(string) bool) {
		scanner := bufio.NewScanner(s.stdout)
		for scanner.Scan() {
			if !yield(scanner.Text()) {
				return
			}
		}

		iterErr = scanner.Err()
	}

	finish := func() error {
		if iterErr != nil {
			iterErr = fmt.Errorf("error while scanning: %w", iterErr)
		}

		return iterErr
	}

	return seq, finish
}

// Returns a single-use iterator over the output from git log.
//
// Lines are split on NULLs with some additional processing.
func (s Subprocess) StdoutNullDelimitedLines() (
	iter.Seq[string],
	func() error,
) {
	var iterErr error

	seq := func(yield func(string) bool) {
		scanner := bufio.NewScanner(s.stdout)

		scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
			null_i := bytes.IndexByte(data, '\x00')

			if null_i >= 0 {
				return null_i + 1, data[:null_i], nil
			}

			if atEOF {
				return 0, data, bufio.ErrFinalToken
			}

			return 0, nil, nil // Scan more
		})

		for scanner.Scan() {
			line := scanner.Text()

			// Handle annoying new line that exists between regular commit
			// fields and --numstat data
			processedLine := strings.TrimPrefix(line, "\n")

			if !yield(processedLine) {
				return
			}
		}

		iterErr = scanner.Err()
	}

	finish := func() error {
		if iterErr != nil {
			iterErr = fmt.Errorf("error while scanning: %w", iterErr)
		}

		return iterErr
	}

	return seq, finish
}

func (s Subprocess) Wait() error {
	logger().Debug("waiting for subprocess...")

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
			Stderr:   strings.TrimSpace(string(stderr)),
			Err:      err,
		}
	}

	return nil
}

func run(
	ctx context.Context,
	args []string,
	needStdin bool,
) (*Subprocess, error) {
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

	var stdin io.WriteCloser
	if needStdin {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to open stdin pipe: %w", err)
		}
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start subprocess: %w", err)
	}

	return &Subprocess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}, nil
}
