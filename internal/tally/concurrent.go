package tally

import (
	"iter"

	"github.com/sinclairtarget/git-who/internal/git"
)

type TallyFunc[T any] func(commits iter.Seq2[git.Commit, error]) (T, error)
type MergeFunc[T any] func(a, b T) T
type FinalizeFunc[T any, R any] func(t T) R
