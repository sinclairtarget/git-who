package tally

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sinclairtarget/git-who/internal/git"
)

type TreeNode struct {
	Tally          Tally
	Children       map[string]*TreeNode
	tallies        map[string]Tally
	lastCommitSeen string
	isFile         bool // A file, rather than a directory
}

func newNode(isFile bool) *TreeNode {
	return &TreeNode{
		Children: map[string]*TreeNode{},
		tallies:  map[string]Tally{},
		isFile:   isFile,
	}
}

func (t *TreeNode) String() string {
	return fmt.Sprintf("{ %d }", len(t.tallies))
}

func cleanPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path cannot be empty")
	}

	if filepath.IsAbs(path) {
		return "", errors.New("path cannot be absolute")
	}

	return filepath.Clean(path), nil
}

// Splits path into first dir and remainder.
func splitPath(path string) (string, string) {
	dir, subpath, found := strings.Cut(path, string(os.PathSeparator))
	if !found {
		return path, ""
	}

	return dir, subpath
}

// Inserts an edit into the tally tree.
func (t *TreeNode) edit(
	path string,
	commit git.Commit,
	diff git.FileDiff,
	opts TallyOpts,
) {
	if path != "" {
		// Insert child
		p, nextP := splitPath(path)
		child, ok := t.Children[p]
		if !ok {
			t.Children[p] = newNode(nextP == "")
			child = t.Children[p]
		}

		child.edit(nextP, commit, diff, opts)
	}

	// Add tally
	key := opts.Key(commit)
	nodeTally, ok := t.tallies[key]
	if !ok {
		nodeTally = Tally{
			AuthorName:  commit.AuthorName,
			AuthorEmail: commit.AuthorEmail,
		}
	}

	nodeTally.LinesAdded += diff.LinesAdded
	nodeTally.LinesRemoved += diff.LinesRemoved

	if commit.Hash != t.lastCommitSeen {
		nodeTally.Commits += 1
		nodeTally.LastCommitTime = commit.Date
		t.lastCommitSeen = commit.Hash
	}

	nodeTally.FileCount = 0
	if len(t.Children) > 0 {
		for _, child := range t.Children {
			childTally, ok := child.tallies[key]
			if ok {
				nodeTally.FileCount += childTally.FileCount
			}
		}
	} else {
		nodeTally.FileCount = 1
	}

	t.tallies[key] = nodeTally

	// Pick best tally for the node according to the tally mode
	sorted := slices.SortedFunc(maps.Values(t.tallies), func(a, b Tally) int {
		return -a.Compare(b, opts.Mode)
	})
	t.Tally = sorted[0]
}

func (t *TreeNode) insert(path string, node *TreeNode) error {
	// Find parent
	cur := t
	var p string
	nextP := path

	for {
		p, nextP = splitPath(nextP)
		if nextP == "" {
			break
		}

		child, ok := cur.Children[p]
		if !ok {
			// Need to create interior node
			cur.Children[p] = newNode(false)
			child = cur.Children[p]
		}
		cur = child
	}

	_, ok := cur.Children[p]
	if ok {
		return fmt.Errorf("path already exists in tree: \"%s\"", path)
	}

	cur.Children[p] = node
	return nil
}

func (t *TreeNode) remove(path string) (*TreeNode, error) {
	// Find parent of target node
	cur := t
	var p string
	nextP := path

	for {
		p, nextP = splitPath(nextP)
		if nextP == "" {
			break
		}

		var ok bool
		cur, ok = cur.Children[p]
		if !ok {
			return nil, fmt.Errorf(
				"could not find existing node for path \"%s\"",
				path,
			)
		}
	}

	// Remove child node from children map
	child := cur.Children[p]
	delete(cur.Children, p)
	return child, nil
}

/*
* Prunes the following types of nodes from the tree:
*
* 1. Interior nodes (directories) with no children.
*
* Returns true if this node needs pruning.
 */
func (t *TreeNode) prune() bool {
	if t.isFile {
		return false
	}

	var hasChildren bool
	for key, child := range t.Children {
		if child.prune() {
			delete(t.Children, key)
		} else {
			hasChildren = true
		}
	}

	return !hasChildren
}

/*
* TallyCommitsByPath() returns a tree of nodes mirroring the working directory
* with a tally for each node.
*
* Handling renamed files is tricky.
*
* When a file is renamed, we move the leaf node and its attached tally objects
* to a new place in the tree. We do not update any interior nodes.
*
* Because no interior nodes are updated, this can lead to situations where, say,
* the tree records more commits happening in a directory than on the files in
* that directory (this could happen if a file were moved out of a directory).
* However, the commit count for the directory still relfects the number of
* commits that would be listed if you were to run `git log` on that directory.
*
* As for the new leaf node, it reports commits, lines added, etc. as if
* `git log --follow` had been used on that filepath.
 */
func TallyCommitsByPath(
	commits iter.Seq2[git.Commit, error],
	opts TallyOpts,
) (*TreeNode, error) {
	root := newNode(false)

	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		for _, diff := range commit.FileDiffs {
			path, err := cleanPath(diff.Path)
			if err != nil {
				return nil,
					fmt.Errorf("error cleaning diff path: \"%s\"", diff.Path)
			}

			if diff.Action == git.Rename {
				// Handle renamed file
				oldPath := path
				newPath, err := cleanPath(diff.MoveDest)
				if err != nil {
					return nil, fmt.Errorf(
						"error cleaning diff path: \"%s\"",
						diff.Path,
					)
				}

				var node *TreeNode
				node, err = root.remove(oldPath)
				if err != nil {
					return nil, fmt.Errorf("error removing old node: %w", err)
				}

				err = root.insert(newPath, node)
				if err != nil {
					// Don't fail, just warn. Git allows files to be renamed to
					// existing filenames sometimes.
					logger().Debug(
						"WARNING: path exists in tree",
						"error",
						err.Error(),
						"commit",
						commit.Name(),
					)
				}

				root.edit(newPath, commit, diff, opts)
			} else if diff.Action == git.Delete {
				root.remove(path)
			} else {
				// Normal file update
				root.edit(path, commit, diff, opts)
			}
		}
	}

	root.prune()
	return root, nil
}
