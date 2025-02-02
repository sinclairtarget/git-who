package concurrent

import (
	"iter"

	"github.com/sinclairtarget/git-who/internal/git"
)

const cacheChunkSize = chunkSize

// Transparently splits off commits to the cache queue
func cacheTee(
	commits iter.Seq2[git.Commit, error],
	toCache chan<- []git.Commit,
) iter.Seq2[git.Commit, error] {
	chunk := []git.Commit{}

	return func(yield func(git.Commit, error) bool) {
		for c, err := range commits {
			if err != nil {
				yield(c, err)
				return
			}

			chunk = append(chunk, c)

			if len(chunk) >= cacheChunkSize {
				toCache <- chunk
				chunk = []git.Commit{}
			}

			if !yield(c, nil) {
				break
			}
		}

		// Make sure to write any remainder
		if len(chunk) > 0 {
			toCache <- chunk
		}
	}
}
