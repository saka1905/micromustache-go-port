package micromustache_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestRenderFuncMatchesMeasuredUpstreamCases(t *testing.T) {
	explicit := mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: true}}
	customTags := mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: ">>"}}}

	tests := []struct {
		name      string
		template  string
		values    map[string]mm.Value
		fallback  mm.Value
		options   mm.CompileOptions
		want      string
		wantPaths []string
	}{
		{"simple", "Hello {{name}}!", map[string]mm.Value{"name": "Ada"}, mm.Undefined{}, mm.CompileOptions{}, "Hello Ada!", []string{"name"}},
		{"multiple", "{{a}}/{{b}}/{{c}}", map[string]mm.Value{"a": "A", "b": "B", "c": "C"}, mm.Undefined{}, mm.CompileOptions{}, "A/B/C", []string{"a", "b", "c"}},
		{"repeated", "{{x}}-{{x}}-{{x}}", map[string]mm.Value{"x": "X"}, mm.Undefined{}, mm.CompileOptions{}, "X-X-X", []string{"x", "x", "x"}},
		{"adjacent", "{{a}}{{b}}{{c}}", map[string]mm.Value{"a": 1, "b": 2, "c": 3}, mm.Undefined{}, mm.CompileOptions{}, "123", []string{"a", "b", "c"}},
		{"plain", "plain text", nil, mm.Undefined{}, mm.CompileOptions{}, "plain text", nil},
		{"empty", "", nil, mm.Undefined{}, mm.CompileOptions{}, "", nil},
		{"custom tags", "<<name>> {{name}}", map[string]mm.Value{"name": "tagged"}, mm.Undefined{}, customTags, "tagged {{name}}", []string{"name"}},
		{"unicode", "こんにちは {{name}} 🚀", map[string]mm.Value{"name": "世界"}, mm.Undefined{}, mm.CompileOptions{}, "こんにちは 世界 🚀", []string{"name"}},
		{"undefined default", "A{{v}}B", map[string]mm.Value{"v": mm.Undefined{}}, mm.Undefined{}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"undefined explicit", "A{{v}}B", map[string]mm.Value{"v": mm.Undefined{}}, mm.Undefined{}, explicit, "AundefinedB", []string{"v"}},
		{"nil default", "A{{v}}B", map[string]mm.Value{"v": nil}, mm.Undefined{}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"nil explicit", "A{{v}}B", map[string]mm.Value{"v": nil}, mm.Undefined{}, explicit, "AnullB", []string{"v"}},
		{"false", "{{v}}", map[string]mm.Value{"v": false}, mm.Undefined{}, mm.CompileOptions{}, "false", []string{"v"}},
		{"zero", "{{v}}", map[string]mm.Value{"v": float64(0)}, mm.Undefined{}, mm.CompileOptions{}, "0", []string{"v"}},
		{"negative zero", "{{v}}", map[string]mm.Value{"v": math.Copysign(0, -1)}, mm.Undefined{}, mm.CompileOptions{}, "0", []string{"v"}},
		{"empty string", "A{{v}}B", map[string]mm.Value{"v": ""}, mm.Undefined{}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"integer", "{{v}}", map[string]mm.Value{"v": 42}, mm.Undefined{}, mm.CompileOptions{}, "42", []string{"v"}},
		{"decimal", "{{v}}", map[string]mm.Value{"v": 123.456}, mm.Undefined{}, mm.CompileOptions{}, "123.456", []string{"v"}},
		{"nan", "{{v}}", map[string]mm.Value{"v": math.NaN()}, mm.Undefined{}, mm.CompileOptions{}, "NaN", []string{"v"}},
		{"positive infinity", "{{v}}", map[string]mm.Value{"v": math.Inf(1)}, mm.Undefined{}, mm.CompileOptions{}, "Infinity", []string{"v"}},
		{"negative infinity", "{{v}}", map[string]mm.Value{"v": math.Inf(-1)}, mm.Undefined{}, mm.CompileOptions{}, "-Infinity", []string{"v"}},
		{"normal string", "{{v}}", map[string]mm.Value{"v": "text"}, mm.Undefined{}, mm.CompileOptions{}, "text", []string{"v"}},
		{"emoji string", "{{v}}", map[string]mm.Value{"v": "🦫✨"}, mm.Undefined{}, mm.CompileOptions{}, "🦫✨", []string{"v"}},
		{"array", "{{v}}", map[string]mm.Value{"v": []any{1, nil, "x"}}, mm.Undefined{}, mm.CompileOptions{}, "1,,x", []string{"v"}},
		{"nested array", "{{v}}", map[string]mm.Value{"v": []any{[]any{1, 2}, []any{3}}}, mm.Undefined{}, mm.CompileOptions{}, "1,2,3", []string{"v"}},
		{"object", "{{v}}", map[string]mm.Value{"v": map[string]any{"a": 1}}, mm.Undefined{}, mm.CompileOptions{}, "[object Object]", []string{"v"}},
		{"nested looking raw path", "{{user.name}}", map[string]mm.Value{"user.name": "raw-dot"}, mm.Undefined{}, mm.CompileOptions{}, "raw-dot", []string{"user.name"}},
		{"quoted raw path", "{{obj['a.b']}}", map[string]mm.Value{"obj['a.b']": "raw-bracket"}, mm.Undefined{}, mm.CompileOptions{}, "raw-bracket", []string{"obj['a.b']"}},
		{"invalid raw path without validation", "{{a.}}", map[string]mm.Value{"a.": "raw-invalid"}, mm.Undefined{}, mm.CompileOptions{}, "raw-invalid", []string{"a."}},
		{"missing default", "A{{missing}}B", nil, mm.Undefined{}, mm.CompileOptions{}, "AB", []string{"missing"}},
		{"missing explicit", "A{{missing}}B", nil, mm.Undefined{}, explicit, "AundefinedB", []string{"missing"}},
		{"quoted unicode raw path", "{{obj['名前']}}", map[string]mm.Value{"obj['名前']": "世界"}, mm.Undefined{}, mm.CompileOptions{}, "世界", []string{"obj['名前']"}},
		{"trimmed path", "{{  user . name  }}", map[string]mm.Value{"user . name": "spaced"}, mm.Undefined{}, mm.CompileOptions{}, "spaced", []string{"user . name"}},
		{"validated path", "{{user.name}}", map[string]mm.Value{"user.name": "validated"}, mm.Undefined{}, mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}, "validated", []string{"user.name"}},
		{"validate ref ignored", "{{missing}}", map[string]mm.Value{"missing": "resolver-owned"}, mm.Undefined{}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{ValidateRef: true}}}, "resolver-owned", []string{"missing"}},
		{"max ref depth ignored", "{{a.b.c}}", map[string]mm.Value{"a.b.c": "resolver-owned"}, mm.Undefined{}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: 1}}}, "resolver-owned", []string{"a.b.c"}},
		{"one e twenty", "{{v}}", map[string]mm.Value{"v": 1e20}, mm.Undefined{}, mm.CompileOptions{}, "100000000000000000000", []string{"v"}},
		{"one e twenty one", "{{v}}", map[string]mm.Value{"v": 1e21}, mm.Undefined{}, mm.CompileOptions{}, "1e+21", []string{"v"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopeMarker := &struct{ name string }{name: test.name}
			scope := mm.Scope{"marker": scopeMarker}
			var gotPaths []string
			resolver := func(path string, receivedScope mm.Scope) (mm.Value, error) {
				if receivedScope["marker"] != scopeMarker {
					t.Fatalf("resolver received a different scope for path %q", path)
				}
				gotPaths = append(gotPaths, path)
				if value, ok := test.values[path]; ok {
					return value, nil
				}
				return test.fallback, nil
			}

			got, err := mm.RenderFunc(test.template, resolver, scope, test.options)
			if err != nil {
				t.Fatalf("RenderFunc() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("RenderFunc() = %q, want %q", got, test.want)
			}
			if !reflect.DeepEqual(gotPaths, test.wantPaths) {
				t.Fatalf("resolver paths = %#v, want %#v", gotPaths, test.wantPaths)
			}
		})
	}
}

func TestRenderFuncErrorsAndStopping(t *testing.T) {
	sentinel := errors.New("resolver sentinel")
	tests := []struct {
		name      string
		template  string
		options   mm.CompileOptions
		resolver  mm.Resolver
		wantKind  error
		wantText  string
		wantPaths []string
	}{
		{"nil resolver", "{{a}}", mm.CompileOptions{}, nil, mm.ErrInvalidResolver, "resolver is nil", nil},
		{"nil resolver without interpolation", "plain", mm.CompileOptions{}, nil, mm.ErrInvalidResolver, "resolver is nil", nil},
		{"nil resolver with empty template", "", mm.CompileOptions{}, nil, mm.ErrInvalidResolver, "resolver is nil", nil},
		{"unclosed before nil resolver", "before {{a", mm.CompileOptions{}, nil, mm.ErrInvalidTemplate, `Missing "}}"`, nil},
		{"invalid path before nil resolver when validated", "{{a.}}", mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}, nil, mm.ErrInvalidPath, `Could not parse path: "a."`, nil},
		{"invalid path validated", "{{a.}}{{b}}", mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}, countingResolver(nil, nil), mm.ErrInvalidPath, `Could not parse path: "a."`, nil},
		{"unclosed tag", "before {{a", mm.CompileOptions{}, countingResolver(nil, nil), mm.ErrInvalidTemplate, `Missing "}}"`, nil},
		{"max path length", "{{abcd}}", mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{MaxPathLen: 3}}, countingResolver(nil, nil), mm.ErrInvalidTemplate, `within 3 characters`, nil},
		{"first resolver error", "{{a}}{{b}}{{c}}", mm.CompileOptions{}, nil, sentinel, "resolver sentinel", []string{"a"}},
		{"second resolver error", "{{a}}{{b}}{{c}}", mm.CompileOptions{}, nil, sentinel, "resolver sentinel", []string{"a", "b"}},
		{"unsupported returned value", "{{bad}}", mm.CompileOptions{}, func(string, mm.Scope) (mm.Value, error) { return struct{}{}, nil }, mm.ErrUnsupportedValue, "unsupported value", []string{"bad"}},
		{"all calls before unsupported stringify", "{{bad}}{{later}}", mm.CompileOptions{}, nil, mm.ErrUnsupportedValue, "unsupported value", []string{"bad", "later"}},
		{"resolver error before stringify", "{{bad}}{{later}}", mm.CompileOptions{}, nil, sentinel, "resolver sentinel", []string{"bad", "later"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotPaths []string
			resolver := test.resolver
			if resolver != nil {
				original := resolver
				resolver = func(path string, scope mm.Scope) (mm.Value, error) {
					gotPaths = append(gotPaths, path)
					return original(path, scope)
				}
			}
			switch test.name {
			case "first resolver error":
				resolver = func(path string, _ mm.Scope) (mm.Value, error) {
					gotPaths = append(gotPaths, path)
					return nil, sentinel
				}
			case "second resolver error":
				resolver = func(path string, _ mm.Scope) (mm.Value, error) {
					gotPaths = append(gotPaths, path)
					if path == "b" {
						return nil, sentinel
					}
					return "A", nil
				}
			case "resolver error before stringify":
				resolver = func(path string, _ mm.Scope) (mm.Value, error) {
					gotPaths = append(gotPaths, path)
					if path == "later" {
						return nil, sentinel
					}
					return struct{}{}, nil
				}
			case "all calls before unsupported stringify":
				resolver = func(path string, _ mm.Scope) (mm.Value, error) {
					gotPaths = append(gotPaths, path)
					if path == "bad" {
						return struct{}{}, nil
					}
					return "later", nil
				}
			}

			got, err := mm.RenderFunc(test.template, resolver, mm.Scope{}, test.options)
			if got != "" {
				t.Fatalf("RenderFunc() result on error = %q, want empty", got)
			}
			if !errors.Is(err, test.wantKind) {
				t.Fatalf("errors.Is(%v, %v) = false", err, test.wantKind)
			}
			if !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("error = %q, want substring %q", err, test.wantText)
			}
			if !reflect.DeepEqual(gotPaths, test.wantPaths) {
				t.Fatalf("resolver paths = %#v, want %#v", gotPaths, test.wantPaths)
			}
		})
	}
}

func TestRenderFuncWrapsResolverError(t *testing.T) {
	want := &typedResolverError{code: 42}
	got, err := mm.RenderFunc("{{user.name}}", func(string, mm.Scope) (mm.Value, error) {
		return nil, want
	}, mm.Scope{}, mm.CompileOptions{})
	if got != "" {
		t.Fatalf("RenderFunc() result on error = %q, want empty", got)
	}
	if !errors.Is(err, want) {
		t.Fatalf("errors.Is(%v, want) = false", err)
	}
	var typed *typedResolverError
	if !errors.As(err, &typed) || typed != want {
		t.Fatalf("errors.As(%v) did not preserve resolver error", err)
	}
	if !strings.Contains(err.Error(), `path "user.name" at index 0`) || !strings.Contains(err.Error(), want.Error()) {
		t.Fatalf("wrapped error lost context or original message: %v", err)
	}
}

func TestRenderAndRenderFuncShareStringification(t *testing.T) {
	values := []mm.Value{
		mm.Undefined{}, nil, false, 0, math.Copysign(0, -1), 123.456,
		math.NaN(), math.Inf(1), "text", []any{1, nil, "x"}, map[string]any{"a": 1},
	}
	for index, value := range values {
		for _, explicit := range []bool{false, true} {
			options := mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: explicit}}
			fromRender, err := mm.Render("{{v}}", mm.Scope{"v": value}, options)
			if err != nil {
				t.Fatalf("Render(value %d, explicit %t) error = %v", index, explicit, err)
			}
			fromResolver, err := mm.RenderFunc("{{v}}", func(string, mm.Scope) (mm.Value, error) {
				return value, nil
			}, mm.Scope{}, options)
			if err != nil {
				t.Fatalf("RenderFunc(value %d, explicit %t) error = %v", index, explicit, err)
			}
			if fromResolver != fromRender {
				t.Fatalf("value %d explicit %t: RenderFunc = %q, Render = %q", index, explicit, fromResolver, fromRender)
			}
		}
	}
}

func TestRenderFuncDoesNotMutateInputs(t *testing.T) {
	template := "{{first}}/{{second}}"
	returned := []any{1, nil, map[string]any{"x": true}}
	wantReturned := []any{1, nil, map[string]any{"x": true}}
	scope := mm.Scope{"marker": "unchanged"}
	wantScope := mm.Scope{"marker": "unchanged"}

	got, err := mm.RenderFunc(template, func(path string, receivedScope mm.Scope) (mm.Value, error) {
		if !reflect.DeepEqual(receivedScope, wantScope) {
			t.Fatalf("resolver scope = %#v, want %#v", receivedScope, wantScope)
		}
		if path == "first" {
			return returned, nil
		}
		return "done", nil
	}, scope, mm.CompileOptions{})
	if err != nil {
		t.Fatalf("RenderFunc() error = %v", err)
	}
	if got != "1,,[object Object]/done" {
		t.Fatalf("RenderFunc() = %q", got)
	}
	if template != "{{first}}/{{second}}" || !reflect.DeepEqual(scope, wantScope) || !reflect.DeepEqual(returned, wantReturned) {
		t.Fatalf("RenderFunc mutated input: template=%q scope=%#v returned=%#v", template, scope, returned)
	}
}

func TestRenderFuncIsDeterministic(t *testing.T) {
	resolver := func(path string, _ mm.Scope) (mm.Value, error) {
		return map[string]any{"z": path, "a": 1}, nil
	}
	const template = "{{a}}/{{b}}"
	first, err := mm.RenderFunc(template, resolver, mm.Scope{}, mm.CompileOptions{})
	if err != nil {
		t.Fatalf("first RenderFunc() error = %v", err)
	}
	for iteration := 0; iteration < 100; iteration++ {
		got, err := mm.RenderFunc(template, resolver, mm.Scope{}, mm.CompileOptions{})
		if err != nil || got != first {
			t.Fatalf("iteration %d: got %q, err %v; first %q", iteration, got, err, first)
		}
	}
}

func TestRenderFuncIsSafeForConcurrentReadOnlyCalls(t *testing.T) {
	const workers = 64
	scope := mm.Scope{"prefix": "value"}
	resolver := func(path string, receivedScope mm.Scope) (mm.Value, error) {
		return receivedScope["prefix"].(string) + ":" + path, nil
	}
	errorsFound := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			got, err := mm.RenderFunc("{{a}}/{{b}}", resolver, scope, mm.CompileOptions{})
			if err != nil {
				errorsFound <- err
			} else if got != "value:a/value:b" {
				errorsFound <- errors.New("unexpected concurrent RenderFunc result: " + got)
			}
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

func countingResolver(paths *[]string, failure error) mm.Resolver {
	return func(path string, _ mm.Scope) (mm.Value, error) {
		if paths != nil {
			*paths = append(*paths, path)
		}
		if failure != nil {
			return nil, failure
		}
		return path, nil
	}
}

type typedResolverError struct {
	code int
}

func (e *typedResolverError) Error() string {
	return "typed resolver failure"
}
