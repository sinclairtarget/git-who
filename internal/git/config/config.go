/*
* Handles reading Git configuration.
 */
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sinclairtarget/git-who/internal/git/cmd"
)

func repoMailmapPath(gitRootPath string) string {
	path := filepath.Join(gitRootPath, ".mailmap")
	return path
}

// Looks up a file pointed to by the mailmap.file setting in the git config.
func globalMailmapPath() (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subprocess, err := cmd.RunConfigGet(
		ctx,
		[]string{"--type=path", "mailmap.file"},
	)
	if err != nil {
		return "", err
	}

	p, err := subprocess.StdoutText()
	if err != nil {
		return "", err
	}

	err = subprocess.Wait()
	if err != nil {
		var subprocessErr *cmd.SubprocessErr
		if errors.As(err, &subprocessErr) {
			logger().Debug(
				"failed to get mailmap path from config or value not present",
				"exitcode",
				subprocessErr.ExitCode,
			)
			p = ""
		} else {
			logger().Debug("got unknown error")
			return "", err
		}
	}

	return p, nil
}

// NOTE: We do NOT respect the blame.ignoreRevsFile option in the git config
// here, we just assume the conventional path for this file in the repo.
//
// The option can be specified multiple times which makes it a tad complicated.
func ignoreRevsPath(gitRootPath string) string {
	path := filepath.Join(gitRootPath, ".git-blame-ignore-revs")
	return path
}

// Checks to see whether the files exist on disk or not
func DetectSupplementalFiles(
	gitRootPath string,
) (_ SupplementalFiles, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"error while checking for supplemental configuration files: %w",
				err,
			)
		}
	}()

	var files SupplementalFiles

	// Repo-local mailmap
	mailmapPath := repoMailmapPath(gitRootPath)
	_, err = os.Stat(mailmapPath)
	if err == nil {
		files.RepoMailmapPath = mailmapPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return files, err
	}

	// Git config mailmap
	mailmapPath, err = globalMailmapPath()
	if err != nil {
		return files, err
	}

	if len(mailmapPath) > 0 {
		_, err = os.Stat(mailmapPath)
		if err == nil {
			files.GlobalMailmapPath = mailmapPath
		} else if !errors.Is(err, os.ErrNotExist) {
			return files, err
		}
	}

	// Repo-local git blame ignore revs file
	ignoreRevsPath := ignoreRevsPath(gitRootPath)
	_, err = os.Stat(ignoreRevsPath)
	if err == nil {
		files.IgnoreRevsPath = ignoreRevsPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return files, err
	}

	return files, nil
}
