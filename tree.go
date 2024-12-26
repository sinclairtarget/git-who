package main

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinclairtarget/git-who/internal/ansi"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const defaultMaxDepth = 100

type printTreeOpts struct {
	mode      tally.TallyMode
	maxDepth  int
	pathWidth int
}

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

	maxDepth := depth
	if depth == 0 {
		maxDepth = defaultMaxDepth
	}

	opts := printTreeOpts{
		maxDepth:  maxDepth,
		mode:      mode,
		pathWidth: 1,
	}

	printTree(root, ".", 0, "", []bool{}, opts)
	return nil
}

func printTree(
	node *tally.TreeNode,
	path string,
	depth int,
	lastAuthor string,
	isFinalChild []bool,
	opts printTreeOpts,
) {
	if depth > opts.maxDepth {
		return
	}

	if depth < opts.maxDepth && len(node.Children) == 1 {
		// Path ellision
		for k, v := range node.Children {
			printTree(
				v,
				filepath.Join(path, k),
				depth+1,
				lastAuthor,
				isFinalChild,
				opts,
			)
		}
		return
	}

	var indent strings.Builder
	for i, isFinal := range isFinalChild {
		if i < len(isFinalChild)-1 {
			if isFinal {
				fmt.Fprintf(&indent, "    ")
			} else {
				fmt.Fprintf(&indent, "│   ")
			}
		} else {
			if isFinal {
				fmt.Fprintf(&indent, "└── ")
			} else {
				fmt.Fprintf(&indent, "├── ")
			}
		}
	}

	pathPart := path
	if len(node.Children) > 0 {
		// Have a directory
		pathPart = path + string(os.PathSeparator)
	}

	var tallyPart string
	if node.Tally.AuthorEmail != lastAuthor {
		tallyPart = fmtTally(node.Tally, opts.mode)
	}

	fmt.Printf(
		"%s%-*s  %s%s%s\n",
		indent.String(),
		opts.pathWidth,
		pathPart,
		ansi.Dim,
		tallyPart,
		ansi.Reset,
	)

	childPaths := slices.Sorted(maps.Keys(node.Children))
	if len(childPaths) > 0 {
		opts.pathWidth = len(slices.MaxFunc(childPaths, func(a, b string) int {
			return cmp.Compare(len(a), len(b))
		}))
	}

	for i, p := range childPaths {
		child := node.Children[p]
		printTree(
			child,
			p,
			depth+1,
			node.Tally.AuthorEmail,
			append(isFinalChild, i == len(childPaths)-1),
			opts,
		)
	}
}

func fmtTally(t tally.Tally, mode tally.TallyMode) string {
	switch mode {
	case tally.CommitMode:
		return fmt.Sprintf("%s (%d)", t.AuthorEmail, t.Commits)
	case tally.FilesMode:
		return fmt.Sprintf("%s (%d)", t.AuthorEmail, t.FileCount)
	case tally.LinesMode:
		return fmt.Sprintf(
			"%s (%s%d%s / %s%d%s)",
			t.AuthorEmail,
			ansi.Green,
			t.LinesAdded,
			ansi.DefaultColor,
			ansi.Red,
			t.LinesRemoved,
			ansi.DefaultColor,
		)
	default:
		panic("unrecognized mode in switch")
	}
}
