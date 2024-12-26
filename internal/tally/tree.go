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
}

func newNode() *TreeNode {
	return &TreeNode{
		Children: map[string]*TreeNode{},
		tallies:  map[string]Tally{},
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
func (t *TreeNode) insert(
	path string,
	commit git.Commit,
	diff git.FileDiff,
	mode TallyMode,
) {
	if path != "" {
		// Insert child
		p, nextP := splitPath(path)
		child, ok := t.Children[p]
		if !ok {
			t.Children[p] = newNode()
			child = t.Children[p]
		}

		child.insert(nextP, commit, diff, mode)
	}

	// Add tally
	nodeTally, ok := t.tallies[commit.AuthorEmail]
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
		t.lastCommitSeen = commit.Hash
	}

	nodeTally.FileCount = 0
	if len(t.Children) > 0 {
		for _, child := range t.Children {
			childTally, ok := child.tallies[commit.AuthorEmail]
			if ok {
				nodeTally.FileCount += childTally.FileCount
			}
		}
	} else {
		nodeTally.FileCount = 1
	}

	t.tallies[commit.AuthorEmail] = nodeTally

	// Pick best tally for the node according to the tally mode
	sorted := slices.SortedFunc(maps.Values(t.tallies), func(a, b Tally) int {
		return -Compare(a, b, mode)
	})
	t.Tally = sorted[0]
}

// Returns a tree of file nodes with an attached tally object.
func TallyCommitsByPath(
	commits iter.Seq2[git.Commit, error],
	mode TallyMode,
) (*TreeNode, error) {
	root := newNode()

	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		for _, diff := range commit.FileDiffs {
			path, err := cleanPath(diff.Path)
			if err != nil {
				return nil,
					fmt.Errorf("error handling diff path: \"%s\"", diff.Path)
			}
			root.insert(path, commit, diff, mode)
		}
	}

	return root, nil
}
