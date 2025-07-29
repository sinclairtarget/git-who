/*
* Handles reading Git configuration.
 */
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
func DetectSupplementalFiles(gitRootPath string) (_ SupplementalFiles, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"error while checking for repository configuration files: %w",
				err,
			)
		}
	}()

	var files SupplementalFiles

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
