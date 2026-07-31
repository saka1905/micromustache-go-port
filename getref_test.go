package micromustache_test

import (
	"errors"
	"reflect"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestGetRef(t *testing.T) {
	tests := []struct {
		name    string
		scope   mm.Scope
		ref     mm.Ref
		options mm.GetOptions
		want    mm.Value
	}{
		{"simple key", mm.Scope{"a": "A"}, mm.Ref{"a"}, mm.GetOptions{}, "A"},
		{"nested Scope", mm.Scope{"a": mm.Scope{"b": 2}}, mm.Ref{"a", "b"}, mm.GetOptions{}, 2},
		{"nested map", mm.Scope{"a": map[string]any{"b": "B"}}, mm.Ref{"a", "b"}, mm.GetOptions{}, "B"},
		{"array index", mm.Scope{"a": []any{"zero", "one"}}, mm.Ref{"a", "1"}, mm.GetOptions{}, "one"},
		{"array length", mm.Scope{"a": []any{"zero", "one"}}, mm.Ref{"a", "length"}, mm.GetOptions{}, 2},
		{"array leading zero is property", mm.Scope{"a": []any{"zero", "one"}}, mm.Ref{"a", "01"}, mm.GetOptions{}, mm.Undefined{}},
		{"array negative is property", mm.Scope{"a": []any{"zero"}}, mm.Ref{"a", "-1"}, mm.GetOptions{}, mm.Undefined{}},
		{"array float is property", mm.Scope{"a": []any{"zero"}}, mm.Ref{"a", "1.0"}, mm.GetOptions{}, mm.Undefined{}},
		{"empty key", mm.Scope{"": "empty"}, mm.Ref{""}, mm.GetOptions{}, "empty"},
		{"own __proto__ key", mm.Scope{"__proto__": "own-proto"}, mm.Ref{"__proto__"}, mm.GetOptions{}, "own-proto"},
		{"own constructor key", mm.Scope{"constructor": "own-constructor"}, mm.Ref{"constructor"}, mm.GetOptions{}, "own-constructor"},
		{"own prototype key", mm.Scope{"prototype": "own-prototype"}, mm.Ref{"prototype"}, mm.GetOptions{}, "own-prototype"},
		{"missing", mm.Scope{}, mm.Ref{"missing"}, mm.GetOptions{}, mm.Undefined{}},
		{"nil scope", nil, mm.Ref{"missing"}, mm.GetOptions{}, mm.Undefined{}},
		{"missing intermediate", mm.Scope{"a": mm.Scope{}}, mm.Ref{"a", "b"}, mm.GetOptions{}, mm.Undefined{}},
		{"nil intermediate", mm.Scope{"a": nil}, mm.Ref{"a", "b"}, mm.GetOptions{}, mm.Undefined{}},
		{"undefined intermediate", mm.Scope{"a": mm.Undefined{}}, mm.Ref{"a", "b"}, mm.GetOptions{}, mm.Undefined{}},
		{"primitive intermediate", mm.Scope{"a": 1}, mm.Ref{"a", "b"}, mm.GetOptions{}, mm.Undefined{}},
		{"terminal undefined with validation", mm.Scope{"v": mm.Undefined{}}, mm.Ref{"v"}, mm.GetOptions{ValidateRef: true}, mm.Undefined{}},
		{"terminal nil", mm.Scope{"v": nil}, mm.Ref{"v"}, mm.GetOptions{}, nil},
		{"terminal false", mm.Scope{"v": false}, mm.Ref{"v"}, mm.GetOptions{}, false},
		{"terminal zero", mm.Scope{"v": 0}, mm.Ref{"v"}, mm.GetOptions{}, 0},
		{"terminal empty string", mm.Scope{"v": ""}, mm.Ref{"v"}, mm.GetOptions{}, ""},
		{"terminal empty array", mm.Scope{"v": []any{}}, mm.Ref{"v"}, mm.GetOptions{}, []any{}},
		{"terminal empty map", mm.Scope{"v": map[string]any{}}, mm.Ref{"v"}, mm.GetOptions{}, map[string]any{}},
		{"slice constructor is not traversed", mm.Scope{"v": []any{}}, mm.Ref{"v", "constructor"}, mm.GetOptions{}, mm.Undefined{}},
		{"struct is not traversed", mm.Scope{"v": struct{ Exported string }{Exported: "value"}}, mm.Ref{"v", "Exported"}, mm.GetOptions{}, mm.Undefined{}},
		{"empty ref", mm.Scope{"a": 1}, mm.Ref{}, mm.GetOptions{}, mm.Scope{"a": 1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for run := 0; run < 2; run++ {
				got, err := mm.GetRef(test.scope, test.ref, test.options)
				if err != nil {
					t.Fatalf("GetRef() error = %v", err)
				}
				if !reflect.DeepEqual(got, test.want) {
					t.Fatalf("GetRef() = %#v, want %#v", got, test.want)
				}
			}
		})
	}
}

func TestGetRefErrors(t *testing.T) {
	tests := []struct {
		name    string
		scope   mm.Scope
		ref     mm.Ref
		options mm.GetOptions
		kind    error
		message string
	}{
		{"missing validation", mm.Scope{}, mm.Ref{"missing"}, mm.GetOptions{ValidateRef: true}, mm.ErrReference, `missing is not defined in the scope at ref: "missing"`},
		{"primitive validation", mm.Scope{"a": 1}, mm.Ref{"a", "b"}, mm.GetOptions{ValidateRef: true}, mm.ErrReference, `b is not defined in the scope at ref: "a > b"`},
		{"depth limit", mm.Scope{}, mm.Ref{"a", "b", "c"}, mm.GetOptions{MaxRefDepth: 2}, mm.ErrReference, `The ref cannot be deeper than 2 levels. Got "a > b > c"`},
		{"default depth limit", mm.Scope{}, mm.Ref{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}, mm.GetOptions{}, mm.ErrReference, `The ref cannot be deeper than 10 levels. Got "a > b > c > d > e > f > g > h > i > j > k"`},
		{"negative depth", mm.Scope{}, mm.Ref{}, mm.GetOptions{MaxRefDepth: -1}, mm.ErrInvalidOption, `Expected a positive number for maxRefDepth. Got -1`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := mm.GetRef(test.scope, test.ref, test.options)
			if got != nil {
				t.Fatalf("GetRef() = %#v, want nil", got)
			}
			if !errors.Is(err, test.kind) {
				t.Fatalf("errors.Is(%v, %v) = false", err, test.kind)
			}
			if err.Error() != test.message {
				t.Fatalf("error = %q, want %q", err, test.message)
			}
		})
	}
}

func TestGetRefDoesNotMutateInput(t *testing.T) {
	scope := mm.Scope{"a": map[string]any{"items": []any{"zero", "one"}}}
	wantScope := mm.Scope{"a": map[string]any{"items": []any{"zero", "one"}}}
	ref := mm.Ref{"a", "items", "1"}
	wantRef := append(mm.Ref(nil), ref...)

	if _, err := mm.GetRef(scope, ref, mm.GetOptions{}); err != nil {
		t.Fatalf("GetRef() error = %v", err)
	}
	if !reflect.DeepEqual(scope, wantScope) {
		t.Fatalf("GetRef() mutated scope: got %#v, want %#v", scope, wantScope)
	}
	if !reflect.DeepEqual(ref, wantRef) {
		t.Fatalf("GetRef() mutated ref: got %#v, want %#v", ref, wantRef)
	}
}
