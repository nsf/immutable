package immutable

import "testing"

func TestMapNoAliasing(t *testing.T) {
	m := NewMap(PairOf("a", 1), PairOf("b", 2))
	x := m.Set("c", 3)
	y := m.Set("c", 4).Set("b", 9)

	if got := m.String(); got != "map[a:1 b:2]" {
		t.Errorf("original = %s, want map[a:1 b:2]", got)
	}
	if got := x.String(); got != "map[a:1 b:2 c:3]" {
		t.Errorf("x = %s", got)
	}
	if got := y.String(); got != "map[a:1 b:9 c:4]" {
		t.Errorf("y = %s", got)
	}
}

func TestMapInputOutputAliasing(t *testing.T) {
	src := map[string]int{"a": 1}
	m := FromMap(src)
	src["a"] = 99
	if v, _ := m.Get("a"); v != 1 {
		t.Errorf("input map mutation leaked into Map: %v", m)
	}

	out := m.ToMap()
	out["a"] = 42
	if v, _ := m.Get("a"); v != 1 {
		t.Errorf("ToMap result aliases internal data: %v", m)
	}
}

func TestMapZeroValue(t *testing.T) {
	var m Map[string, int]
	if m.Len() != 0 || !m.IsEmpty() || m.Has("x") {
		t.Errorf("zero value not empty: %v", m)
	}
	if _, ok := m.Get("x"); ok {
		t.Error("Get on empty returned ok")
	}
	if got := m.Set("x", 1); !got.Has("x") {
		t.Error("Set on zero value broken")
	}
	if got := m.Delete("x"); !got.IsEmpty() {
		t.Error("Delete on empty broken")
	}
}

func TestMapOps(t *testing.T) {
	m := NewMap(PairOf("a", 1), PairOf("b", 2), PairOf("c", 3))

	if got := m.SetAll(PairOf("d", 4), PairOf("a", 9)).String(); got != "map[a:9 b:2 c:3 d:4]" {
		t.Errorf("SetAll = %s", got)
	}
	if got := m.Update("a", func(v int) int { return v * 10 }).String(); got != "map[a:10 b:2 c:3]" {
		t.Errorf("Update = %s", got)
	}
	if got := m.Update("zz", func(v int) int { return v }); !EqualMapValues(got, m) {
		t.Error("Update on missing key should be a no-op")
	}
	if got := m.Delete("b").String(); got != "map[a:1 c:3]" {
		t.Errorf("Delete = %s", got)
	}
	if got := m.Delete("zz"); !EqualMapValues(got, m) {
		t.Error("Delete of missing key should return m unchanged")
	}
	if v := m.GetOr("a", 77); v != 1 {
		t.Errorf("GetOr existing = %d", v)
	}
	if v := m.GetOr("zz", 77); v != 77 {
		t.Errorf("GetOr missing = %d", v)
	}
	if m.Len() != 3 {
		t.Errorf("Len = %d", m.Len())
	}
	// Receiver untouched by everything above.
	if got := m.String(); got != "map[a:1 b:2 c:3]" {
		t.Errorf("receiver mutated: %s", got)
	}
}

func TestMapSetLogic(t *testing.T) {
	a := NewMap(PairOf("x", 1), PairOf("y", 2))
	b := NewMap(PairOf("y", 20), PairOf("z", 3))

	u := a.Union(b, func(fromA, fromB int) int { return fromA + fromB })
	if got := u.String(); got != "map[x:1 y:22 z:3]" {
		t.Errorf("Union = %s", got)
	}
	if got := a.Intersect(b).String(); got != "map[y:2]" {
		t.Errorf("Intersect = %s", got)
	}
	if got := a.Subtract(b).String(); got != "map[x:1]" {
		t.Errorf("Subtract = %s", got)
	}
	if !EqualMapValues(a.Union(b, func(x, y int) int { return y }),
		NewMap(PairOf("x", 1), PairOf("y", 20), PairOf("z", 3))) {
		t.Error("Union with right-biased combine wrong")
	}
}

func TestMapTransformations(t *testing.T) {
	m := NewMap(PairOf("a", 1), PairOf("b", 2), PairOf("c", 3))

	if got := m.MapValues(func(v int) int { return v * 2 }).String(); got != "map[a:2 b:4 c:6]" {
		t.Errorf("MapValues = %s", got)
	}
	if got := m.MapValues(func(v int) string { return string(rune('0' + v)) }).String(); got != "map[a:1 b:2 c:3]" {
		t.Errorf("MapValues = %s", got)
	}
	if got := m.Filter(func(k string, v int) bool { return k != "b" && v >= 1 }).String(); got != "map[a:1 c:3]" {
		t.Errorf("Filter = %s", got)
	}
	if got := m.Reduce(0, func(acc int, _ string, v int) int { return acc + v }); got != 6 {
		t.Errorf("Reduce = %d, want 6", got)
	}
	if got := m.String(); got != "map[a:1 b:2 c:3]" {
		t.Errorf("receiver mutated by transformations: %s", got)
	}
}

func TestMapQueries(t *testing.T) {
	m := NewMap(PairOf("a", 1), PairOf("b", 2))
	if !m.Any(func(_ string, v int) bool { return v == 2 }) || m.All(func(_ string, v int) bool { return v > 1 }) {
		t.Error("Any/All wrong")
	}
	if !EqualMaps(m, NewMap(PairOf("a", 5), PairOf("b", 6)), func(x, y int) bool { return x < y }) {
		t.Error("EqualMaps wrong")
	}
	if ks := m.Keys().ToSlice(); len(ks) != 2 {
		t.Errorf("Keys = %v", ks)
	}
	if vs := m.Values().ToSlice(); len(vs) != 2 {
		t.Errorf("Values = %v", vs)
	}
	if got := SortedKeys(NewMap(PairOf(3, "c"), PairOf(1, "a"), PairOf(2, "b"))); got.String() != "[1 2 3]" {
		t.Errorf("SortedKeys = %s", got)
	}
}

func TestMapRangeIteration(t *testing.T) {
	m := NewMap(PairOf("a", 1), PairOf("b", 2), PairOf("c", 3))
	var sum int
	seen := map[string]bool{}
	for k, v := range m.Seq2() {
		sum += v
		seen[k] = true
	}
	if sum != 6 || len(seen) != 3 {
		t.Errorf("Seq2 sum=%d seen=%v", sum, seen)
	}
	var vsum int
	for v := range m.Seq() {
		vsum += v
	}
	if vsum != 6 {
		t.Errorf("Seq sum = %d", vsum)
	}
}
