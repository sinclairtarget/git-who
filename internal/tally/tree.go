package tally

import (
	"fmt"
	"iter"
	"os"
	"strings"

	"github.com/sinclairtarget/git-who/internal/git"
)

// A file tree of edits to the repo
type TreeNode struct {
	Tally         Tally // Winning tally
	Children      map[string]*TreeNode
	intermediates map[string]intermediateTally
}

func newNode() *TreeNode {
	return &TreeNode{
		Children:      map[string]*TreeNode{},
		intermediates: map[string]intermediateTally{},
	}
}

func (t *TreeNode) String() string {
	return fmt.Sprintf("{ %d }", len(t.intermediates))
}

// Stores per-path tallies for a single author
type authorPaths struct {
	name  string
	email string
	paths map[string]intermediateTally
}

// Tally metrics per author per path
func tallyByPaths(
	commits iter.Seq2[git.Commit, error],
	wtreefiles map[string]bool,
	opts TallyOpts,
) (map[string]authorPaths, error) {
	authors := map[string]authorPaths{}

	// Tally over commits
	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		key := opts.Key(commit)
		author, ok := authors[key]
		if !ok {
			author = authorPaths{
				name:  commit.AuthorName,
				email: commit.AuthorEmail,
				paths: map[string]intermediateTally{},
			}
			authors[key] = author
		}

		for _, diff := range commit.FileDiffs {
			if !commit.IsMerge {
				pathTally, ok := author.paths[diff.Path]
				if !ok {
					pathTally = newTally(1)
				}

				pathTally.commitset[commit.ShortHash] = true
				pathTally.added += diff.LinesAdded
				pathTally.removed += diff.LinesRemoved
				pathTally.lastCommitTime = commit.Date

				author.paths[diff.Path] = pathTally
			}

			// If file move would create a file in the working tree, move tally
			// to that path, potentially overwriting.
			destInWTree := wtreefiles[diff.MoveDest]
			if destInWTree {
				for key, _ := range authors {
					pathTally, ok := authors[key].paths[diff.Path]
					if ok {
						delete(authors[key].paths, diff.Path)
						authors[key].paths[diff.MoveDest] = pathTally
					}
				}
			}
		}
	}

	return authors, nil
}

// Splits path into first dir and remainder.
func splitPath(path string) (string, string) {
	dir, subpath, found := strings.Cut(path, string(os.PathSeparator))
	if !found {
		return path, ""
	}

	return dir, subpath
}

// Inserts an intermediate tally into the tally tree.
func (t *TreeNode) insert(path string, key string, tally intermediateTally) {
	if path == "" {
		// Leaf
		t.intermediates[key] = tally
		return
	}

	// Insert child
	p, nextP := splitPath(path)
	child, ok := t.Children[p]
	if !ok {
		t.Children[p] = newNode()
		child = t.Children[p]
	}

	child.insert(nextP, key, tally)
}

func (t *TreeNode) tally(
	authors map[string]authorPaths,
	mode TallyMode,
) *TreeNode {
	for p, child := range t.Children {
		t.Children[p] = child.tally(authors, mode)
	}

	authorTallies := map[string]Tally{}

	for key, author := range authors {
		authorTally := authorTallies[key]
		authorTally.AuthorName = author.name
		authorTally.AuthorEmail = author.email

		var intermediate intermediateTally
		if len(t.Children) > 0 {
			intermediate = newTally(0)
			for _, child := range t.Children {
				childIntermediate, ok := child.intermediates[key]
				if ok {
					intermediate = intermediate.Add(childIntermediate)
				}
			}
			if intermediate.numTallied == 0 {
				continue // Author didn't edit any children
			}

			t.intermediates[key] = intermediate
		} else {
			var ok bool
			intermediate, ok = t.intermediates[key]
			if !ok {
				continue
			}
		}

		authorTally.Commits = intermediate.Commits()
		authorTally.LinesAdded = intermediate.added
		authorTally.LinesRemoved = intermediate.removed
		authorTally.LastCommitTime = intermediate.lastCommitTime
		authorTally.FileCount = intermediate.numTallied

		authorTallies[key] = authorTally
	}

	// Pick best tally for the node according to the tally mode
	sorted := sortTallies(authorTallies, mode)
	t.Tally = sorted[0]
	return t
}

/*
* TallyCommitsTree() returns a tree of nodes mirroring the working directory
* with a tally for each node.
 */
func TallyCommitsTree(
	commits iter.Seq2[git.Commit, error],
	wtreefiles map[string]bool,
	opts TallyOpts,
) (*TreeNode, error) {
	root := newNode()

	authors, err := tallyByPaths(commits, wtreefiles, opts)
	if err != nil {
		return root, err
	}

	for key, author := range authors {
		for path, pathTally := range author.paths {
			if inWTree := wtreefiles[path]; inWTree {
				root.insert(path, key, pathTally)
			}
		}
	}

	return root.tally(authors, opts.Mode), nil
}
