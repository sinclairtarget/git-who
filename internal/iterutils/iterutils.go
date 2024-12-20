// Iterator helpers
package iterutils

import "iter"

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
