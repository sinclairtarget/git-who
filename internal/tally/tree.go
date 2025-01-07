package tally

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/sinclairtarget/git-who/internal/git"
)

// A file tree of edits to the repo
type TreeNode struct {
	Tally          Tally
	Children       map[string]*TreeNode
	tallies        map[string]Tally
	lastCommitSeen string
	isFile         bool // A file, rather than a directory
	inWTree        bool // In working tree
}

func newNode(isFile bool, inWTree bool) *TreeNode {
	return &TreeNode{
		Children: map[string]*TreeNode{},
		tallies:  map[string]Tally{},
		isFile:   isFile,
		inWTree:  inWTree,
	}
}

func (t *TreeNode) String() string {
	return fmt.Sprintf("{ %d }", len(t.tallies))
}

func checkPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path cannot be empty")
	}

	if filepath.IsAbs(path) {
		return "", errors.New("path cannot be absolute")
	}

	return path, nil
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
	inWTree bool,
	opts TallyOpts,
) {
	if path != "" {
		// Insert child
		p, nextP := splitPath(path)
		child, ok := t.Children[p]
		if !ok {
			t.Children[p] = newNode(nextP == "", inWTree)
			child = t.Children[p]
		}

		child.edit(nextP, commit, diff, inWTree, opts)
	}

	// Update whether in working directory
	t.inWTree = inWTree

	// Add tally
	key := opts.Key(commit)
	nodeTally, ok := t.tallies[key]
	if !ok {
		nodeTally = Tally{
			AuthorName:  commit.AuthorName,
			AuthorEmail: commit.AuthorEmail,
		}
	}

	if path == "" || inWTree {
		nodeTally.LinesAdded += diff.LinesAdded
		nodeTally.LinesRemoved += diff.LinesRemoved
	}

	if commit.Hash != t.lastCommitSeen {
		if path == "" || inWTree {
			nodeTally.Commits += 1
			nodeTally.LastCommitTime = commit.Date
		}
		t.lastCommitSeen = commit.Hash
	}

	nodeTally.FileCount = 0
	if len(t.Children) > 0 {
		for _, child := range t.Children {
			if !child.inWTree {
				continue
			}
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
	sorted := sortTallies(t.tallies, opts.Mode)
	t.Tally = sorted[0]
}

func (t *TreeNode) insert(path string, node *TreeNode) error {
	if node == nil {
		panic("cannot insert nil node into tally tree")
	}

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
			cur.Children[p] = newNode(false, node.inWTree)
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

// Removes the node at the given path from the tree. Returns a nil node if there
// is no node at that path.
func (t *TreeNode) remove(path string) *TreeNode {
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
			return nil
		}
	}

	// Remove child node from children map
	child, ok := cur.Children[p]
	if !ok {
		return nil
	}

	delete(cur.Children, p)
	return child
}

/*
* Prunes the following types of nodes from the tree:
*
* 1. Nodes that don't exist in the working tree.
* 2. Interior nodes (directories) with no children.
*
* Returns true if this node needs pruning.
 */
func (t *TreeNode) prune() bool {
	if !t.inWTree {
		return true
	}

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
* When a file is renamed, we move the leaf node and its attached tally objects
* to a new place in the tree. We do not update any interior nodes.
*
* Because no interior nodes are updated, this can lead to situations where, say,
* the tree records more commits happening in a directory than on the files in
* that directory (this could happen if a file were moved out of a directory).
*
* We prune all paths from the tree that are not in the given working tree before
* returning. We also only allow renames to paths in the working tree.
 */
func TallyCommitsByPath(
	commits iter.Seq2[git.Commit, error],
	wtreefiles map[string]bool,
	opts TallyOpts,
) (*TreeNode, error) {
	root := newNode(false, true)

	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		for _, diff := range commit.FileDiffs {
			path, err := checkPath(diff.Path)
			if err != nil {
				return nil,
					fmt.Errorf("error cleaning diff path: \"%s\"", diff.Path)
			}

			if diff.Action == git.Rename {
				// Handle renamed file
				oldPath := path
				newPath, err := checkPath(diff.MoveDest)
				if err != nil {
					return nil, fmt.Errorf(
						"error cleaning diff path: \"%s\"",
						diff.Path,
					)
				}

				isWTreeDest := wtreefiles[newPath]
				if !isWTreeDest {
					continue
				}

				node := root.remove(oldPath)
				if node != nil {
					err = root.insert(newPath, node)
					if err != nil {
						// Don't fail, just warn.
						logger().Debug(
							"WARNING: path exists in tree",
							"error",
							err.Error(),
							"commit",
							commit.Name(),
						)
					}
				}

				root.edit(newPath, commit, diff, true, opts)
			} else if !commit.IsMerge {
				// Normal file update
				inWTree := wtreefiles[path]
				root.edit(path, commit, diff, inWTree, opts)
			}
		}
	}

	root.prune()
	return root, nil
}
