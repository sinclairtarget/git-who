package main

import (
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
	mode     tally.TallyMode
	maxDepth int
	maxWidth int
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

	maxWidth := calcMaxWidth(root, ".", 0, maxDepth, 0)
	opts := printTreeOpts{
		maxDepth: maxDepth,
		maxWidth: maxWidth,
		mode:     mode,
	}

	printTree(root, ".", 0, "", []bool{}, opts)
	return nil
}

func calcMaxWidth(
	node *tally.TreeNode,
	path string,
	depth int,
	maxDepth int,
	indent int,
) int {
	if depth > maxDepth {
		return 0
	}

	widthThisNode := 4*indent + len(path)
	max := widthThisNode

	if depth < maxDepth && len(node.Children) == 1 {
		// Path ellision
		for p, child := range node.Children {
			childWidth := calcMaxWidth(
				child,
				filepath.Join(path, p),
				depth+1,
				maxDepth,
				indent,
			)
			if childWidth > max {
				max = childWidth
			}
		}
	} else {
		for p, child := range node.Children {
			childWidth := calcMaxWidth(
				child,
				p,
				depth+1,
				maxDepth,
				indent+1,
			)
			if childWidth > max {
				max = childWidth
			}
		}
	}

	return max
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

	var indentBuilder strings.Builder
	for i, isFinal := range isFinalChild {
		if i < len(isFinalChild)-1 {
			if isFinal {
				fmt.Fprintf(&indentBuilder, "    ")
			} else {
				fmt.Fprintf(&indentBuilder, "│   ")
			}
		} else {
			if isFinal {
				fmt.Fprintf(&indentBuilder, "└── ")
			} else {
				fmt.Fprintf(&indentBuilder, "├── ")
			}
		}
	}

	pathPart := path
	if len(node.Children) > 0 {
		// Have a directory
		pathPart = path + string(os.PathSeparator)
	}

	var tallyPart string
	var separator string
	if node.Tally.AuthorEmail != lastAuthor {
		tallyPart = fmtTally(node.Tally, opts.mode)
		separator = strings.Repeat(
			".",
			max(2, opts.maxWidth+2-len(isFinalChild)*4-len(pathPart)),
		)
	}

	if len(node.Children) > 0 {
		fmt.Printf(
			"%s%s%s%s%s%s\n",
			indentBuilder.String(),
			pathPart,
			ansi.Dim,
			separator,
			ansi.Reset,
			tallyPart,
		)
	} else {
		fmt.Printf(
			"%s%s%s%s%s%s\n",
			indentBuilder.String(),
			pathPart,
			ansi.Dim,
			separator,
			tallyPart,
			ansi.Reset,
		)
	}

	childPaths := slices.Sorted(maps.Keys(node.Children))
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
		return fmt.Sprintf("%-15s (%d)", t.AuthorName, t.Commits)
	case tally.FilesMode:
		return fmt.Sprintf("%-15s (%d)", t.AuthorName, t.FileCount)
	case tally.LinesMode:
		return fmt.Sprintf(
			"%-15s (%s%d%s / %s%d%s)",
			t.AuthorName,
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
