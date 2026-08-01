package micromustache_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestRendererRenderFuncMatchesTopLevelAndMeasuredCases(t *testing.T) {
	explicit := mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: true}}
	customTags := mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: ">>"}}}
	tests := []struct {
		name      string
		template  string
		values    map[string]mm.Value
		options   mm.CompileOptions
		want      string
		wantPaths []string
	}{
		{"simple", "Hello {{name}}!", map[string]mm.Value{"name": "Ada"}, mm.CompileOptions{}, "Hello Ada!", []string{"name"}},
		{"multiple", "{{a}}/{{b}}/{{c}}", map[string]mm.Value{"a": "A", "b": "B", "c": "C"}, mm.CompileOptions{}, "A/B/C", []string{"a", "b", "c"}},
		{"repeated", "{{x}}-{{x}}-{{x}}", map[string]mm.Value{"x": "X"}, mm.CompileOptions{}, "X-X-X", []string{"x", "x", "x"}},
		{"adjacent", "{{a}}{{b}}{{c}}", map[string]mm.Value{"a": 1, "b": 2, "c": 3}, mm.CompileOptions{}, "123", []string{"a", "b", "c"}},
		{"plain", "plain text", nil, mm.CompileOptions{}, "plain text", nil},
		{"empty", "", nil, mm.CompileOptions{}, "", nil},
		{"custom tags", "<<name>> {{name}}", map[string]mm.Value{"name": "tagged"}, customTags, "tagged {{name}}", []string{"name"}},
		{"unicode", "こんにちは {{name}} 🚀", map[string]mm.Value{"name": "世界"}, mm.CompileOptions{}, "こんにちは 世界 🚀", []string{"name"}},
		{"undefined default", "A{{v}}B", map[string]mm.Value{"v": mm.Undefined{}}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"undefined explicit", "A{{v}}B", map[string]mm.Value{"v": mm.Undefined{}}, explicit, "AundefinedB", []string{"v"}},
		{"nil default", "A{{v}}B", map[string]mm.Value{"v": nil}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"nil explicit", "A{{v}}B", map[string]mm.Value{"v": nil}, explicit, "AnullB", []string{"v"}},
		{"false", "{{v}}", map[string]mm.Value{"v": false}, mm.CompileOptions{}, "false", []string{"v"}},
		{"zero", "{{v}}", map[string]mm.Value{"v": float64(0)}, mm.CompileOptions{}, "0", []string{"v"}},
		{"negative zero", "{{v}}", map[string]mm.Value{"v": math.Copysign(0, -1)}, mm.CompileOptions{}, "0", []string{"v"}},
		{"empty string", "A{{v}}B", map[string]mm.Value{"v": ""}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"integer", "{{v}}", map[string]mm.Value{"v": 42}, mm.CompileOptions{}, "42", []string{"v"}},
		{"decimal", "{{v}}", map[string]mm.Value{"v": 123.456}, mm.CompileOptions{}, "123.456", []string{"v"}},
		{"nan", "{{v}}", map[string]mm.Value{"v": math.NaN()}, mm.CompileOptions{}, "NaN", []string{"v"}},
		{"positive infinity", "{{v}}", map[string]mm.Value{"v": math.Inf(1)}, mm.CompileOptions{}, "Infinity", []string{"v"}},
		{"negative infinity", "{{v}}", map[string]mm.Value{"v": math.Inf(-1)}, mm.CompileOptions{}, "-Infinity", []string{"v"}},
		{"array", "{{v}}", map[string]mm.Value{"v": []any{1, nil, "x"}}, mm.CompileOptions{}, "1,,x", []string{"v"}},
		{"nested array", "{{v}}", map[string]mm.Value{"v": []any{[]any{1, 2}, []any{3}}}, mm.CompileOptions{}, "1,2,3", []string{"v"}},
		{"object", "{{v}}", map[string]mm.Value{"v": map[string]any{"a": 1}}, mm.CompileOptions{}, "[object Object]", []string{"v"}},
		{"dot raw path", "{{user.name}}", map[string]mm.Value{"user.name": "dot"}, mm.CompileOptions{}, "dot", []string{"user.name"}},
		{"bracket raw path", "{{items[1]}}", map[string]mm.Value{"items[1]": "index"}, mm.CompileOptions{}, "index", []string{"items[1]"}},
		{"quoted raw path", "{{obj['a.b']}}", map[string]mm.Value{"obj['a.b']": "quoted"}, mm.CompileOptions{}, "quoted", []string{"obj['a.b']"}},
		{"quoted unicode raw path", "{{obj['名前']}}", map[string]mm.Value{"obj['名前']": "世界"}, mm.CompileOptions{}, "世界", []string{"obj['名前']"}},
		{"trimmed raw path", "{{  user . name  }}", map[string]mm.Value{"user . name": "spaced"}, mm.CompileOptions{}, "spaced", []string{"user . name"}},
		{"invalid raw path unvalidated", "{{a.}}", map[string]mm.Value{"a.": "accepted"}, mm.CompileOptions{}, "accepted", []string{"a."}},
		{"validated path", "{{user.name}}", map[string]mm.Value{"user.name": "validated"}, mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}, "validated", []string{"user.name"}},
		{"missing default", "A{{missing}}B", nil, mm.CompileOptions{}, "AB", []string{"missing"}},
		{"missing explicit", "A{{missing}}B", nil, explicit, "AundefinedB", []string{"missing"}},
		{"validate ref ignored", "{{missing}}", map[string]mm.Value{"missing": "resolver-owned"}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{ValidateRef: true}}}, "resolver-owned", []string{"missing"}},
		{"max ref depth ignored", "{{a.b.c}}", map[string]mm.Value{"a.b.c": "resolver-owned"}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: 1}}}, "resolver-owned", []string{"a.b.c"}},
		{"one e twenty", "{{v}}", map[string]mm.Value{"v": 1e20}, mm.CompileOptions{}, "100000000000000000000", []string{"v"}},
		{"one e twenty one", "{{v}}", map[string]mm.Value{"v": 1e21}, mm.CompileOptions{}, "1e+21", []string{"v"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer, err := mm.Compile(test.template, test.options)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			scopeMarker := &struct{ name string }{name: test.name}
			scope := mm.Scope{"marker": scopeMarker}
			makeResolver := func(paths *[]string) mm.Resolver {
				return func(path string, received mm.Scope) (mm.Value, error) {
					if received["marker"] != scopeMarker {
						t.Fatalf("resolver received different scope for path %q", path)
					}
					*paths = append(*paths, path)
					if value, ok := test.values[path]; ok {
						return value, nil
					}
					return mm.Undefined{}, nil
				}
			}

			var compiledPaths []string
			compiled, err := renderer.RenderFunc(makeResolver(&compiledPaths), scope)
			if err != nil || compiled != test.want {
				t.Fatalf("Renderer.RenderFunc() = %q, %v; want %q", compiled, err, test.want)
			}
			var directPaths []string
			direct, err := mm.RenderFunc(test.template, makeResolver(&directPaths), scope, test.options)
			if err != nil || direct != test.want {
				t.Fatalf("RenderFunc() = %q, %v; want %q", direct, err, test.want)
			}
			if !reflect.DeepEqual(compiledPaths, test.wantPaths) || !reflect.DeepEqual(directPaths, test.wantPaths) {
				t.Fatalf("paths compiled=%#v direct=%#v want=%#v", compiledPaths, directPaths, test.wantPaths)
			}
		})
	}
}

func TestRendererRenderFuncErrorsMatchTopLevelClassification(t *testing.T) {
	sentinel := &compiledResolverError{code: 42}
	tests := []struct {
		name      string
		template  string
		make      func(*[]string) mm.Resolver
		wantKind  error
		wantText  string
		wantPaths []string
	}{
		{"nil resolver", "{{a}}{{b}}{{c}}", func(*[]string) mm.Resolver { return nil }, mm.ErrInvalidResolver, "resolver is nil", nil},
		{"nil resolver empty template", "", func(*[]string) mm.Resolver { return nil }, mm.ErrInvalidResolver, "resolver is nil", nil},
		{"nil resolver plain template", "plain", func(*[]string) mm.Resolver { return nil }, mm.ErrInvalidResolver, "resolver is nil", nil},
		{"first resolver error", "{{a}}{{b}}{{c}}", func(paths *[]string) mm.Resolver {
			return func(path string, _ mm.Scope) (mm.Value, error) { *paths = append(*paths, path); return nil, sentinel }
		}, sentinel, "compiled resolver failure", []string{"a"}},
		{"second resolver error", "{{a}}{{b}}{{c}}", func(paths *[]string) mm.Resolver {
			return func(path string, _ mm.Scope) (mm.Value, error) {
				*paths = append(*paths, path)
				if path == "b" {
					return nil, sentinel
				}
				return path, nil
			}
		}, sentinel, "compiled resolver failure", []string{"a", "b"}},
		{"unsupported first value after all calls", "{{a}}{{b}}{{c}}", func(paths *[]string) mm.Resolver {
			return func(path string, _ mm.Scope) (mm.Value, error) {
				*paths = append(*paths, path)
				if path == "a" {
					return struct{}{}, nil
				}
				return path, nil
			}
		}, mm.ErrUnsupportedValue, "unsupported value", []string{"a", "b", "c"}},
		{"unsupported second value after all calls", "{{a}}{{b}}{{c}}", func(paths *[]string) mm.Resolver {
			return func(path string, _ mm.Scope) (mm.Value, error) {
				*paths = append(*paths, path)
				if path == "b" {
					return struct{}{}, nil
				}
				return path, nil
			}
		}, mm.ErrUnsupportedValue, "unsupported value", []string{"a", "b", "c"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer, err := mm.Compile(test.template, mm.CompileOptions{})
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			var compiledPaths []string
			got, compiledErr := renderer.RenderFunc(test.make(&compiledPaths), mm.Scope{})
			if got != "" || !errors.Is(compiledErr, test.wantKind) || !strings.Contains(compiledErr.Error(), test.wantText) {
				t.Fatalf("Renderer.RenderFunc() = %q, %v", got, compiledErr)
			}
			var directPaths []string
			direct, directErr := mm.RenderFunc(test.template, test.make(&directPaths), mm.Scope{}, mm.CompileOptions{})
			if direct != "" || !errors.Is(directErr, test.wantKind) || !strings.Contains(directErr.Error(), test.wantText) {
				t.Fatalf("RenderFunc() = %q, %v", direct, directErr)
			}
			if !reflect.DeepEqual(compiledPaths, test.wantPaths) || !reflect.DeepEqual(directPaths, test.wantPaths) {
				t.Fatalf("paths compiled=%#v direct=%#v want=%#v", compiledPaths, directPaths, test.wantPaths)
			}
		})
	}
}

func TestCompileRenderFuncErrorStages(t *testing.T) {
	tests := []struct {
		name     string
		template string
		options  mm.CompileOptions
		kind     error
		text     string
	}{
		{"unclosed tag", "before {{a", mm.CompileOptions{}, mm.ErrInvalidTemplate, `Missing "}}"`},
		{"max path length", "{{abcd}}", mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{MaxPathLen: 3}}, mm.ErrInvalidTemplate, "within 3 characters"},
		{"validated invalid path", "{{a.}}", mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}, mm.ErrInvalidPath, `Could not parse path: "a."`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			renderer, err := mm.Compile(test.template, test.options)
			if renderer != nil || !errors.Is(err, test.kind) || !strings.Contains(err.Error(), test.text) {
				t.Fatalf("Compile() = %#v, %v", renderer, err)
			}
			got, directErr := mm.RenderFunc(test.template, func(string, mm.Scope) (mm.Value, error) { calls++; return "unreached", nil }, mm.Scope{}, test.options)
			if got != "" || !errors.Is(directErr, test.kind) || calls != 0 {
				t.Fatalf("RenderFunc() = %q, %v; calls=%d", got, directErr, calls)
			}
		})
	}
}

func TestRendererRenderFuncReusesTemplateNotResolverValues(t *testing.T) {
	renderer, err := mm.Compile("{{value}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	t.Run("different resolvers", func(t *testing.T) {
		for _, want := range []string{"A", "B", "C"} {
			got, err := renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { return want, nil }, mm.Scope{})
			if err != nil || got != want {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		}
	})
	t.Run("same resolver called again", func(t *testing.T) {
		calls := 0
		resolver := func(string, mm.Scope) (mm.Value, error) { calls++; return fmt.Sprint(calls), nil }
		for _, want := range []string{"1", "2", "3"} {
			got, err := renderer.RenderFunc(resolver, mm.Scope{})
			if err != nil || got != want {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		}
	})
	t.Run("missing then present", func(t *testing.T) {
		value := mm.Value(mm.Undefined{})
		resolver := func(string, mm.Scope) (mm.Value, error) { return value, nil }
		if got, _ := renderer.RenderFunc(resolver, mm.Scope{}); got != "" {
			t.Fatalf("missing = %q", got)
		}
		value = "present"
		if got, _ := renderer.RenderFunc(resolver, mm.Scope{}); got != "present" {
			t.Fatalf("present = %q", got)
		}
	})
	t.Run("present then missing", func(t *testing.T) {
		value := mm.Value("present")
		resolver := func(string, mm.Scope) (mm.Value, error) { return value, nil }
		if got, _ := renderer.RenderFunc(resolver, mm.Scope{}); got != "present" {
			t.Fatalf("present = %q", got)
		}
		value = mm.Undefined{}
		if got, _ := renderer.RenderFunc(resolver, mm.Scope{}); got != "" {
			t.Fatalf("missing = %q", got)
		}
	})
	t.Run("resolver error then success", func(t *testing.T) {
		if got, err := renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { return nil, errors.New("failure") }, mm.Scope{}); got != "" || err == nil {
			t.Fatalf("error call = %q, %v", got, err)
		}
		if got, err := renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { return "recovered", nil }, mm.Scope{}); got != "recovered" || err != nil {
			t.Fatalf("recovery = %q, %v", got, err)
		}
	})
	t.Run("unsupported value then success", func(t *testing.T) {
		if got, err := renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { return struct{}{}, nil }, mm.Scope{}); got != "" || !errors.Is(err, mm.ErrUnsupportedValue) {
			t.Fatalf("unsupported = %q, %v", got, err)
		}
		if got, err := renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { return "recovered", nil }, mm.Scope{}); got != "recovered" || err != nil {
			t.Fatalf("recovery = %q, %v", got, err)
		}
	})
}

func TestRendererRenderFuncIgnoresLazyParsedRefFailure(t *testing.T) {
	renderer, err := mm.Compile("{{a.}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if got, err := renderer.Render(mm.Scope{}); got != "" || !errors.Is(err, mm.ErrInvalidPath) {
		t.Fatalf("Renderer.Render() = %q, %v", got, err)
	}
	paths := []string{}
	got, err := renderer.RenderFunc(func(path string, _ mm.Scope) (mm.Value, error) { paths = append(paths, path); return "raw", nil }, mm.Scope{})
	if err != nil || got != "raw" || !reflect.DeepEqual(paths, []string{"a."}) {
		t.Fatalf("Renderer.RenderFunc() = %q, %v; paths=%#v", got, err, paths)
	}
}

func TestRendererRenderFuncPreservesTypedResolverError(t *testing.T) {
	renderer, err := mm.Compile("{{user.name}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	want := &compiledResolverError{code: 7}
	got, err := renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { return nil, want }, mm.Scope{})
	if got != "" || !errors.Is(err, want) {
		t.Fatalf("Renderer.RenderFunc() = %q, %v", got, err)
	}
	var typed *compiledResolverError
	if !errors.As(err, &typed) || typed != want {
		t.Fatalf("errors.As(%v) did not preserve resolver error", err)
	}
}

func TestRendererRenderFuncRejectsInvalidReceivers(t *testing.T) {
	tests := []struct {
		name     string
		renderer *mm.Renderer
	}{{"nil", nil}, {"zero", &mm.Renderer{}}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			got, err := test.renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { called = true; return "x", nil }, mm.Scope{})
			if got != "" || !errors.Is(err, mm.ErrInvalidRenderer) || called {
				t.Fatalf("RenderFunc() = %q, %v; called=%t", got, err, called)
			}
		})
	}
}

func TestRendererRenderFuncDoesNotMutateInputsAndIsDeterministic(t *testing.T) {
	renderer, err := mm.Compile("{{first}}/{{second}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	returned := []any{1, nil, map[string]any{"x": true}}
	wantReturned := []any{1, nil, map[string]any{"x": true}}
	scope := mm.Scope{"marker": "unchanged"}
	wantScope := mm.Scope{"marker": "unchanged"}
	resolver := func(path string, received mm.Scope) (mm.Value, error) {
		if !reflect.DeepEqual(received, wantScope) {
			t.Fatalf("scope = %#v", received)
		}
		if path == "first" {
			return returned, nil
		}
		return "done", nil
	}
	for iteration := 0; iteration < 100; iteration++ {
		got, err := renderer.RenderFunc(resolver, scope)
		if err != nil || got != "1,,[object Object]/done" {
			t.Fatalf("iteration %d = %q, %v", iteration, got, err)
		}
	}
	if !reflect.DeepEqual(scope, wantScope) || !reflect.DeepEqual(returned, wantReturned) {
		t.Fatalf("inputs mutated: scope=%#v returned=%#v", scope, returned)
	}
}

func TestRendererRenderFuncIsSafeForConcurrentReadOnlyCalls(t *testing.T) {
	renderer, err := mm.Compile("{{a}}/{{b}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	const workers = 64
	scope := mm.Scope{"prefix": "value"}
	errorsFound := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			got, err := renderer.RenderFunc(func(path string, received mm.Scope) (mm.Value, error) {
				return received["prefix"].(string) + ":" + path, nil
			}, scope)
			if err != nil {
				errorsFound <- err
			} else if got != "value:a/value:b" {
				errorsFound <- fmt.Errorf("unexpected result %q", got)
			}
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

type compiledResolverError struct{ code int }

func (e *compiledResolverError) Error() string { return "compiled resolver failure" }
