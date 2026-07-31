package micromustache_test

import (
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestRenderMatchesMeasuredUpstreamCases(t *testing.T) {
	explicit := mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: true}}
	customTags := mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: ">>"}}}

	tests := []struct {
		name     string
		template string
		scope    mm.Scope
		options  mm.CompileOptions
		want     string
	}{
		{"plain text", "plain text", mm.Scope{}, mm.CompileOptions{}, "plain text"},
		{"empty template", "", mm.Scope{}, mm.CompileOptions{}, ""},
		{"string", "Hello {{name}}!", mm.Scope{"name": "Ada"}, mm.CompileOptions{}, "Hello Ada!"},
		{"nested path", "{{user.name}}", mm.Scope{"user": mm.Scope{"name": "Lin"}}, mm.CompileOptions{}, "Lin"},
		{"bracket key", `{{user["first-name"]}}`, mm.Scope{"user": mm.Scope{"first-name": "Jo"}}, mm.CompileOptions{}, "Jo"},
		{"quoted dot key", `{{obj['a.b']}}`, mm.Scope{"obj": mm.Scope{"a.b": "quoted"}}, mm.CompileOptions{}, "quoted"},
		{"array index", "{{items[1]}}", mm.Scope{"items": []any{"zero", "one"}}, mm.CompileOptions{}, "one"},
		{"array length", "{{items.length}}", mm.Scope{"items": []any{"zero", "one"}}, mm.CompileOptions{}, "2"},
		{"repeated path", "{{x}}/{{x}}", mm.Scope{"x": "same"}, mm.CompileOptions{}, "same/same"},
		{"adjacent paths", "{{a}}{{b}}{{c}}", mm.Scope{"a": "A", "b": "B", "c": "C"}, mm.CompileOptions{}, "ABC"},
		{"custom tags", "<<name>> {{name}}", mm.Scope{"name": "tagged"}, customTags, "tagged {{name}}"},
		{"unicode value", "こんにちは {{name}} 🚀", mm.Scope{"name": "世界"}, mm.CompileOptions{}, "こんにちは 世界 🚀"},
		{"quoted unicode key", "こんにちは {{obj['名前']}} 🚀", mm.Scope{"obj": mm.Scope{"名前": "世界"}}, mm.CompileOptions{}, "こんにちは 世界 🚀"},
		{"missing default", "A{{missing}}B", mm.Scope{}, mm.CompileOptions{}, "AB"},
		{"missing explicit", "A{{missing}}B", mm.Scope{}, explicit, "AundefinedB"},
		{"undefined default", "A{{v}}B", mm.Scope{"v": mm.Undefined{}}, mm.CompileOptions{}, "AB"},
		{"undefined explicit", "A{{v}}B", mm.Scope{"v": mm.Undefined{}}, explicit, "AundefinedB"},
		{"null default", "A{{v}}B", mm.Scope{"v": nil}, mm.CompileOptions{}, "AB"},
		{"null explicit", "A{{v}}B", mm.Scope{"v": nil}, explicit, "AnullB"},
		{"true", "{{v}}", mm.Scope{"v": true}, mm.CompileOptions{}, "true"},
		{"false", "{{v}}", mm.Scope{"v": false}, mm.CompileOptions{}, "false"},
		{"zero", "{{v}}", mm.Scope{"v": float64(0)}, mm.CompileOptions{}, "0"},
		{"negative zero", "{{v}}", mm.Scope{"v": math.Copysign(0, -1)}, mm.CompileOptions{}, "0"},
		{"safe integer", "{{v}}", mm.Scope{"v": int64(9007199254740991)}, mm.CompileOptions{}, "9007199254740991"},
		{"negative integer", "{{v}}", mm.Scope{"v": -123456}, mm.CompileOptions{}, "-123456"},
		{"decimal", "{{v}}", mm.Scope{"v": 123.456}, mm.CompileOptions{}, "123.456"},
		{"nan", "{{v}}", mm.Scope{"v": math.NaN()}, mm.CompileOptions{}, "NaN"},
		{"positive infinity", "{{v}}", mm.Scope{"v": math.Inf(1)}, mm.CompileOptions{}, "Infinity"},
		{"negative infinity", "{{v}}", mm.Scope{"v": math.Inf(-1)}, mm.CompileOptions{}, "-Infinity"},
		{"empty string", "A{{v}}B", mm.Scope{"v": ""}, mm.CompileOptions{}, "AB"},
		{"emoji string", "{{v}}", mm.Scope{"v": "🦫✨"}, mm.CompileOptions{}, "🦫✨"},
		{"empty array", "A{{v}}B", mm.Scope{"v": []any{}}, mm.CompileOptions{}, "AB"},
		{"number array", "{{v}}", mm.Scope{"v": []any{1, 2, 3}}, mm.CompileOptions{}, "1,2,3"},
		{"mixed array", "{{v}}", mm.Scope{"v": []any{true, "x", nil, mm.Undefined{}}}, mm.CompileOptions{}, "true,x,,"},
		{"nested array", "{{v}}", mm.Scope{"v": []any{[]any{1, 2}, []any{3}}}, mm.CompileOptions{}, "1,2,3"},
		{"empty object", "{{v}}", mm.Scope{"v": mm.Scope{}}, mm.CompileOptions{}, "[object Object]"},
		{"plain object", "{{v}}", mm.Scope{"v": map[string]any{"a": 1}}, mm.CompileOptions{}, "[object Object]"},
		{"prototype looking object keys", "{{v}}", mm.Scope{"v": map[string]any{"__proto__": "own", "constructor": "own", "prototype": "own"}}, mm.CompileOptions{}, "[object Object]"},
		{"array with object", "{{v}}", mm.Scope{"v": []any{mm.Scope{"a": 1}, []any{"z"}}}, mm.CompileOptions{}, "[object Object],z"},
		{"explicit array null", "{{v}}", mm.Scope{"v": []any{nil, mm.Undefined{}, false}}, explicit, ",,false"},
		{"literal order", "0{{a}}1{{b}}2", mm.Scope{"a": 7, "b": "X"}, mm.CompileOptions{}, "071X2"},
		{"number one e twenty", "{{v}}", mm.Scope{"v": 1e20}, mm.CompileOptions{}, "100000000000000000000"},
		{"number one e twenty one", "{{v}}", mm.Scope{"v": 1e21}, mm.CompileOptions{}, "1e+21"},
		{"number one e minus six", "{{v}}", mm.Scope{"v": 1e-6}, mm.CompileOptions{}, "0.000001"},
		{"number one e minus seven", "{{v}}", mm.Scope{"v": 1e-7}, mm.CompileOptions{}, "1e-7"},
		{"repeated explicit missing", "A{{x}}B{{x}}C", mm.Scope{}, explicit, "AundefinedBundefinedC"},
		{"numeric bracket normalization", "{{items[01]}}", mm.Scope{"items": []any{"zero", "one"}}, mm.CompileOptions{}, "one"},
		{"path whitespace", "{{  user . name  }}", mm.Scope{"user": mm.Scope{"name": "spaced"}}, mm.CompileOptions{}, "spaced"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := mm.Render(test.template, test.scope, test.options)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Render() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRenderPreservesMeasuredErrorsAndOrder(t *testing.T) {
	tests := []struct {
		name     string
		template string
		scope    mm.Scope
		options  mm.CompileOptions
		kind     error
		message  string
	}{
		{
			"invalid tags", "<<name>>", mm.Scope{"name": "Ada"},
			mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: "<<"}}},
			mm.ErrInvalidOption, `The open and close symbols should be two distinct non-empty strings which don't contain each other. Got "<<" and "<<"`,
		},
		{
			"unclosed tag", "before {{name", mm.Scope{"name": "Ada"}, mm.CompileOptions{},
			mm.ErrInvalidTemplate, `Missing "}}" in the template for the "{{" at position 7 within 1000 characters`,
		},
		{
			"invalid path", "{{a.}}", mm.Scope{"a": mm.Scope{}}, mm.CompileOptions{},
			mm.ErrInvalidPath, `Could not parse path: "a."`,
		},
		{
			"invalid path with validation", "{{a.}}", mm.Scope{"a": mm.Scope{}},
			mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}},
			mm.ErrInvalidPath, `Could not parse path: "a."`,
		},
		{
			"missing validated ref", "{{missing}}", mm.Scope{},
			mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{ValidateRef: true}}},
			mm.ErrReference, `missing is not defined in the scope at ref: "missing"`,
		},
		{
			"max path length", "{{abcd}}", mm.Scope{"abcd": "x"},
			mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{MaxPathLen: 3}},
			mm.ErrInvalidTemplate, `Missing "}}" in the template for the "{{" at position 0 within 3 characters`,
		},
		{
			"max ref depth", "{{a.b.c}}", mm.Scope{"a": mm.Scope{"b": mm.Scope{"c": "x"}}},
			mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: 2}}},
			mm.ErrReference, `The ref cannot be deeper than 2 levels. Got "a > b > c"`,
		},
		{
			"all paths parse before lookup", "{{missing}}{{a.}}", mm.Scope{},
			mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{ValidateRef: true}}},
			mm.ErrInvalidPath, `Could not parse path: "a."`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := mm.Render(test.template, test.scope, test.options)
			if got != "" {
				t.Fatalf("Render() result on error = %q, want empty", got)
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

func TestRenderRejectsUnsupportedGoValues(t *testing.T) {
	type record struct{ Value string }
	type namedString string

	cyclic := make([]any, 1)
	cyclic[0] = cyclic
	var tooDeep any = "end"
	for range 1002 {
		tooDeep = []any{tooDeep}
	}
	value := 1
	tests := []struct {
		name  string
		value any
	}{
		{"struct", record{Value: "x"}},
		{"pointer", &value},
		{"function", func() {}},
		{"channel", make(chan int)},
		{"complex", complex(1, 2)},
		{"typed slice", []string{"x"}},
		{"typed map", map[string]string{"x": "y"}},
		{"named scalar", namedString("x")},
		{"unsafe signed integer", int64(9007199254740992)},
		{"unsafe unsigned integer", uint64(9007199254740992)},
		{"cyclic array", cyclic},
		{"excessive array nesting", tooDeep},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := mm.Render("{{v}}", mm.Scope{"v": test.value}, mm.CompileOptions{})
			if got != "" {
				t.Fatalf("Render() result on error = %q, want empty", got)
			}
			if !errors.Is(err, mm.ErrUnsupportedValue) {
				t.Fatalf("errors.Is(%v, ErrUnsupportedValue) = false", err)
			}
		})
	}
}

func TestRenderDoesNotMutateInputs(t *testing.T) {
	template := "{{user.name}}/{{list}}"
	scope := mm.Scope{
		"user": mm.Scope{"name": "Ada"},
		"list": []any{1, nil, mm.Scope{"x": true}},
	}
	want := mm.Scope{
		"user": mm.Scope{"name": "Ada"},
		"list": []any{1, nil, mm.Scope{"x": true}},
	}
	options := mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: true}}

	if _, err := mm.Render(template, scope, options); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if template != "{{user.name}}/{{list}}" {
		t.Fatalf("Render() mutated template: %q", template)
	}
	if !reflect.DeepEqual(scope, want) {
		t.Fatalf("Render() mutated scope: got %#v, want %#v", scope, want)
	}
	if options != (mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: true}}) {
		t.Fatalf("Render() mutated options: %#v", options)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	scope := mm.Scope{"v": map[string]any{"toString": "not a Go method", "z": 1, "a": 2}}
	const template = "before={{v}}=after"

	first, err := mm.Render(template, scope, mm.CompileOptions{})
	if err != nil {
		t.Fatalf("first Render() error = %v", err)
	}
	for iteration := 0; iteration < 100; iteration++ {
		got, err := mm.Render(template, scope, mm.CompileOptions{})
		if err != nil {
			t.Fatalf("Render() iteration %d error = %v", iteration, err)
		}
		if got != first || got != "before=[object Object]=after" {
			t.Fatalf("Render() iteration %d = %q, first = %q", iteration, got, first)
		}
	}
}

func TestRenderIsSafeForConcurrentReadOnlyCalls(t *testing.T) {
	const workers = 64
	scope := mm.Scope{"user": mm.Scope{"name": "Ada"}, "items": []any{1, 2, 3}}
	errorsFound := make(chan error, workers)
	var group sync.WaitGroup

	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			got, err := mm.Render("{{user.name}}={{items}}", scope, mm.CompileOptions{})
			if err != nil {
				errorsFound <- err
				return
			}
			if got != "Ada=1,2,3" {
				errorsFound <- errors.New("unexpected concurrent render result: " + got)
			}
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}
