package main

import (
	"github.com/sinclairtarget/git-who/internal/tally"
)

func tree(revs []string, path string, mode tally.TallyMode, depth int) error {
	logger().Debug(
		"called tree()",
		"revs",
		revs,
		"path",
		path,
		"mode",
		mode,
		"depth",
		depth,
	)
	return nil
}
