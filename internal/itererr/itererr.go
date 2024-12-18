// Iterator with error field you can check after iteration.
//
// Otherwise I don't know how the code implementing an iterator is supposed to
// propagate an error that occurred during cleanup.
package itererr

import "iter"

type Iter[T any] struct {
	Seq iter.Seq[T]
	Err error
}

func FromSlice[T any](slice []T) *Iter[T] {
	it := Iter[T] {}

	it.Seq = func(yield func(T) bool) {
		for _, t := range slice {
			if (!yield(t)) {
				break
			}
		}
	}

	return &it
}
