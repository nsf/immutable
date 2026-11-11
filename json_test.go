package immutable

import (
	"encoding/json"
	"strings"
	"testing"
)

// -- Slice. --

func TestSliceMarshalJSON(t *testing.T) {
	if b, err := json.Marshal(NewSlice(1, 2, 3)); err != nil || string(b) != "[1,2,3]" {
		t.Errorf("Marshal = %s (err=%v), want [1,2,3]", b, err)
	}
	var zero Slice[int]
	if b, err := json.Marshal(zero); err != nil || string(b) != "[]" {
		t.Errorf("zero value marshaled to %s (err=%v), want []", b, err)
	}
	if b, err := json.Marshal(EmptySlice[string]()); err != nil || string(b) != "[]" {
		t.Errorf("empty marshaled to %s (err=%v), want []", b, err)
	}
	// Nested collections compose.
	n := NewSlice(NewMap(PairOf("a", 1)), EmptyMap[string, int]())
	if b, err := json.Marshal(n); err != nil || string(b) != `[{"a":1},{}]` {
		t.Errorf("nested = %s (err=%v), want [{\"a\":1},{}]", b, err)
	}
}

func TestSliceMarshalUsesElementMarshaler(t *testing.T) {
	// json.RawMessage has its own MarshalJSON; make sure it is honored.
	b, err := json.Marshal(NewSlice(json.RawMessage(`{"x":1}`), json.RawMessage(`[2]`)))
	if err != nil || string(b) != `[{"x":1},[2]]` {
		t.Errorf("element marshaler = %s (err=%v), want [{\"x\":1},[2]]", b, err)
	}
}

func TestSliceUnmarshalJSON(t *testing.T) {
	var s Slice[int]
	if err := json.Unmarshal([]byte(`[1,2,3]`), &s); err != nil {
		t.Fatal(err)
	}
	if s.String() != "[1 2 3]" {
		t.Errorf("unmarshaled = %s, want [1 2 3]", s)
	}
	// Decoded Slice must not alias anything observable: mutating a copy
	// must not affect the Slice.
	cp := s.ToSlice()
	cp[0] = 99
	if s.At(0) != 1 {
		t.Errorf("ToSlice after unmarshal aliases internal data: %v", s)
	}
	// Internal invariant len(data) == cap(data) must hold so Append cannot
	// write into the Slice's own array via a shared tail.
	a := s.Append(4)
	b := s.Append(5)
	if s.Len() != 3 || a.String() != "[1 2 3 4]" || b.String() != "[1 2 3 5]" {
		t.Errorf("aliasing after Append on unmarshaled Slice: s=%v a=%v b=%v", s, a, b)
	}

	if err := json.Unmarshal([]byte(`null`), &s); err != nil {
		t.Fatal(err)
	}
	if !s.IsEmpty() {
		t.Errorf("null should decode to empty, got %v", s)
	}
}

func TestSliceUnmarshalErrors(t *testing.T) {
	var s Slice[int]
	// Type errors reach UnmarshalJSON and get wrapped with context.
	for _, in := range []string{`{}`, `[1,"x"]`} {
		if err := json.Unmarshal([]byte(in), &s); err == nil {
			t.Errorf("expected error for input %s", in)
		} else if !strings.Contains(err.Error(), "immutable") {
			t.Errorf("error for %s lacks context: %v", in, err)
		}
	}
	// Malformed JSON is rejected by encoding/json itself before UnmarshalJSON
	// is called; it must still fail, without our prefix.
	for _, in := range []string{`[1,]`, `tru e`} {
		if err := json.Unmarshal([]byte(in), &s); err == nil {
			t.Errorf("expected error for input %s", in)
		}
	}
}

// -- Map. --

func TestMapMarshalJSON(t *testing.T) {
	m := NewMap(PairOf("a", 1), PairOf("b", 2))
	if b, err := json.Marshal(m); err != nil || string(b) != `{"a":1,"b":2}` {
		t.Errorf("Marshal = %s (err=%v)", b, err)
	}
	var zero Map[string, int]
	if b, err := json.Marshal(zero); err != nil || string(b) != "{}" {
		t.Errorf("zero value marshaled to %s (err=%v), want {}", b, err)
	}
	// Integer keys work, same as a plain map.
	if b, err := json.Marshal(NewMap(PairOf(1, "x"))); err != nil || string(b) != `{"1":"x"}` {
		t.Errorf("int keys = %s (err=%v), want {\"1\":\"x\"}", b, err)
	}
	// Nested collections compose.
	n := NewMap(PairOf("s", NewSlice(1, 2)))
	if b, err := json.Marshal(n); err != nil || string(b) != `{"s":[1,2]}` {
		t.Errorf("nested = %s (err=%v), want {\"s\":[1,2]}", b, err)
	}
}

func TestMapMarshalUnsupportedKeys(t *testing.T) {
	m := NewMap(PairOf(key{1}, "v"))
	if _, err := json.Marshal(m); err == nil {
		t.Error("expected error marshaling Map with struct keys")
	} else if !strings.Contains(err.Error(), "immutable") {
		t.Errorf("error lacks context: %v", err)
	}
}

func TestMapUnmarshalJSON(t *testing.T) {
	var m Map[string, int]
	if err := json.Unmarshal([]byte(`{"a":1,"b":2}`), &m); err != nil {
		t.Fatal(err)
	}
	if !EqualMapValues(m, NewMap(PairOf("a", 1), PairOf("b", 2))) {
		t.Errorf("unmarshaled = %v", m)
	}
	// Decoded map is owned by this Map only: mutating a ToMap copy must
	// not affect it, and Set must not disturb it.
	cp := m.ToMap()
	cp["a"] = 99
	if v, _ := m.Get("a"); v != 1 {
		t.Errorf("ToMap after unmarshal aliases internal data: %v", m)
	}
	m2 := m.Set("c", 3)
	if m.Has("c") || !m2.Has("c") {
		t.Errorf("Set on unmarshaled Map aliased storage: %v / %v", m, m2)
	}

	if err := json.Unmarshal([]byte(`null`), &m); err != nil {
		t.Fatal(err)
	}
	if !m.IsEmpty() {
		t.Errorf("null should decode to empty, got %v", m)
	}

	// Integer keys decode back.
	var mi Map[int, string]
	if err := json.Unmarshal([]byte(`{"7":"x"}`), &mi); err != nil {
		t.Fatal(err)
	}
	if v, ok := mi.Get(7); !ok || v != "x" {
		t.Errorf("int-key decode = %v", mi)
	}
}

func TestMapUnmarshalErrors(t *testing.T) {
	var m Map[string, int]
	// Type errors reach UnmarshalJSON and get wrapped with context.
	for _, in := range []string{`[]`, `{"a":"x"}`} {
		if err := json.Unmarshal([]byte(in), &m); err == nil {
			t.Errorf("expected error for input %s", in)
		} else if !strings.Contains(err.Error(), "immutable") {
			t.Errorf("error for %s lacks context: %v", in, err)
		}
	}
	// Malformed JSON is rejected by encoding/json itself before UnmarshalJSON
	// is called; it must still fail, without our prefix.
	for _, in := range []string{`{"a":1,}`, `not json`} {
		if err := json.Unmarshal([]byte(in), &m); err == nil {
			t.Errorf("expected error for input %s", in)
		}
	}
}

type key struct{ X int }

// -- Struct integration: omitzero and roundtrip. --

type config struct {
	Tags Slice[string]           `json:"tags,omitzero"`
	Env  Map[string, string]     `json:"env,omitzero"`
	Name string                  `json:"name"`
	Nest Map[string, Slice[int]] `json:"nest,omitzero"`
}

func TestStructRoundtripAndOmitZero(t *testing.T) {
	var empty config
	b, err := json.Marshal(empty)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"name":""}` {
		t.Errorf("omitzero = %s, want {\"name\":\"\"}", b)
	}

	c := config{
		Tags: NewSlice("x", "y"),
		Env:  NewMap(PairOf("K", "V")),
		Name: "n",
		Nest: NewMap(PairOf("nums", NewSlice(1, 2))),
	}
	b, err = json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"tags":["x","y"],"env":{"K":"V"},"name":"n","nest":{"nums":[1,2]}}` {
		t.Errorf("marshal = %s", b)
	}

	var back config
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !EqualValues(back.Tags, c.Tags) || !EqualMapValues(back.Env, c.Env) ||
		!EqualMaps(back.Nest, c.Nest, func(a, b Slice[int]) bool { return EqualValues(a, b) }) {
		t.Errorf("roundtrip mismatch: %+v", back)
	}
}

func TestJSONDoesNotMutateSource(t *testing.T) {
	// Marshaling must not disturb the receiver: marshal twice around a
	// derived value and check both snapshots.
	s := NewSlice(1, 2, 3)
	b1, _ := json.Marshal(s)
	d := s.Append(4)
	b2, _ := json.Marshal(s)
	if string(b1) != "[1,2,3]" || string(b2) != "[1,2,3]" || d.String() != "[1 2 3 4]" {
		t.Errorf("marshal mutated state: %s %s %v", b1, b2, d)
	}

	m := NewMap(PairOf("a", 1))
	m1, _ := json.Marshal(m)
	m.Set("b", 2)
	m2, _ := json.Marshal(m)
	if string(m1) != `{"a":1}` || string(m2) != `{"a":1}` {
		t.Errorf("map marshal changed: %s %s", m1, m2)
	}
}
