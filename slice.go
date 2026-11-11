// Package immutable provides Slice, a generic immutable (persistent) wrapper
// around []T.
//
// A Slice is a value type: every "mutating" operation returns a brand-new
// Slice and leaves the receiver untouched. Because of that, a Slice is safe
// for concurrent use and can be handed out to untrusted code without risk of
// modification — as long as T itself holds no mutable state (e.g. T must not
// be a map, pointer, slice or struct exposing one).
//
// Internally the invariant len(data) == cap(data) (or data == nil) is
// maintained, so every operation that would "append" is forced to allocate a
// fresh array. This trades some allocation for a hard immutability guarantee:
// two Slices can never share a writable backing array.
//
// The zero value Slice[T] is a valid, empty Slice.
package immutable

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"slices"
)

// Slice is an immutable sequence of T.
type Slice[T any] struct {
	// Always satisfies: data == nil || len(data) == cap(data).
	data []T
}

// NewSlice returns a Slice holding the given items.
// The items are copied, so later changes to the caller's arguments
// (e.g. when called as NewSlice(xs...)) do not affect the result.
func NewSlice[T any](items ...T) Slice[T] {
	return Slice[T]{data: slices.Clone(items)}
}

// FromSlice returns a Slice copying the contents of s.
// The input slice is not retained or aliased.
func FromSlice[T any](s []T) Slice[T] {
	return Slice[T]{data: slices.Clone(s)}
}

// EmptySlice returns an empty Slice.
func EmptySlice[T any]() Slice[T] {
	return Slice[T]{}
}

// Len returns the number of elements.
func (s Slice[T]) Len() int { return len(s.data) }

// Get returns the element at index i and true, or the zero T and false if
// i is out of range.
func (s Slice[T]) Get(i int) (T, bool) {
	if i < 0 || i >= len(s.data) {
		var zero T
		return zero, false
	}
	return s.data[i], true
}

// At returns the element at index i. It panics if i is out of range,
// just like indexing a plain slice.
func (s Slice[T]) At(i int) T {
	return s.data[i]
}

// First returns the first element and true, or the zero T and false if empty.
func (s Slice[T]) First() (T, bool) { return s.Get(0) }

// Last returns the last element and true, or the zero T and false if empty.
func (s Slice[T]) Last() (T, bool) { return s.Get(len(s.data) - 1) }

// Sub returns the half-open range [lo, hi) of the Slice as a new Slice.
// It panics on invalid indexes, just like slicing a plain slice.
func (s Slice[T]) Sub(lo, hi int) Slice[T] {
	return Slice[T]{data: slices.Clone(s.data[lo:hi])}
}

// ToSlice returns a plain []T copy of the Slice's contents. The caller owns
// the returned slice and may modify it freely.
func (s Slice[T]) ToSlice() []T { return slices.Clone(s.data) }

// Seq returns an iterator over the elements, for use with range.
func (s Slice[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s.data {
			if !yield(v) {
				return
			}
		}
	}
}

// Seq2 returns an iterator over index/element pairs, for use with range.
func (s Slice[T]) Seq2() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, v := range s.data {
			if !yield(i, v) {
				return
			}
		}
	}
}

// -- "Mutating" operations: each returns a new Slice. --

// Append returns a new Slice with items added to the end.
func (s Slice[T]) Append(items ...T) Slice[T] {
	out := make([]T, 0, len(s.data)+len(items))
	out = append(out, s.data...)
	out = append(out, items...)
	return Slice[T]{data: out}
}

// Prepend returns a new Slice with items added to the beginning,
// keeping their relative order (Prepend(x, y) on [a] yields [x, y, a]).
func (s Slice[T]) Prepend(items ...T) Slice[T] {
	out := make([]T, 0, len(s.data)+len(items))
	out = append(out, items...)
	out = append(out, s.data...)
	return Slice[T]{data: out}
}

// Concat returns a new Slice with the contents of s followed by other.
func (s Slice[T]) Concat(other Slice[T]) Slice[T] {
	out := make([]T, 0, len(s.data)+len(other.data))
	out = append(out, s.data...)
	out = append(out, other.data...)
	return Slice[T]{data: out}
}

// Set returns a new Slice with the element at index i replaced by v.
// It panics if i is out of range.
func (s Slice[T]) Set(i int, v T) Slice[T] {
	if i < 0 || i >= len(s.data) {
		panic(fmt.Sprintf("immutable: index %d out of range [0, %d)", i, len(s.data)))
	}
	out := slices.Clone(s.data)
	out[i] = v
	return Slice[T]{data: out}
}

// Insert returns a new Slice with v inserted at index i, shifting later
// elements right. i may equal Len() to append at the end.
// It panics if i is outside [0, Len()].
func (s Slice[T]) Insert(i int, v T) Slice[T] {
	if i < 0 || i > len(s.data) {
		panic(fmt.Sprintf("immutable: index %d out of range [0, %d]", i, len(s.data)))
	}
	out := make([]T, 0, len(s.data)+1)
	out = append(out, s.data[:i]...)
	out = append(out, v)
	out = append(out, s.data[i:]...)
	return Slice[T]{data: out}
}

// Remove returns a new Slice with the element at index i removed.
// It panics if i is out of range.
func (s Slice[T]) Remove(i int) Slice[T] { return s.RemoveRange(i, i+1) }

// RemoveFirst returns a new Slice without the first element; an empty Slice
// is returned unchanged.
func (s Slice[T]) RemoveFirst() Slice[T] {
	if len(s.data) == 0 {
		return s
	}
	return s.RemoveRange(0, 1)
}

// RemoveLast returns a new Slice without the last element; an empty Slice
// is returned unchanged.
func (s Slice[T]) RemoveLast() Slice[T] {
	if len(s.data) == 0 {
		return s
	}
	return s.RemoveRange(len(s.data)-1, len(s.data))
}

// RemoveRange returns a new Slice with the half-open range [lo, hi) removed.
// It panics on invalid indexes, just like slicing a plain slice.
func (s Slice[T]) RemoveRange(lo, hi int) Slice[T] {
	removed := s.data[lo:hi] // panics exactly like a plain slice expression
	out := make([]T, 0, len(s.data)-len(removed))
	out = append(out, s.data[:lo]...)
	out = append(out, s.data[hi:]...)
	return Slice[T]{data: out}
}

// Reverse returns a new Slice with the elements in reverse order.
func (s Slice[T]) Reverse() Slice[T] {
	out := slices.Clone(s.data)
	slices.Reverse(out)
	return Slice[T]{data: out}
}

// -- Transformations. --

// Map returns a new Slice of type U containing f applied to each element.
func (s Slice[T]) Map[U any](f func(T) U) Slice[U] {
	out := make([]U, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, f(v))
	}
	return Slice[U]{data: out}
}

// Filter returns a new Slice with the elements for which pred is true,
// preserving order.
func (s Slice[T]) Filter(pred func(T) bool) Slice[T] {
	out := make([]T, 0, len(s.data))
	for _, v := range s.data {
		if pred(v) {
			out = append(out, v)
		}
	}
	return Slice[T]{data: out}
}

// Reduce folds the elements into a single value, starting from init and
// applying f left to right.
func (s Slice[T]) Reduce[U any](init U, f func(U, T) U) U {
	acc := init
	for _, v := range s.data {
		acc = f(acc, v)
	}
	return acc
}

// Sorted returns a new Slice sorted according to the comparison function
// cmp (negative if a < b, zero if equal, positive if a > b).
// The sort is not stable; use SortedStable for a stable sort.
func (s Slice[T]) Sorted(cmp func(a, b T) int) Slice[T] {
	out := slices.Clone(s.data)
	slices.SortFunc(out, cmp)
	return Slice[T]{data: out}
}

// SortedStable is like Sorted but preserves the original order of
// equal elements.
func (s Slice[T]) SortedStable(cmp func(a, b T) int) Slice[T] {
	out := slices.Clone(s.data)
	slices.SortStableFunc(out, cmp)
	return Slice[T]{data: out}
}

// -- Queries. --

// Any reports whether at least one element satisfies pred.
func (s Slice[T]) Any(pred func(T) bool) bool {
	return slices.ContainsFunc(s.data, pred)
}

// All reports whether every element satisfies pred.
func (s Slice[T]) All(pred func(T) bool) bool {
	for _, v := range s.data {
		if !pred(v) {
			return false
		}
	}
	return true
}

// IndexOfFunc returns the index of the first element satisfying pred, or -1.
func (s Slice[T]) IndexOfFunc(pred func(T) bool) int {
	return slices.IndexFunc(s.data, pred)
}

// String implements fmt.Stringer, rendering the elements as a Go slice
// literal. Elements are formatted with %v.
func (s Slice[T]) String() string {
	return fmt.Sprintf("%v", s.data)
}

// -- JSON. --

// IsZero implements json.IsZeroer, so struct fields of type Slice[T] can
// use the `json:",omitzero"` tag option. It is equivalent to Len() == 0.
func (s Slice[T]) IsZero() bool { return len(s.data) == 0 }

// MarshalJSON implements json.Marshaler, rendering the Slice as a JSON
// array. Elements are marshaled with their own JSON representations.
// An empty or zero Slice marshals to [] (never null), since nil and empty
// are indistinguishable in this API.
func (s Slice[T]) MarshalJSON() ([]byte, error) {
	if len(s.data) == 0 {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(s.data)
	if err != nil {
		return nil, fmt.Errorf("immutable: marshaling Slice: %w", err)
	}
	return b, nil
}

// UnmarshalJSON implements json.Unmarshaler, decoding a JSON array into a
// fresh Slice. The decoded data is newly allocated and not shared with any
// other value. A JSON null decodes to the zero (empty) Slice.
func (s *Slice[T]) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		s.data = nil
		return nil
	}
	var out []T
	if err := json.Unmarshal(b, &out); err != nil {
		return fmt.Errorf("immutable: unmarshaling Slice: %w", err)
	}
	s.data = slices.Clone(out)
	return nil
}

// -- Package-level helpers requiring extra constraints on T. --

// IndexOf returns the index of the first element equal to v, or -1.
func IndexOf[T comparable](s Slice[T], v T) int { return slices.Index(s.data, v) }

// Contains reports whether any element equals v.
func Contains[T comparable](s Slice[T], v T) bool { return slices.Contains(s.data, v) }

// Equal reports whether a and b have the same length and all corresponding
// elements satisfy eq.
func Equal[T any](a, b Slice[T], eq func(x, y T) bool) bool {
	return slices.EqualFunc(a.data, b.data, eq)
}

// EqualValues is like Equal for comparable T, using ==.
func EqualValues[T comparable](a, b Slice[T]) bool { return slices.Equal(a.data, b.data) }

// Sort returns a new Slice of ordered elements in ascending order.
func Sort[T cmp.Ordered](s Slice[T]) Slice[T] {
	out := slices.Clone(s.data)
	slices.Sort(out)
	return Slice[T]{data: out}
}

// SortBy returns a new Slice ordered ascending by the key function.
// The sort is stable.
func SortBy[T any, K cmp.Ordered](s Slice[T], key func(T) K) Slice[T] {
	out := slices.Clone(s.data)
	slices.SortStableFunc(out, func(a, b T) int { return cmp.Compare(key(a), key(b)) })
	return Slice[T]{data: out}
}
