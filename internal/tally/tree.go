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

var EmptyTreeErr = errors.New("No commits; tree is empty.")

// A file tree of edits to the repo
type TreeNode struct {
	Tally      FinalTally
	Children   map[string]*TreeNode
	InWorkTree bool // In git working tree/directory
	tallies    map[string]Tally
}

func newNode(inWTree bool) *TreeNode {
	return &TreeNode{
		Children:   map[string]*TreeNode{},
		InWorkTree: inWTree,
		tallies:    map[string]Tally{},
	}
}

func (t *TreeNode) String() string {
	return fmt.Sprintf("{ %d }", len(t.tallies))
}

// Splits path into first dir and remainder.
func splitPath(path string) (string, string) {
	dir, subpath, found := strings.Cut(path, string(os.PathSeparator))
	if !found {
		return path, ""
	}

	return dir, subpath
}

func (t *TreeNode) insert(path string, key string, tally Tally, inWTree bool) {
	if path == "" {
		// Leaf
		t.tallies[key] = tally
		return
	}

	// Insert child
	p, nextP := splitPath(path)
	child, ok := t.Children[p]
	if !ok {
		child = newNode(inWTree)
	}
	child.InWorkTree = child.InWorkTree || inWTree
	t.Children[p] = child

	child.insert(nextP, key, tally, inWTree)
}

func (t *TreeNode) Rank(mode TallyMode) *TreeNode {
	if len(t.Children) > 0 {
		// Recursively sum up metrics.
		// For each author, merge the tallies for all children together.
		for p, child := range t.Children {
			t.Children[p] = child.Rank(mode)

			for key, childTally := range child.tallies {
				tally, ok := t.tallies[key]
				if !ok {
					tally.name = childTally.name
					tally.email = childTally.email
					tally.commitset = map[string]bool{}
				}

				tally = tally.Combine(childTally)
				t.tallies[key] = tally
			}
		}
	}

	// Pick best tally for the node according to the tally mode
	sorted := Rank(t.tallies, mode)
	t.Tally = sorted[0]
	return t
}

/*
* TallyCommitsTree() returns a tree of nodes mirroring the working directory
* with a tally for each node.
 */
func TallyCommitsTree(
	commits iter.Seq2[git.Commit, error],
	opts TallyOpts,
	worktreePaths map[string]bool,
	gitRootPath string,
) (*TreeNode, error) {
	// Tally paths
	talliesByPath, err := TallyCommitsByPath(commits, opts)
	if err != nil {
		return nil, err
	}

	return TallyCommitsTreeFromPaths(talliesByPath, worktreePaths, gitRootPath)
}

func TallyCommitsTreeFromPaths(
	talliesByPath TalliesByPath,
	worktreePaths map[string]bool,
	gitRootPath string,
) (*TreeNode, error) {
	root := newNode(true)

	wd, err := os.Getwd()
	if err != nil {
		return root, err
	}

	// Build tree
	for key, pathTallies := range talliesByPath {
		for path, tally := range pathTallies {
			relPath := path
			if gitRootPath != "" {
				// Adjust path for working dir
				absPath := filepath.Join(gitRootPath, path)
				relPath, err = filepath.Rel(wd, absPath)
				if err != nil || !filepath.IsLocal(relPath) {
					continue // Skip any paths outside of working dir
				}
			}

			inWTree := worktreePaths[relPath]
			root.insert(relPath, key, tally, inWTree)
		}
	}

	if len(root.Children) == 0 {
		return root, EmptyTreeErr
	}

	return root, nil
}
