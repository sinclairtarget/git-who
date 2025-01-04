package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"os/exec"
	"slices"
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

type LogFilters struct {
	Since    string
	Authors  []string
	Nauthors []string
}

// Turn into CLI args we can pass to `git log`
func (f LogFilters) ToArgs() []string {
	args := []string{}

	if f.Since != "" {
		args = append(args, "--since", f.Since)
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

	filterArgs := filters.ToArgs()

	var args []string
	if len(paths) > 0 {
		args = slices.Concat(baseArgs, filterArgs, revs, []string{"--"}, paths)
	} else {
		args = slices.Concat(baseArgs, filterArgs, revs)
	}

	subprocess, err := Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	return subprocess, nil
}

// Runs git log without --numstat or --summary, which is much faster.
func RunShortLog(
	ctx context.Context,
	revs []string,
	paths []string,
	filters LogFilters,
) (*Subprocess, error) {
	var baseArgs = []string{
		"log",
		"--pretty=format:%H%n%h%n%an%n%ae%n%ad%n%s%n",
		"--date=unix",
		"--no-merges", // Ensures every commit has file diffs
		"--reverse",   // Needed to handle file renaming
	}

	filterArgs := filters.ToArgs()

	var args []string
	if len(paths) > 0 {
		args = slices.Concat(baseArgs, filterArgs, revs, []string{"--"}, paths)
	} else {
		args = slices.Concat(baseArgs, filterArgs, revs)
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

// Runs git rev-list. When countOnly is true, passes --count, which is much
// faster than printing then getting all the revisions when all you need is the
// count.
func RunRevList(
	ctx context.Context,
	revs []string,
	paths []string,
	filters LogFilters,
	countOnly bool,
) (*Subprocess, error) {
	var baseArgs []string
	if countOnly {
		baseArgs = []string{"rev-list", "--count"}
	} else {
		baseArgs = []string{"rev-list"}
	}

	filterArgs := filters.ToArgs()

	var args []string
	if len(paths) > 0 {
		args = slices.Concat(baseArgs, filterArgs, revs, []string{"--"}, paths)
	} else {
		args = slices.Concat(baseArgs, filterArgs, revs)
	}

	subprocess, err := Run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to run git rev-list: %w", err)
	}

	return subprocess, nil
}
