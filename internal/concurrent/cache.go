package concurrent

import (
	"iter"

	"github.com/sinclairtarget/git-who/internal/git"
)

const cacheChunkSize = chunkSize

// Transparently splits off commits to the cache queue
func cacheTee(
	commits iter.Seq[git.Commit],
	toCache chan<- []git.Commit,
) iter.Seq[git.Commit] {
	chunk := []git.Commit{}

	return func(yield func(git.Commit) bool) {
		for c := range commits {
			chunk = append(chunk, c)

			if len(chunk) >= cacheChunkSize {
				toCache <- chunk
				chunk = []git.Commit{}
			}

			if !yield(c) {
				break
			}
		}

		// Make sure to write any remainder
		if len(chunk) > 0 {
			toCache <- chunk
		}
	}
}

// We want to get a list of revs from an iterator over commits while passing
// through the iterator to someone else for consumption.
//
// A little awkward... is there a better way to do this?
func revTee(
	commits iter.Seq[git.Commit],
	revs *[]string,
) iter.Seq[git.Commit] {
	return func(yield func(git.Commit) bool) {
		for c := range commits {
			*revs = append(*revs, c.Hash)
			if !yield(c) {
				return
			}
		}
	}
}
