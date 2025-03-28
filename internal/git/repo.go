package git

import (
	"bufio"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type RepoConfigFiles struct {
	MailmapPath    string
	IgnoreRevsPath string
}

func (rf RepoConfigFiles) HasMailmap() bool {
	return len(rf.MailmapPath) > 0
}

func (rf RepoConfigFiles) HasIgnoreRevs() bool {
	return len(rf.IgnoreRevsPath) > 0
}

func (rf RepoConfigFiles) MailmapHash(h hash.Hash32) error {
	if rf.HasMailmap() {
		f, err := os.Open(rf.MailmapPath)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return fmt.Errorf("could not read mailmap file: %v", err)
			}
			defer f.Close()

			_, err = io.Copy(h, f)
			if err != nil {
				return fmt.Errorf("error hashing mailmap file: %v", err)
			}
		}
	}

	return nil
}

// Get git blame ignored revisions
func (rf RepoConfigFiles) IgnoreRevs() (_ []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error reading git blame ignore revs: %w", err)
		}
	}()

	var revs []string

	if !rf.HasIgnoreRevs() {
		return revs, nil
	}

	f, err := os.Open(rf.IgnoreRevsPath)
	if err != nil {
		return revs, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isRev(line) {
			revs = append(revs, line)
		}
	}

	err = scanner.Err()
	if err != nil {
		return revs, err
	}

	return revs, nil
}

// NOTE: We do NOT respect the git config here, we just assume the conventional
// path for this file.
func MailmapPath(gitRootPath string) string {
	path := filepath.Join(gitRootPath, ".mailmap")
	return path
}

// NOTE: We do NOT respect the git config here, we just assume the conventional
// path for this file.
func IgnoreRevsPath(gitRootPath string) string {
	path := filepath.Join(gitRootPath, ".git-blame-ignore-revs")
	return path
}

// Checks to see whether the files exist on disk or not
func CheckRepoConfigFiles(gitRootPath string) (_ RepoConfigFiles, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"error while checking for repository configuration files: %w",
				err,
			)
		}
	}()

	var files RepoConfigFiles

	mailmapPath := MailmapPath(gitRootPath)
	_, err = os.Stat(mailmapPath)
	if err == nil {
		files.MailmapPath = mailmapPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return files, err
	}

	ignoreRevsPath := IgnoreRevsPath(gitRootPath)
	_, err = os.Stat(ignoreRevsPath)
	if err == nil {
		files.IgnoreRevsPath = ignoreRevsPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return files, err
	}

	return files, nil
}
