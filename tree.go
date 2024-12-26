package main

import (
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

func tree(
	revs []string,
	paths []string,
	mode tally.TallyMode,
	depth int,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running \"tree\": %w", err)
		}
	}()

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

	root, err := func() (_ *tally.TreeNode, err error) {
		commits, closer, err := git.Commits(revs, paths)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err == nil {
				err = closer()
			}
		}()

		root, err := tally.TallyCommitsByPath(commits, mode)
		if err != nil {
			return nil, err
		}

		return root, nil
	}()
	if err != nil {
		return fmt.Errorf("failed to tally commits: %w", err)
	}

	fmt.Println(root)
	return nil
}
