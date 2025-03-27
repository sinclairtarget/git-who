package git

import (
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
