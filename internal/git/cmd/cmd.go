/*
* Handles invoking Git as a subprocess.
 */
package cmd

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

const (
	logFormat        = "--pretty=format:%H%x00%h%x00%p%x00%an%x00%ae%x00%ad%x00"
	mailmapLogFormat = "--pretty=format:%H%x00%h%x00%p%x00%aN%x00%aE%x00%ad%x00"
)

// Runs git log
func RunLog(
	ctx context.Context,
	revs []string,
	pathspecs []string,
	filters LogFilters,
	needDiffs bool,
	useMailmap bool,
) (*Subprocess, error) {
	var baseArgs []string

	if useMailmap {
		baseArgs = []string{
			"log",
			mailmapLogFormat,
			"-z",
			"--date=unix",
			"--reverse",
			"--no-show-signature",
		}
	} else {
		baseArgs = []string{
			"log",
			logFormat,
			"-z",
			"--date=unix",
			"--reverse",
			"--no-show-signature",
			"--no-mailmap",
		}
	}

	if needDiffs {
		baseArgs = append(baseArgs, "--numstat")
	}

	filterArgs := filters.ToArgs()

	var args []string
	if len(pathspecs) > 0 {
		args = slices.Concat(
			baseArgs,
			filterArgs,
			revs,
			[]string{"--"},
			pathspecs,
		)
	} else {
		args = slices.Concat(baseArgs, filterArgs, revs)
	}

	needStdin := false
	subprocess, err := run(ctx, args, needStdin)
	if err != nil {
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	return subprocess, nil
}

// Runs git log --stdin
func RunStdinLog(
	ctx context.Context,
	pathspecs []string, // Doesn't limit commits, but limits diffs!
	needDiffs bool,
	useMailmap bool,
) (*Subprocess, error) {
	var baseArgs []string

	if useMailmap {
		baseArgs = []string{
			"log",
			mailmapLogFormat,
			"-z",
			"--date=unix",
			"--reverse",
			"--no-show-signature",
			"--stdin",
			"--no-walk",
		}
	} else {
		baseArgs = []string{
			"log",
			logFormat,
			"-z",
			"--date=unix",
			"--reverse",
			"--no-show-signature",
			"--stdin",
			"--no-walk",
			"--no-mailmap",
		}
	}

	if needDiffs {
		baseArgs = append(baseArgs, "--numstat")
	}

	var args []string
	if len(pathspecs) > 0 {
		args = slices.Concat(baseArgs, []string{"--"}, pathspecs)
	} else {
		args = baseArgs
	}

	needStdin := true
	subprocess, err := run(ctx, args, needStdin)
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

	needStdin := false
	subprocess, err := run(ctx, slices.Concat(baseArgs, args), needStdin)
	if err != nil {
		return nil, fmt.Errorf("failed to run git rev-parse: %w", err)
	}

	return subprocess, nil
}

func RunRevParseTopLevel(ctx context.Context) (*Subprocess, error) {
	var args = []string{"rev-parse", "--show-toplevel"}

	needStdin := false
	subprocess, err := run(ctx, args, needStdin)
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
	pathspecs []string,
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
	if len(pathspecs) > 0 {
		args = slices.Concat(
			baseArgs,
			filterArgs,
			revs,
			[]string{"--"},
			pathspecs,
		)
	} else {
		args = slices.Concat(baseArgs, filterArgs, revs)
	}

	needStdin := false
	subprocess, err := run(ctx, args, needStdin)
	if err != nil {
		return nil, fmt.Errorf("failed to run git rev-list: %w", err)
	}

	return subprocess, nil
}

func RunLsFiles(ctx context.Context, pathspecs []string) (*Subprocess, error) {
	baseArgs := []string{
		"ls-files",
		"--exclude-standard",
		"-z",
	}

	var args []string
	if len(pathspecs) > 0 {
		args = slices.Concat(baseArgs, pathspecs)
	} else {
		args = slices.Concat(baseArgs, []string{"--"}, pathspecs)
	}

	needStdin := false
	subprocess, err := run(ctx, args, needStdin)
	if err != nil {
		return nil, fmt.Errorf("failed to run git ls-files: %w", err)
	}

	return subprocess, nil
}
