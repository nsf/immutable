package immutable

import (
	"cmp"
	"slices"
	"testing"
)

func TestNoAliasingOnAppend(t *testing.T) {
	s := NewSlice(1, 2, 3)
	a := s.Append(4)
	b := s.Append(5)

	if got, want := a.String(), "[1 2 3 4]"; got != want {
		t.Errorf("a = %s, want %s", got, want)
	}
	if got, want := b.String(), "[1 2 3 5]"; got != want {
		t.Errorf("b = %s, want %s", got, want)
	}
	if got, want := s.String(), "[1 2 3]"; got != want {
		t.Errorf("original mutated after appends: %s, want %s", got, want)
	}
}

func TestFromSliceDoesNotAliasInput(t *testing.T) {
	x := []int{1, 2, 3}
	s := FromSlice(x)
	x[0] = 99
	if s.At(0) != 1 {
		t.Errorf("input slice mutation leaked into Slice: %v", s)
	}

	out := s.ToSlice()
	out[0] = 42
	if s.At(0) != 1 {
		t.Errorf("ToSlice result aliases internal data: %v", s)
	}
}

func TestZeroValue(t *testing.T) {
	var s Slice[int]
	if s.Len() != 0 || !s.IsEmpty() {
		t.Errorf("zero value not empty: len=%d", s.Len())
	}
	if _, ok := s.First(); ok {
		t.Error("First on empty returned ok")
	}
	if _, ok := s.Get(0); ok {
		t.Error("Get(0) on empty returned ok")
	}
	if s.Append(1).Len() != 1 {
		t.Error("Append on zero value broken")
	}
}

func TestMutatingOpsReturnNewInstances(t *testing.T) {
	s := NewSlice("a", "b", "c")

	if got := s.Set(1, "B").String(); got != "[a B c]" {
		t.Errorf("Set = %s", got)
	}
	if got := s.Insert(1, "x").String(); got != "[a x b c]" {
		t.Errorf("Insert = %s", got)
	}
	if got := s.Remove(1).String(); got != "[a c]" {
		t.Errorf("Remove = %s", got)
	}
	if got := s.RemoveRange(0, 2).String(); got != "[c]" {
		t.Errorf("RemoveRange = %s", got)
	}
	if got := s.RemoveFirst().String(); got != "[b c]" {
		t.Errorf("RemoveFirst = %s", got)
	}
	if got := s.RemoveLast().String(); got != "[a b]" {
		t.Errorf("RemoveLast = %s", got)
	}
	if got := s.Prepend("z").String(); got != "[z a b c]" {
		t.Errorf("Prepend = %s", got)
	}
	if got := s.Concat(NewSlice("d")).String(); got != "[a b c d]" {
		t.Errorf("Concat = %s", got)
	}
	if got := s.Reverse().String(); got != "[c b a]" {
		t.Errorf("Reverse = %s", got)
	}
	if got := s.Sub(1, 3).String(); got != "[b c]" {
		t.Errorf("Sub = %s", got)
	}
	// Receiver must be untouched by all of the above.
	if got := s.String(); got != "[a b c]" {
		t.Errorf("receiver mutated: %s", got)
	}
}

func TestTransformations(t *testing.T) {
	n := NewSlice(3, 1, 2)

	if got := n.Map(func(x int) int { return x * 2 }).String(); got != "[6 2 4]" {
		t.Errorf("Map = %s", got)
	}
	if got := n.MapAs(func(x int) string { return string(rune('a' + x)) }).String(); got != "[d b c]" {
		t.Errorf("MapAs = %s", got)
	}
	if got := n.Filter(func(x int) bool { return x > 1 }).String(); got != "[3 2]" {
		t.Errorf("Filter = %s", got)
	}
	if got := n.Reduce(0, func(acc, x int) int { return acc + x }); got != 6 {
		t.Errorf("Reduce = %d, want 6", got)
	}
	if got := Sort(n).String(); got != "[1 2 3]" {
		t.Errorf("Sort = %s", got)
	}
	type person struct {
		name string
		age  int
	}
	ps := NewSlice(person{"bob", 30}, person{"amy", 25}, person{"cal", 30})
	youngest := SortBy(ps, func(v person) int { return v.age })
	if got := youngest.MapAs(func(v person) string { return v.name }); got.String() != "[amy bob cal]" {
		t.Errorf("SortBy unstable or wrong: %s", got)
	}
	if s := n.Sorted(func(a, b int) int { return cmp.Compare(b, a) }); s.String() != "[3 2 1]" {
		t.Errorf("Sorted desc = %s", s)
	}
	if got := n.String(); got != "[3 1 2]" {
		t.Errorf("receiver mutated by transformations: %s", got)
	}
}

func TestQueries(t *testing.T) {
	s := NewSlice(1, 2, 3)
	if !Contains(s, 2) || Contains(s, 9) {
		t.Error("Contains wrong")
	}
	if IndexOf(s, 3) != 2 || IndexOf(s, 9) != -1 {
		t.Error("IndexOf wrong")
	}
	if !s.Any(func(x int) bool { return x == 3 }) || s.All(func(x int) bool { return x > 1 }) {
		t.Error("Any/All wrong")
	}
	if s.IndexOfFunc(func(x int) bool { return x > 2 }) != 2 {
		t.Error("IndexOfFunc wrong")
	}
	if !EqualValues(s, NewSlice(1, 2, 3)) || EqualValues(s, NewSlice(1, 2)) {
		t.Error("EqualValues wrong")
	}
	if !Equal(s, NewSlice(1, 2, 4), func(a, b int) bool { return a <= b }) {
		t.Error("Equal wrong")
	}
	last, ok := s.Last()
	if !ok || last != 3 {
		t.Error("Last wrong")
	}
}

func TestRangeIteration(t *testing.T) {
	s := NewSlice(10, 20, 30)
	var sum, idxSum int
	for i, v := range s.Seq2() {
		sum += v
		idxSum += i
	}
	if sum != 60 || idxSum != 3 {
		t.Errorf("Seq2: sum=%d idxSum=%d", sum, idxSum)
	}
	var seen []int
	for v := range s.Seq() {
		seen = append(seen, v)
		if v == 20 {
			break
		}
	}
	if !slices.Equal(seen, []int{10, 20}) {
		t.Errorf("Seq early break = %v", seen)
	}
}

func TestPanics(t *testing.T) {
	s := NewSlice(1, 2, 3)
	panics := []func(){
		func() { s.At(3) },
		func() { s.At(-1) },
		func() { s.Set(3, 0) },
		func() { s.Insert(4, 0) },
		func() { s.Remove(3) },
		func() { s.RemoveRange(-1, 1) },
		func() { s.RemoveRange(2, 1) },
		func() { s.Sub(0, 4) },
	}
	for i, f := range panics {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("case %d: expected panic", i)
				}
			}()
			f()
		}()
	}
}
