package immutable

import (
	"cmp"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
)

// Map is an immutable associative array from K to V.
//
// Like Slice, every "mutating" operation returns a brand-new Map and leaves
// the receiver untouched. Internally the map is never modified once built,
// so derived Maps may share their storage freely: copying or passing a Map
// by value is O(1), while each write copies the entries (copy-on-write).
// This makes Map a good fit for mostly-read data; a hot write path on large
// maps would need a structural-sharing trie instead.
//
// The zero value Map[K, V] is a valid, empty Map.
//
// The same caveat as for Slice applies: immutability of the entries depends
// on V itself holding no mutable state.
type Map[K comparable, V any] struct {
	// Never modified after the Map value is created.
	data map[K]V
}

// Pair is a key/value pair, used by NewMap and SetAll.
type Pair[K comparable, V any] struct {
	Key K
	Val V
}

// PairOf constructs a Pair.
func PairOf[K comparable, V any](k K, v V) Pair[K, V] { return Pair[K, V]{Key: k, Val: v} }

// NewMap returns a Map built from the given pairs. Later pairs with a
// duplicate key win.
func NewMap[K comparable, V any](pairs ...Pair[K, V]) Map[K, V] {
	out := make(map[K]V, len(pairs))
	for _, p := range pairs {
		out[p.Key] = p.Val
	}
	return Map[K, V]{data: out}
}

// FromMap returns a Map copying the contents of m.
// The input map is not retained or aliased.
func FromMap[K comparable, V any](m map[K]V) Map[K, V] {
	return Map[K, V]{data: maps.Clone(m)}
}

// EmptyMap returns an empty Map.
func EmptyMap[K comparable, V any]() Map[K, V] { return Map[K, V]{} }

// -- Read-only access. --

// Len returns the number of entries.
func (m Map[K, V]) Len() int { return len(m.data) }

// IsEmpty reports whether the Map holds no entries.
func (m Map[K, V]) IsEmpty() bool { return len(m.data) == 0 }

// Get returns the value for key k and true, or the zero V and false if
// k is absent.
func (m Map[K, V]) Get(k K) (V, bool) {
	v, ok := m.data[k]
	return v, ok
}

// GetOr returns the value for key k, or def if k is absent.
func (m Map[K, V]) GetOr(k K, def V) V {
	if v, ok := m.data[k]; ok {
		return v
	}
	return def
}

// Has reports whether k is present.
func (m Map[K, V]) Has(k K) bool {
	_, ok := m.data[k]
	return ok
}

// Keys returns a Slice of the map's keys in unspecified order.
func (m Map[K, V]) Keys() Slice[K] {
	ks := make([]K, 0, len(m.data))
	for k := range m.data {
		ks = append(ks, k)
	}
	return FromSlice(ks)
}

// Values returns a Slice of the map's values in unspecified order
// (the same unspecified order as Keys, so they correspond pairwise).
func (m Map[K, V]) Values() Slice[V] {
	vs := make([]V, 0, len(m.data))
	for _, v := range m.data {
		vs = append(vs, v)
	}
	return FromSlice(vs)
}

// ToMap returns a plain map[K]V copy of the Map's entries. The caller owns
// the returned map and may modify it freely.
func (m Map[K, V]) ToMap() map[K]V {
	out := make(map[K]V, len(m.data))
	maps.Copy(out, m.data)
	return out
}

// Seq2 returns an iterator over key/value pairs, for use with range.
// Iteration order is unspecified, as for a plain map.
func (m Map[K, V]) Seq2() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range m.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Seq returns an iterator over the values, for use with range.
func (m Map[K, V]) Seq() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range m.data {
			if !yield(v) {
				return
			}
		}
	}
}

// Any reports whether at least one entry satisfies pred.
func (m Map[K, V]) Any(pred func(K, V) bool) bool {
	for k, v := range m.data {
		if pred(k, v) {
			return true
		}
	}
	return false
}

// All reports whether every entry satisfies pred.
func (m Map[K, V]) All(pred func(K, V) bool) bool {
	for k, v := range m.data {
		if !pred(k, v) {
			return false
		}
	}
	return true
}

// -- "Mutating" operations: each returns a new Map. --

// Set returns a new Map with k mapped to v.
func (m Map[K, V]) Set(k K, v V) Map[K, V] {
	out := make(map[K]V, len(m.data)+1)
	maps.Copy(out, m.data)
	out[k] = v
	return Map[K, V]{data: out}
}

// SetAll returns a new Map with all the given pairs added; later pairs with
// duplicate keys win. If pairs is empty, m is returned unchanged.
func (m Map[K, V]) SetAll(pairs ...Pair[K, V]) Map[K, V] {
	if len(pairs) == 0 {
		return m
	}
	out := make(map[K]V, len(m.data)+len(pairs))
	maps.Copy(out, m.data)
	for _, p := range pairs {
		out[p.Key] = p.Val
	}
	return Map[K, V]{data: out}
}

// Update returns a new Map with the value for k replaced by f(old).
// If k is absent, m is returned unchanged.
func (m Map[K, V]) Update(k K, f func(V) V) Map[K, V] {
	old, ok := m.data[k]
	if !ok {
		return m
	}
	return m.Set(k, f(old))
}

// Delete returns a new Map without the entry for k. If k is absent, m is
// returned unchanged (sharing storage, which is safe).
func (m Map[K, V]) Delete(k K) Map[K, V] {
	if _, ok := m.data[k]; !ok {
		return m
	}
	out := make(map[K]V, len(m.data)-1)
	maps.Copy(out, m.data)
	delete(out, k)
	return Map[K, V]{data: out}
}

// Union returns a new Map with all entries of m and other. For keys present
// in both, combine receives (m's value, other's value) and its result wins.
func (m Map[K, V]) Union(other Map[K, V], combine func(fromM, fromOther V) V) Map[K, V] {
	out := make(map[K]V, len(m.data)+len(other.data))
	maps.Copy(out, m.data)
	for k, v := range other.data {
		if old, ok := out[k]; ok {
			out[k] = combine(old, v)
		} else {
			out[k] = v
		}
	}
	return Map[K, V]{data: out}
}

// Intersect returns a new Map with only the entries whose keys are present
// in both m and other, taking values from m.
func (m Map[K, V]) Intersect(other Map[K, V]) Map[K, V] {
	out := make(map[K]V)
	for k, v := range m.data {
		if _, ok := other.data[k]; ok {
			out[k] = v
		}
	}
	return Map[K, V]{data: out}
}

// Subtract returns a new Map with the entries of m whose keys are absent
// from other.
func (m Map[K, V]) Subtract(other Map[K, V]) Map[K, V] {
	out := make(map[K]V, len(m.data))
	for k, v := range m.data {
		if _, ok := other.data[k]; !ok {
			out[k] = v
		}
	}
	return Map[K, V]{data: out}
}

// -- Transformations. --

// MapValues returns a new Map whose values are of type W, produced by f.
func (m Map[K, V]) MapValues[W any](f func(V) W) Map[K, W] {
	out := make(map[K]W, len(m.data))
	for k, v := range m.data {
		out[k] = f(v)
	}
	return Map[K, W]{data: out}
}

// Filter returns a new Map with the entries for which pred is true.
func (m Map[K, V]) Filter(pred func(K, V) bool) Map[K, V] {
	out := make(map[K]V)
	for k, v := range m.data {
		if pred(k, v) {
			out[k] = v
		}
	}
	return Map[K, V]{data: out}
}

// Reduce folds the entries into a single value, starting from init and
// applying f over each key/value pair. Iteration order is unspecified, so f
// should be associative and commutative for meaningful results.
func (m Map[K, V]) Reduce[U any](init U, f func(U, K, V) U) U {
	acc := init
	for k, v := range m.data {
		acc = f(acc, k, v)
	}
	return acc
}

// String implements fmt.Stringer. Entries are rendered as a Go map literal;
// as for a plain map, entry order in the output is unspecified.
func (m Map[K, V]) String() string {
	return fmt.Sprintf("%v", m.data)
}

// -- JSON. --

// IsZero implements json.IsZeroer, so struct fields of type Map[K, V] can
// use the `json:",omitzero"` tag option. It is equivalent to IsEmpty.
func (m Map[K, V]) IsZero() bool { return len(m.data) == 0 }

// MarshalJSON implements json.Marshaler, rendering the Map as a JSON object.
// An empty or zero Map marshals to {} (never null).
//
// Key marshaling follows encoding/json's rules for a plain map[K]V: K must
// be a string, an integer type, or implement encoding.TextMarshaler; other
// key types (e.g. structs, floats) produce an error at marshal time.
func (m Map[K, V]) MarshalJSON() ([]byte, error) {
	if len(m.data) == 0 {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(m.data)
	if err != nil {
		return nil, fmt.Errorf("immutable: marshaling Map: %w", err)
	}
	return b, nil
}

// UnmarshalJSON implements json.Unmarshaler, decoding a JSON object into a
// fresh Map. The decoded map is newly allocated and not shared with any
// other value. A JSON null decodes to the zero (empty) Map. As for a plain
// map[K]V, K must be string-like (string, integer type or
// encoding.TextUnmarshaler); otherwise decoding fails with an error.
func (m *Map[K, V]) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		m.data = nil
		return nil
	}
	var out map[K]V
	if err := json.Unmarshal(b, &out); err != nil {
		return fmt.Errorf("immutable: unmarshaling Map: %w", err)
	}
	m.data = out
	return nil
}

// -- Package-level helpers requiring extra constraints. --

// SortedKeys returns the keys of m in ascending order.
func SortedKeys[K cmp.Ordered, V any](m Map[K, V]) Slice[K] {
	ks := slices.Sorted(maps.Keys(m.data))
	return FromSlice(ks)
}

// Equal reports whether a and b have the same keys and all corresponding
// values satisfy eq.
func EqualMaps[K comparable, V any](a, b Map[K, V], eq func(x, y V) bool) bool {
	return maps.EqualFunc(a.data, b.data, eq)
}

// EqualMapValues is like EqualMaps for comparable V, using ==.
func EqualMapValues[K comparable, V comparable](a, b Map[K, V]) bool {
	return maps.Equal(a.data, b.data)
}
