package git

import (
	"strings"
)

func isSupportedPathspec(pathspec string) bool {
	return true
}

func pathspecMatch(path string, pathspec string) bool {
	// TODO: Implement SOME of Git's pathspec magic
	return strings.HasPrefix(path, pathspec)
}
