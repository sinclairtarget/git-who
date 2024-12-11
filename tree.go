package main

import (
	"fmt"

	"github.com/sinclairtarget/git-who/internal/tally"
)

func tree(revs []string, path string, mode tally.TallyMode, depth int) error {
	fmt.Printf("tree() revs: %v, path: %s, mode: %v, depth: %d\n",
		revs,
		path,
		mode,
		depth)
	return nil
}
