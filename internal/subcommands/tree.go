package subcommands

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/concurrent"
	"github.com/sinclairtarget/git-who/internal/format"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/git/cmd"
	"github.com/sinclairtarget/git-who/internal/git/config"
	"github.com/sinclairtarget/git-who/internal/pretty"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const defaultMaxDepth = 100

type printTreeOpts struct {
	mode       tally.TallyMode
	maxDepth   int
	showHidden bool
	key        func(t tally.FinalTally) string
}

type treeOutputLine struct {
	indent    string
	path      string
	metric    string
	tally     tally.FinalTally
	showLine  bool
	showTally bool
	dimTally  bool
	dimPath   bool
}

func Tree(
	revs []string,
	pathspecs []string,
	mode tally.TallyMode,
	depth int,
	showEmail bool,
	showHidden bool,
	countMerges bool,
	since string,
	until string,
	authors []string,
	nauthors []string,
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
		"pathspecs",
		pathspecs,
		"mode",
		mode,
		"depth",
		depth,
		"showEmail",
		showEmail,
		"showHidden",
		showHidden,
		"countMerges",
		countMerges,
		"since",
		since,
		"until",
		until,
		"authors",
		authors,
		"nauthors",
		nauthors,
	)

	wtreeset, err := git.WorkingTreeFiles(pathspecs)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	filters := cmd.LogFilters{
		Since:    since,
		Until:    until,
		Authors:  authors,
		Nauthors: nauthors,
	}

	tallyOpts := tally.TallyOpts{Mode: mode, CountMerges: countMerges}
	if showEmail {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorEmail }
	} else {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorName }
	}

	gitRootPath, err := git.GetRoot()
	if err != nil {
		return err
	}

	configFiles, err := config.DetectSupplementalFiles(gitRootPath)
	if err != nil {
		return err
	}

	var root *tally.TreeNode
	if runtime.GOMAXPROCS(0) > 1 {
		root, err = concurrent.TallyCommitsTree(
			ctx,
			revs,
			pathspecs,
			filters,
			configFiles,
			tallyOpts,
			wtreeset,
			gitRootPath,
			cache.GetCache(gitRootPath, configFiles),
			pretty.AllowDynamic(os.Stdout),
		)

		if err == tally.EmptyTreeErr {
			logger().Debug("Tree was empty.")
			return nil
		}

		if err != nil {
			return err
		}
	} else {
		root, err = func() (_ *tally.TreeNode, err error) {
			commits, finish := git.CommitsWithOpts(
				ctx,
				revs,
				pathspecs,
				filters,
				true,
				configFiles,
			)
			defer func() { err = finish() }()

			root, err := tally.TallyCommitsTree(
				commits,
				tallyOpts,
				wtreeset,
				gitRootPath,
			)
			return root, err
		}()

		if err == tally.EmptyTreeErr {
			logger().Debug("Tree was empty.")
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to tally commits: %w", err)
		}
	}

	root = root.Rank(mode)

	maxDepth := depth
	if depth == 0 {
		maxDepth = defaultMaxDepth
	}

	opts := printTreeOpts{
		maxDepth:   maxDepth,
		mode:       mode,
		showHidden: showHidden,
	}
	if showEmail {
		opts.key = func(t tally.FinalTally) string { return t.AuthorEmail }
	} else {
		opts.key = func(t tally.FinalTally) string { return t.AuthorName }
	}

	lines := toLines(root, ".", 0, "", []bool{}, opts, []treeOutputLine{})
	printTree(lines, showEmail)
	return nil
}

// Recursively descend tree, turning tree nodes into output lines.
func toLines(
	node *tally.TreeNode,
	path string,
	depth int,
	lastAuthor string,
	isFinalChild []bool,
	opts printTreeOpts,
	lines []treeOutputLine,
) []treeOutputLine {
	if path == tally.NoDiffPathname {
		return lines
	}

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
	line.showLine = node.InWorkTree || opts.showHidden
	line.dimTally = len(node.Children) > 0
	line.dimPath = !node.InWorkTree

	newAuthor := opts.key(node.Tally) != lastAuthor
	line.showTally = opts.showHidden || newAuthor || len(node.Children) > 0

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

	// Find last non-hidden child
	finalChildIndex := 0
	for i, p := range childPaths {
		child := node.Children[p]
		if child.InWorkTree || opts.showHidden {
			finalChildIndex = i
		}
	}

	for i, p := range childPaths {
		child := node.Children[p]
		lines = toLines(
			child,
			p,
			depth+1,
			opts.key(node.Tally),
			append(isFinalChild, i == finalChildIndex),
			opts,
			lines,
		)
	}

	return lines
}

func fmtTallyMetric(t tally.FinalTally, opts printTreeOpts) string {
	switch opts.mode {
	case tally.CommitMode:
		return fmt.Sprintf("(%s)", format.Number(t.Commits))
	case tally.FilesMode:
		return fmt.Sprintf("(%s)", format.Number(t.FileCount))
	case tally.LinesMode:
		return fmt.Sprintf(
			"(%s%s%s / %s%s%s)",
			pretty.Green(),
			format.Number(t.LinesAdded),
			pretty.DefaultColor(),
			pretty.Red(),
			format.Number(t.LinesRemoved),
			pretty.DefaultColor(),
		)
	case tally.LastModifiedMode:
		return fmt.Sprintf(
			"(%s)",
			format.RelativeTime(progStart, t.LastCommitTime),
		)
	case tally.FirstModifiedMode:
		return fmt.Sprintf(
			"(%s)",
			format.RelativeTime(progStart, t.FirstCommitTime),
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
		if !line.showLine {
			continue
		}

		var path string
		if line.dimPath {
			path = fmt.Sprintf("%s%s%s", pretty.Dim(), line.path, pretty.Reset())
		} else {
			path = line.path
		}

		if !line.showTally {
			fmt.Printf("%s%s\n", line.indent, path)
			continue
		}

		var author string
		if showEmail {
			author = format.Abbrev(format.GitEmail(line.tally.AuthorEmail), 25)
		} else {
			author = format.Abbrev(line.tally.AuthorName, 25)
		}

		indentLen := utf8.RuneCountInString(line.indent)
		pathLen := utf8.RuneCountInString(line.path)
		separator := strings.Repeat(".", tallyStart-indentLen-pathLen)

		if line.dimTally {
			fmt.Printf(
				"%s%s%s%s%s%s %s\n",
				line.indent,
				path,
				pretty.Dim(),
				separator,
				pretty.Reset(),
				author,
				line.metric,
			)
		} else {
			fmt.Printf(
				"%s%s%s%s%s %s%s\n",
				line.indent,
				path,
				pretty.Dim(),
				separator,
				author,
				line.metric,
				pretty.Reset(),
			)
		}
	}
}
