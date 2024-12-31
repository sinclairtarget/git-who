package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/sinclairtarget/git-who/internal/ansi"
	"github.com/sinclairtarget/git-who/internal/format"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const defaultMaxDepth = 100

type printTreeOpts struct {
	mode     tally.TallyMode
	maxDepth int
}

type treeOutputLine struct {
	indent    string
	path      string
	metric    string
	tally     tally.Tally
	isDir     bool
	showTally bool
}

func tree(
	revs []string,
	paths []string,
	mode tally.TallyMode,
	depth int,
	showEmail bool,
	since string,
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
		"showEmail",
		showEmail,
		"since",
		since,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	commitOpts := git.CommitOpts{Since: since, PopulateDiffs: true}
	commits, closer, err := git.CommitsWithOpts(ctx, revs, paths, commitOpts)
	if err != nil {
		return err
	}

	tallyOpts := tally.TallyOpts{Mode: mode}
	if showEmail {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorEmail }
	} else {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorName }
	}

	root, err := tally.TallyCommitsByPath(commits, tallyOpts)
	if err != nil {
		return fmt.Errorf("failed to tally commits: %w", err)
	}

	err = closer()
	if err != nil {
		return err
	}

	maxDepth := depth
	if depth == 0 {
		maxDepth = defaultMaxDepth
	}

	opts := printTreeOpts{
		maxDepth: maxDepth,
		mode:     mode,
	}
	lines := toLines(root, ".", 0, "", []bool{}, opts, []treeOutputLine{})
	printTree(lines, showEmail)
	return nil
}

func toLines(
	node *tally.TreeNode,
	path string,
	depth int,
	lastAuthor string,
	isFinalChild []bool,
	opts printTreeOpts,
	lines []treeOutputLine,
) []treeOutputLine {
	if depth > opts.maxDepth {
		return lines
	}

	if depth < opts.maxDepth && len(node.Children) == 1 {
		// Path ellision
		for k, v := range node.Children {
			lines = toLines(
				v,
				filepath.Join(path, k),
				depth+1,
				lastAuthor,
				isFinalChild,
				opts,
				lines,
			)
		}
		return lines
	}

	var line treeOutputLine

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
	line.indent = indentBuilder.String()

	line.path = path
	if len(node.Children) > 0 {
		// Have a directory
		line.path = path + string(os.PathSeparator)
	}

	line.tally = node.Tally
	line.metric = fmtTallyMetric(node.Tally, opts)
	line.isDir = len(node.Children) > 0
	line.showTally = node.Tally.AuthorEmail != lastAuthor

	lines = append(lines, line)

	childPaths := slices.SortedFunc(
		maps.Keys(node.Children),
		func(a, b string) int {
			// Show directories first
			aHasChildren := len(node.Children[a].Children) > 0
			bHasChildren := len(node.Children[b].Children) > 0

			if aHasChildren == bHasChildren {
				return strings.Compare(a, b) // Sort alphabetically
			} else if aHasChildren {
				return -1
			} else {
				return 1
			}
		},
	)

	for i, p := range childPaths {
		child := node.Children[p]
		lines = toLines(
			child,
			p,
			depth+1,
			node.Tally.AuthorEmail,
			append(isFinalChild, i == len(childPaths)-1),
			opts,
			lines,
		)
	}

	return lines
}

func fmtTallyMetric(t tally.Tally, opts printTreeOpts) string {
	switch opts.mode {
	case tally.CommitMode:
		return fmt.Sprintf("(%d)", t.Commits)
	case tally.FilesMode:
		return fmt.Sprintf("(%d)", t.FileCount)
	case tally.LinesMode:
		return fmt.Sprintf(
			"(%s%d%s / %s%d%s)",
			ansi.Green,
			t.LinesAdded,
			ansi.DefaultColor,
			ansi.Red,
			t.LinesRemoved,
			ansi.DefaultColor,
		)
	case tally.LastModifiedMode:
		return fmt.Sprintf(
			"(%s)",
			format.RelativeTime(progStart, t.LastCommitTime),
		)
	default:
		panic("unrecognized mode in switch")
	}
}

func printTree(lines []treeOutputLine, showEmail bool) {
	longest := 0
	for _, line := range lines {
		indentLen := utf8.RuneCountInString(line.indent)
		pathLen := utf8.RuneCountInString(line.path)
		if indentLen+pathLen > longest {
			longest = indentLen + pathLen
		}
	}

	tallyStart := longest + 4 // Use at least 4 "." to separate path from tally

	for _, line := range lines {
		if !line.showTally {
			fmt.Printf("%s%s\n", line.indent, line.path)
			continue
		}

		indentLen := utf8.RuneCountInString(line.indent)
		pathLen := utf8.RuneCountInString(line.path)

		separator := strings.Repeat(".", tallyStart-indentLen-pathLen)

		var author string
		if showEmail {
			author = format.Abbrev(format.GitEmail(line.tally.AuthorEmail), 25)
		} else {
			author = format.Abbrev(line.tally.AuthorName, 25)
		}

		if line.isDir {
			fmt.Printf(
				"%s%s%s%s%s%s %s\n",
				line.indent,
				line.path,
				ansi.Dim,
				separator,
				ansi.Reset,
				author,
				line.metric,
			)
		} else {
			fmt.Printf(
				"%s%s%s%s%s %s%s\n",
				line.indent,
				line.path,
				ansi.Dim,
				separator,
				author,
				line.metric,
				ansi.Reset,
			)
		}
	}
}
