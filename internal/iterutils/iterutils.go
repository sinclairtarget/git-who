// Iterator helpers
package iterutils

import (
	"fmt"
	"iter"
)

// Turns a Seq into a Seq2 where the second element is always nil
func WithoutErrors[V any](seq iter.Seq[V]) iter.Seq2[V, error] {
	return func(yield func(V, error) bool) {
		for v := range seq {
			if !yield(v, nil) {
				break
			}
		}
	}
}

// Turns a seq2 into a slice
func Collect[V any](seq iter.Seq2[V, error]) ([]V, error) {
	s := []V{}
	for v, err := range seq {
		if err != nil {
			return nil, fmt.Errorf("error collecting sequence: %w", err)
		}
		s = append(s, v)
	}

	return s, nil
}
