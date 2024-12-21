package main

import (
	"github.com/sinclairtarget/git-who/internal/tally"
)

func tree(revs []string, paths []string, mode tally.TallyMode, depth int) error {
	logger().Debug(
		"called tree()",
		"revs",
		revs,
		"paths",
		paths,
		"mode",
		mode,
		"depth",
		depth,
	)
	return nil
}
