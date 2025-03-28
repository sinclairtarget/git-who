package git

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type RepoFiles struct {
	MailmapPath    string
	IgnoreRevsPath string
}

func (rf RepoFiles) HasMailmap() bool {
	return len(rf.MailmapPath) > 0
}

func (rf RepoFiles) HasIgnoreRevs() bool {
	return len(rf.IgnoreRevsPath) > 0
}

// Returns a hash of the files we care about in the repo.
func (rf RepoFiles) Hash() (string, error) {
	h := fnv.New32()

	if rf.HasMailmap() {
		f, err := os.Open(rf.MailmapPath)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return "", fmt.Errorf("could not read mailmap file: %v", err)
			}
			defer f.Close()

			_, err = io.Copy(h, f)
			if err != nil {
				return "", fmt.Errorf("error hashing mailmap file: %v", err)
			}
		}
	}

	if rf.HasIgnoreRevs() {
		f, err := os.Open(rf.IgnoreRevsPath)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return "", fmt.Errorf(
					"could not read ignore revs file: %v",
					err,
				)
			}
			defer f.Close()

			_, err = io.Copy(h, f)
			if err != nil {
				return "", fmt.Errorf("error hashing ignore revs file: %v", err)
			}
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
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
func CheckRepoFiles(gitRootPath string) (_ RepoFiles, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"error while checking for repository configuration files: %w",
				err,
			)
		}
	}()

	var files RepoFiles

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
