package micromustache_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		scope   mm.Scope
		path    string
		options mm.GetOptions
		want    mm.Value
	}{
		{"simple key", mm.Scope{"a": "A"}, "a", mm.GetOptions{}, "A"},
		{"nested dot", mm.Scope{"a": mm.Scope{"b": 2}}, "a.b", mm.GetOptions{}, 2},
		{"leading dot", mm.Scope{"a": "A"}, " . a ", mm.GetOptions{}, "A"},
		{"double quoted key", mm.Scope{"a.b": "dot"}, `["a.b"]`, mm.GetOptions{}, "dot"},
		{"single quoted key", mm.Scope{"a]b": "bracket"}, `['a]b']`, mm.GetOptions{}, "bracket"},
		{"backtick quoted key", mm.Scope{"a": "A"}, "[`a`]", mm.GetOptions{}, "A"},
		{"empty quoted key", mm.Scope{"": "empty"}, `['']`, mm.GetOptions{}, "empty"},
		{"literal backslash", mm.Scope{`a\b`: "slash"}, `['a\b']`, mm.GetOptions{}, "slash"},
		{"quote is not unescaped", mm.Scope{`a\'b`: "literal"}, `['a\'b']`, mm.GetOptions{}, "literal"},
		{"array index", mm.Scope{"a": []any{"zero", "one"}}, "a[1]", mm.GetOptions{}, "one"},
		{"array leading zeros normalize", mm.Scope{"a": []any{"zero", "one"}}, "a[01]", mm.GetOptions{}, "one"},
		{"array plus and spaces normalize", mm.Scope{"a": []any{"zero", "one"}}, "a[ + 01 ]", mm.GetOptions{}, "one"},
		{"quoted leading zero stays property", mm.Scope{"a": []any{"zero", "one"}}, `a['01']`, mm.GetOptions{}, mm.Undefined{}},
		{"map leading zero key", mm.Scope{"a": map[string]any{"01": "leading"}}, `a['01']`, mm.GetOptions{}, "leading"},
		{"map negative key", mm.Scope{"a": map[string]any{"-1": "negative"}}, `a['-1']`, mm.GetOptions{}, "negative"},
		{"map float key", mm.Scope{"a": map[string]any{"1.0": "float"}}, `a['1.0']`, mm.GetOptions{}, "float"},
		{"unicode quoted key", mm.Scope{"😀": "emoji"}, `['😀']`, mm.GetOptions{}, "emoji"},
		{"emoji inside quoted key", mm.Scope{"a😀b": "surrogate"}, `['a😀b']`, mm.GetOptions{}, "surrogate"},
		{"mixed dot and bracket", mm.Scope{"a": map[string]any{"b": []any{"zero", map[string]any{"c": 3}}}}, `a["b"][1].c`, mm.GetOptions{}, 3},
		{"sixteen digit normalized key", mm.Scope{"a": map[string]any{"1234567890123456": "sixteen"}}, "a[0001234567890123456]", mm.GetOptions{}, "sixteen"},
		{"missing", mm.Scope{}, "missing", mm.GetOptions{}, mm.Undefined{}},
		{"terminal nil", mm.Scope{"v": nil}, "v", mm.GetOptions{}, nil},
		{"terminal undefined", mm.Scope{"v": mm.Undefined{}}, "v", mm.GetOptions{ValidateRef: true}, mm.Undefined{}},
		{"empty path returns scope", mm.Scope{"a": 1}, "", mm.GetOptions{}, mm.Scope{"a": 1}},
		{"array length", mm.Scope{"a": []any{"zero", "one"}}, "a.length", mm.GetOptions{}, 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for run := 0; run < 2; run++ {
				got, err := mm.Get(test.scope, test.path, test.options)
				if err != nil {
					t.Fatalf("Get() error = %v", err)
				}
				if !reflect.DeepEqual(got, test.want) {
					t.Fatalf("Get() = %#v, want %#v", got, test.want)
				}
			}
		})
	}
}

func TestGetPathErrors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		options mm.GetOptions
		kind    error
		message string
	}{
		{"trailing dot", "a.", mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a."`},
		{"empty segment", "a..b", mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a..b"`},
		{"malformed bracket", "a[", mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a["`},
		{"unclosed quote", `a['b]`, mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a['b]"`},
		{"negative numeric index", "a[-1]", mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a[-1]"`},
		{"floating numeric index", "a[1.0]", mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a[1.0]"`},
		{"bare Unicode", "😀", mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "😀"`},
		{"quoted newline", "['a\nb']", mm.GetOptions{}, mm.ErrInvalidPath, "Could not parse path: \"['a\nb']\""},
		{"seventeen digit index", "a[12345678901234567]", mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a[12345678901234567]"`},
		{"dot before bracket", `.["a"]`, mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: ".["a"]"`},
		{"segment adjacency", `a["b"]c`, mm.GetOptions{}, mm.ErrInvalidPath, `Could not parse path: "a["b"]c"`},
		{"missing validation", "missing", mm.GetOptions{ValidateRef: true}, mm.ErrReference, `missing is not defined in the scope at ref: "missing"`},
		{"depth validation", "a.b.c", mm.GetOptions{MaxRefDepth: 2}, mm.ErrReference, `The ref cannot be deeper than 2 levels. Got "a > b > c"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := mm.Get(mm.Scope{}, test.path, test.options)
			if got != nil {
				t.Fatalf("Get() = %#v, want nil", got)
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

func TestGetHasNoPathLengthLimit(t *testing.T) {
	path := strings.Repeat("a", 5000)
	got, err := mm.Get(mm.Scope{}, path, mm.GetOptions{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !reflect.DeepEqual(got, mm.Undefined{}) {
		t.Fatalf("Get() = %#v, want Undefined{}", got)
	}
}

func TestGetDoesNotMutateInput(t *testing.T) {
	scope := mm.Scope{"a": map[string]any{"items": []any{"zero", "one"}}}
	want := mm.Scope{"a": map[string]any{"items": []any{"zero", "one"}}}
	if _, err := mm.Get(scope, "a.items[1]", mm.GetOptions{}); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !reflect.DeepEqual(scope, want) {
		t.Fatalf("Get() mutated scope: got %#v, want %#v", scope, want)
	}
}
