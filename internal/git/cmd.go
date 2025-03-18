package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os/exec"
	"slices"
	"strings"
)

const (
	logFormat     = "--pretty=format:%H%n%h%n%p%n%aN%n%aE%n%ad%n%s%n" // newline
	logDiffFormat = "--pretty=format:%H%n%h%n%p%n%aN%n%aE%n%ad%n%s"
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

// Returns a single-use iterator over the output of the command.
//
// Lines are split on both newlines and NULLs.
func (s Subprocess) StdoutLogLines() iter.Seq2[string, error] {
	scanner := bufio.NewScanner(s.stdout)

	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		null_i := bytes.IndexByte(data, '\x00')
		newline_i := bytes.IndexByte(data, '\n')

		if null_i >= 0 && newline_i >= 0 {
			i := min(null_i, newline_i)
			return i + 1, data[:i], nil
		} else if newline_i > 0 {
			return newline_i + 1, data[:newline_i], nil
		} else if null_i > 0 {
			return null_i + 1, data[:null_i], nil
		}

		if atEOF {
			return 0, data, bufio.ErrFinalToken
		}

		return 0, nil, nil // Scan more
	})

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

type LogFilters struct {
	Since    string
	Until    string
	Authors  []string
	Nauthors []string
}

// Turn into CLI args we can pass to `git log`
func (f LogFilters) ToArgs() []string {
	args := []string{}

	if f.Since != "" {
		args = append(args, "--since", f.Since)
	}

	if f.Until != "" {
		args = append(args, "--until", f.Until)
	}

	for _, author := range f.Authors {
		args = append(args, "--author", author)
	}

	if len(f.Nauthors) > 0 {
		args = append(args, "--perl-regexp")

		// Build regex pattern OR-ing together all the nauthors
		var b strings.Builder
		for i, nauthor := range f.Nauthors {
			b.WriteString(nauthor)
			if i < len(f.Nauthors)-1 {
				b.WriteString("|")
			}
		}

		regex := fmt.Sprintf(`^((?!%s).*)$`, b.String())
		args = append(args, "--author", regex)
	}

	return args
}

// Runs git log
func RunLog(
	ctx context.Context,
	revs []string,
	paths []string,
	filters LogFilters,
	needDiffs bool,
) (*Subprocess, error) {
	var baseArgs []string
	if needDiffs {
		baseArgs = []string{
			"log",
			logDiffFormat,
			"-z",
			"--date=unix",
			"--reverse",
			"--numstat",
			"--diff-merges=first-parent",
		}
	} else {
		// Runs git log without --numstat, which is much faster.
		baseArgs = []string{
			"log",
			logFormat,
			"-z",
			"--date=unix",
			"--reverse",
		}
	}

	filterArgs := filters.ToArgs()

	var args []string
	if len(paths) > 0 {
		args = slices.Concat(baseArgs, filterArgs, revs, []string{"--"}, paths)
	} else {
		args = slices.Concat(baseArgs, filterArgs, revs)
	}

	subprocess, err := run(ctx, args, false)
	if err != nil {
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	return subprocess, nil
}

// Runs git log --stdin
func RunStdinLog(
	ctx context.Context,
	paths []string, // Doesn't limit commits, but limits diffs!
	needDiffs bool,
) (*Subprocess, error) {
	var baseArgs []string
	if needDiffs {
		baseArgs = []string{
			"log",
			logDiffFormat,
			"-z",
			"--date=unix",
			"--stdin",
			"--no-walk",
			"--reverse",
			"--numstat",
			"--diff-merges=first-parent",
		}
	} else {
		// Runs git log without --numstat, which is much faster.
		baseArgs = []string{
			"log",
			logFormat,
			"-z",
			"--date=unix",
			"--stdin",
			"--no-walk",
			"--reverse",
		}
	}

	var args []string
	if len(paths) > 0 {
		args = slices.Concat(baseArgs, []string{"--"}, paths)
	} else {
		args = baseArgs
	}

	subprocess, err := run(ctx, args, true)
	if err != nil {
		return nil, fmt.Errorf("error running git log --stdin: %w", err)
	}

	return subprocess, nil
}

// Runs git rev-parse
func RunRevParse(ctx context.Context, args []string) (*Subprocess, error) {
	var baseArgs = []string{
		"rev-parse",
		"--no-flags",
	}

	subprocess, err := run(ctx, slices.Concat(baseArgs, args), false)
	if err != nil {
		return nil, fmt.Errorf("failed to run git rev-parse: %w", err)
	}

	return subprocess, nil
}

// Runs git rev-list. When countOnly is true, passes --count, which is much
// faster than printing then getting all the revisions when all you need is the
// count.
func RunRevList(
	ctx context.Context,
	revs []string,
	paths []string,
	filters LogFilters,
) (*Subprocess, error) {
	if len(revs) == 0 {
		return nil, errors.New("git rev-list requires revision spec")
	}

	baseArgs := []string{
		"rev-list",
		"--reverse",
	}

	filterArgs := filters.ToArgs()

	var args []string
	if len(paths) > 0 {
		args = slices.Concat(baseArgs, filterArgs, revs, []string{"--"}, paths)
	} else {
		args = slices.Concat(baseArgs, filterArgs, revs)
	}

	subprocess, err := run(ctx, args, false)
	if err != nil {
		return nil, fmt.Errorf("failed to run git rev-list: %w", err)
	}

	return subprocess, nil
}

func RunLsFiles(ctx context.Context, paths []string) (*Subprocess, error) {
	baseArgs := []string{"ls-files", "--exclude-standard"}

	var args []string
	if len(paths) > 0 {
		args = slices.Concat(baseArgs, paths)
	} else {
		args = slices.Concat(baseArgs, []string{"--"}, paths)
	}

	subprocess, err := run(ctx, args, false)
	if err != nil {
		return nil, fmt.Errorf("failed to run git ls-files: %w", err)
	}

	return subprocess, nil
}
