package micromustache_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestRendererRenderFuncAsyncNormalCases(t *testing.T) {
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
		{"empty", "", nil, mm.CompileOptions{}, "", nil},
		{"no interpolation", "plain text", nil, mm.CompileOptions{}, "plain text", nil},
		{"custom tags", "<<name>> {{name}}", map[string]mm.Value{"name": "tagged"}, customTags, "tagged {{name}}", []string{"name"}},
		{"unicode", "こんにちは {{name}} 🚀", map[string]mm.Value{"name": "世界"}, mm.CompileOptions{}, "こんにちは 世界 🚀", []string{"name"}},
		{"undefined", "A{{v}}B", map[string]mm.Value{"v": mm.Undefined{}}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"undefined explicit", "A{{v}}B", map[string]mm.Value{"v": mm.Undefined{}}, explicit, "AundefinedB", []string{"v"}},
		{"nil", "A{{v}}B", map[string]mm.Value{"v": nil}, mm.CompileOptions{}, "AB", []string{"v"}},
		{"nil explicit", "A{{v}}B", map[string]mm.Value{"v": nil}, explicit, "AnullB", []string{"v"}},
		{"false", "{{v}}", map[string]mm.Value{"v": false}, mm.CompileOptions{}, "false", []string{"v"}},
		{"zero", "{{v}}", map[string]mm.Value{"v": 0}, mm.CompileOptions{}, "0", []string{"v"}},
		{"string", "{{v}}", map[string]mm.Value{"v": "text"}, mm.CompileOptions{}, "text", []string{"v"}},
		{"array", "{{v}}", map[string]mm.Value{"v": []any{1, nil, "x"}}, mm.CompileOptions{}, "1,,x", []string{"v"}},
		{"map", "{{v}}", map[string]mm.Value{"v": map[string]any{"a": 1}}, mm.CompileOptions{}, "[object Object]", []string{"v"}},
		{"raw dot path", "{{user.name}}", map[string]mm.Value{"user.name": "dot"}, mm.CompileOptions{}, "dot", []string{"user.name"}},
		{"trimmed raw path", "{{  user . name  }}", map[string]mm.Value{"user . name": "spaced"}, mm.CompileOptions{}, "spaced", []string{"user . name"}},
		{"invalid path without validation", "{{a.}}", map[string]mm.Value{"a.": "raw"}, mm.CompileOptions{}, "raw", []string{"a."}},
		{"max ref depth ignored", "{{a.b.c}}", map[string]mm.Value{"a.b.c": "resolver-owned"}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: 1}}}, "resolver-owned", []string{"a.b.c"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer, err := mm.Compile(test.template, test.options)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			marker := &struct{ name string }{name: test.name}
			scope := mm.Scope{"marker": marker}
			var mutex sync.Mutex
			var paths []string
			got, err := renderer.RenderFuncAsync(context.Background(), func(ctx context.Context, path string, received mm.Scope) (mm.Value, error) {
				if ctx == nil || received["marker"] != marker {
					return nil, errors.New("resolver arguments changed")
				}
				mutex.Lock()
				paths = append(paths, path)
				mutex.Unlock()
				if value, ok := test.values[path]; ok {
					return value, nil
				}
				return mm.Undefined{}, nil
			}, scope)
			if err != nil || got != test.want {
				t.Fatalf("Renderer.RenderFuncAsync() = %q, %v; want %q", got, err, test.want)
			}
			if !sameStringCounts(paths, test.wantPaths) {
				t.Fatalf("paths = %#v; want occurrences %#v", paths, test.wantPaths)
			}
		})
	}
}

func TestRendererRenderFuncAsyncConcurrencyContract(t *testing.T) {
	renderer, err := mm.Compile("{{a}}/{{b}}/{{c}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	t.Run("all start before completion", func(t *testing.T) {
		started := make(chan string, 3)
		release := make(chan struct{})
		result := make(chan asyncCallResult, 1)
		go func() {
			got, callErr := renderer.RenderFuncAsync(context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
				started <- path
				<-release
				return strings.ToUpper(path), nil
			}, mm.Scope{})
			result <- asyncCallResult{value: got, err: callErr}
		}()
		paths := []string{receiveString(t, started), receiveString(t, started), receiveString(t, started)}
		if !sameStringCounts(paths, []string{"a", "b", "c"}) {
			t.Fatalf("started paths = %#v", paths)
		}
		close(release)
		assertAsyncResult(t, result, "A/B/C", nil)
	})

	t.Run("reverse completion preserves template order", func(t *testing.T) {
		got := runCompiledAsyncInCompletionOrder(t, renderer, []string{"c", "b", "a"})
		if got != "A/B/C" {
			t.Fatalf("result = %q", got)
		}
	})

	t.Run("repeated occurrences run independently", func(t *testing.T) {
		repeated, compileErr := mm.Compile("{{x}}/{{x}}/{{x}}", mm.CompileOptions{})
		if compileErr != nil {
			t.Fatalf("Compile() error = %v", compileErr)
		}
		var mutex sync.Mutex
		calls := 0
		got, callErr := repeated.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) {
			mutex.Lock()
			calls++
			mutex.Unlock()
			return "X", nil
		}, mm.Scope{})
		if callErr != nil || got != "X/X/X" || calls != 3 {
			t.Fatalf("result = %q, %v; calls=%d", got, callErr, calls)
		}
	})

	t.Run("shared completion gate preserves index order", func(t *testing.T) {
		started := make(chan struct{}, 3)
		release := make(chan struct{})
		result := make(chan asyncCallResult, 1)
		go func() {
			got, callErr := renderer.RenderFuncAsync(context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
				started <- struct{}{}
				<-release
				return strings.ToUpper(path), nil
			}, mm.Scope{})
			result <- asyncCallResult{value: got, err: callErr}
		}()
		for range 3 {
			receiveSignal(t, started)
		}
		close(release)
		assertAsyncResult(t, result, "A/B/C", nil)
	})
}

func TestRendererRenderFuncAsyncErrorsAndContext(t *testing.T) {
	valid, err := mm.Compile("{{a}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	resolver := func(context.Context, string, mm.Scope) (mm.Value, error) { return "A", nil }

	t.Run("nil context", func(t *testing.T) {
		got, callErr := valid.RenderFuncAsync(nil, resolver, mm.Scope{})
		assertRendererAsyncError(t, got, callErr, mm.ErrInvalidContext, "Renderer.RenderFuncAsync context is nil")
	})
	t.Run("nil resolver", func(t *testing.T) {
		got, callErr := valid.RenderFuncAsync(context.Background(), nil, mm.Scope{})
		assertRendererAsyncError(t, got, callErr, mm.ErrInvalidResolver, "Renderer.RenderFuncAsync resolver is nil")
	})
	t.Run("nil receiver", func(t *testing.T) {
		var nilRenderer *mm.Renderer
		got, callErr := nilRenderer.RenderFuncAsync(context.Background(), resolver, mm.Scope{})
		assertRendererAsyncError(t, got, callErr, mm.ErrInvalidRenderer, "Renderer")
	})
	t.Run("zero receiver", func(t *testing.T) {
		var zero mm.Renderer
		got, callErr := zero.RenderFuncAsync(context.Background(), resolver, mm.Scope{})
		assertRendererAsyncError(t, got, callErr, mm.ErrInvalidRenderer, "Renderer")
	})
	t.Run("resolver error preserves type", func(t *testing.T) {
		sentinel := &rendererAsyncError{code: 17}
		got, callErr := valid.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return nil, sentinel }, mm.Scope{})
		assertRendererAsyncError(t, got, callErr, sentinel, `path "a" at index 0`)
		var typed *rendererAsyncError
		if !errors.As(callErr, &typed) || typed != sentinel {
			t.Fatalf("errors.As(%v) did not preserve resolver error", callErr)
		}
	})
	t.Run("first observed of multiple errors", func(t *testing.T) {
		multi, compileErr := mm.Compile("{{a}}{{b}}{{c}}", mm.CompileOptions{})
		if compileErr != nil {
			t.Fatalf("Compile() error = %v", compileErr)
		}
		gates := map[string]chan struct{}{"a": make(chan struct{}), "b": make(chan struct{}), "c": make(chan struct{})}
		started := make(chan string, 3)
		finished := make(chan struct{}, 3)
		result := make(chan asyncCallResult, 1)
		go func() {
			got, callErr := multi.RenderFuncAsync(context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
				started <- path
				<-gates[path]
				finished <- struct{}{}
				return nil, errors.New(path + " failure")
			}, mm.Scope{})
			result <- asyncCallResult{value: got, err: callErr}
		}()
		for range 3 {
			receiveString(t, started)
		}
		close(gates["b"])
		call := receiveAsyncResult(t, result)
		close(gates["a"])
		close(gates["c"])
		for range 3 {
			receiveSignal(t, finished)
		}
		assertRendererAsyncError(t, call.value, call.err, call.err, `path "b" at index 1`)
	})
	t.Run("canceled before call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		calls := 0
		got, callErr := valid.RenderFuncAsync(ctx, func(context.Context, string, mm.Scope) (mm.Value, error) { calls++; return "A", nil }, mm.Scope{})
		if calls != 0 {
			t.Fatalf("resolver calls = %d", calls)
		}
		assertRendererAsyncError(t, got, callErr, context.Canceled, "Renderer.RenderFuncAsync context")
	})
	t.Run("cancel during call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{}, 1)
		release := make(chan struct{})
		finished := make(chan struct{}, 1)
		result := make(chan asyncCallResult, 1)
		go func() {
			got, callErr := valid.RenderFuncAsync(ctx, func(context.Context, string, mm.Scope) (mm.Value, error) {
				started <- struct{}{}
				<-release
				finished <- struct{}{}
				return "A", nil
			}, mm.Scope{})
			result <- asyncCallResult{value: got, err: callErr}
		}()
		receiveSignal(t, started)
		cancel()
		call := receiveAsyncResult(t, result)
		assertRendererAsyncError(t, call.value, call.err, context.Canceled, "Renderer.RenderFuncAsync context")
		close(release)
		receiveSignal(t, finished)
	})
	t.Run("deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		got, callErr := valid.RenderFuncAsync(ctx, func(ctx context.Context, _ string, _ mm.Scope) (mm.Value, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}, mm.Scope{})
		assertRendererAsyncError(t, got, callErr, context.DeadlineExceeded, "Renderer.RenderFuncAsync context")
	})
	t.Run("unsupported value after collection", func(t *testing.T) {
		two, compileErr := mm.Compile("{{a}}{{b}}", mm.CompileOptions{})
		if compileErr != nil {
			t.Fatalf("Compile() error = %v", compileErr)
		}
		calls := 0
		var mutex sync.Mutex
		got, callErr := two.RenderFuncAsync(context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
			mutex.Lock()
			calls++
			mutex.Unlock()
			if path == "a" {
				return struct{}{}, nil
			}
			return "B", nil
		}, mm.Scope{})
		if calls != 2 {
			t.Fatalf("resolver calls = %d", calls)
		}
		assertRendererAsyncError(t, got, callErr, mm.ErrUnsupportedValue, "unsupported value")
	})
	t.Run("validated invalid path fails at compile", func(t *testing.T) {
		renderer, compileErr := mm.Compile("{{a.}}", mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}})
		if renderer != nil || !errors.Is(compileErr, mm.ErrInvalidPath) {
			t.Fatalf("Compile() = %#v, %v", renderer, compileErr)
		}
	})
	t.Run("max path length fails at compile", func(t *testing.T) {
		renderer, compileErr := mm.Compile("{{abcd}}", mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{MaxPathLen: 3}})
		if renderer != nil || !errors.Is(compileErr, mm.ErrInvalidTemplate) {
			t.Fatalf("Compile() = %#v, %v", renderer, compileErr)
		}
	})
	t.Run("lazy invalid path remains raw resolver input", func(t *testing.T) {
		raw, compileErr := mm.Compile("{{a.}}", mm.CompileOptions{})
		if compileErr != nil {
			t.Fatalf("Compile() error = %v", compileErr)
		}
		got, callErr := raw.RenderFuncAsync(context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) { return path, nil }, mm.Scope{})
		if callErr != nil || got != "a." {
			t.Fatalf("Renderer.RenderFuncAsync() = %q, %v", got, callErr)
		}
	})
}

func TestRendererRenderFuncAsyncReuse(t *testing.T) {
	t.Run("different resolvers", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
		for _, want := range []string{"A", "B"} {
			got, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return want, nil }, mm.Scope{})
			if err != nil || got != want {
				t.Fatalf("result = %q, %v; want %q", got, err, want)
			}
		}
	})
	t.Run("same resolver has no result cache", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
		calls := 0
		resolver := func(context.Context, string, mm.Scope) (mm.Value, error) { calls++; return calls, nil }
		first, firstErr := renderer.RenderFuncAsync(context.Background(), resolver, mm.Scope{})
		second, secondErr := renderer.RenderFuncAsync(context.Background(), resolver, mm.Scope{})
		if firstErr != nil || secondErr != nil || first != "1" || second != "2" || calls != 2 {
			t.Fatalf("first=%q/%v second=%q/%v calls=%d", first, firstErr, second, secondErr, calls)
		}
	})
	t.Run("different contexts", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got, err := renderer.RenderFuncAsync(ctx, func(context.Context, string, mm.Scope) (mm.Value, error) { return "bad", nil }, mm.Scope{}); got != "" || !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled result = %q, %v", got, err)
		}
		if got, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "good", nil }, mm.Scope{}); got != "good" || err != nil {
			t.Fatalf("reused result = %q, %v", got, err)
		}
	})
	t.Run("missing then present", func(t *testing.T) {
		assertRendererAsyncSequence(t, mm.Undefined{}, "present", "", "present")
	})
	t.Run("present then missing", func(t *testing.T) {
		assertRendererAsyncSequence(t, "present", mm.Undefined{}, "present", "")
	})
	t.Run("after resolver error", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
		sentinel := errors.New("first failure")
		if _, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return nil, sentinel }, mm.Scope{}); !errors.Is(err, sentinel) {
			t.Fatalf("first error = %v", err)
		}
		if got, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "recovered", nil }, mm.Scope{}); got != "recovered" || err != nil {
			t.Fatalf("reused result = %q, %v", got, err)
		}
	})
	t.Run("after cancellation", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _ = renderer.RenderFuncAsync(ctx, func(context.Context, string, mm.Scope) (mm.Value, error) { return "bad", nil }, mm.Scope{})
		if got, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "after", nil }, mm.Scope{}); got != "after" || err != nil {
			t.Fatalf("reused result = %q, %v", got, err)
		}
	})
	t.Run("after deadline", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		_, _ = renderer.RenderFuncAsync(ctx, func(context.Context, string, mm.Scope) (mm.Value, error) { return "bad", nil }, mm.Scope{})
		if got, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "after", nil }, mm.Scope{}); got != "after" || err != nil {
			t.Fatalf("reused result = %q, %v", got, err)
		}
	})
	t.Run("after unsupported value", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
		if _, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return struct{}{}, nil }, mm.Scope{}); !errors.Is(err, mm.ErrUnsupportedValue) {
			t.Fatalf("first error = %v", err)
		}
		if got, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "after", nil }, mm.Scope{}); got != "after" || err != nil {
			t.Fatalf("reused result = %q, %v", got, err)
		}
	})
	t.Run("different completion orders", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{a}}/{{b}}/{{c}}", mm.CompileOptions{})
		for _, order := range [][]string{{"c", "b", "a"}, {"a", "b", "c"}} {
			if got := runCompiledAsyncInCompletionOrder(t, renderer, order); got != "A/B/C" {
				t.Fatalf("order %#v result = %q", order, got)
			}
		}
	})
}

func TestRendererRenderFuncAsyncMatchesTopLevel(t *testing.T) {
	tests := []struct {
		name     string
		template string
		values   map[string]mm.Value
		options  mm.CompileOptions
	}{
		{"simple", "Hello {{name}}", map[string]mm.Value{"name": "Ada"}, mm.CompileOptions{}},
		{"repeated", "{{x}}/{{x}}", map[string]mm.Value{"x": "X"}, mm.CompileOptions{}},
		{"custom tags", "<<name>>", map[string]mm.Value{"name": "tag"}, mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: ">>"}}}},
		{"explicit", "{{missing}}/{{nil}}", map[string]mm.Value{"nil": nil}, mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: true}}},
		{"unicode", "{{名前}} 🚀", map[string]mm.Value{"名前": "世界"}, mm.CompileOptions{}},
		{"max ref depth ignored", "{{a.b.c}}", map[string]mm.Value{"a.b.c": "raw"}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: 1}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer := mustCompileRenderer(t, test.template, test.options)
			scope := mm.Scope{"scope": test.name}
			compiledPaths := make(chan string, 8)
			directPaths := make(chan string, 8)
			makeResolver := func(paths chan<- string) mm.AsyncResolver {
				return func(_ context.Context, path string, received mm.Scope) (mm.Value, error) {
					if !reflect.DeepEqual(received, scope) {
						return nil, errors.New("scope changed")
					}
					paths <- path
					if value, ok := test.values[path]; ok {
						return value, nil
					}
					return mm.Undefined{}, nil
				}
			}
			compiled, compiledErr := renderer.RenderFuncAsync(context.Background(), makeResolver(compiledPaths), scope)
			direct, directErr := mm.RenderFuncAsync(context.Background(), test.template, makeResolver(directPaths), scope, test.options)
			if compiledErr != nil || directErr != nil || compiled != direct {
				t.Fatalf("compiled=%q/%v direct=%q/%v", compiled, compiledErr, direct, directErr)
			}
			close(compiledPaths)
			close(directPaths)
			if !sameStringCounts(drainStrings(compiledPaths), drainStrings(directPaths)) {
				t.Fatal("resolver path occurrence counts differ")
			}
		})
	}

	t.Run("resolver error", func(t *testing.T) {
		sentinel := errors.New("resolver failure")
		resolver := func(context.Context, string, mm.Scope) (mm.Value, error) { return nil, sentinel }
		renderer := mustCompileRenderer(t, "{{a}}", mm.CompileOptions{})
		_, compiledErr := renderer.RenderFuncAsync(context.Background(), resolver, mm.Scope{})
		_, directErr := mm.RenderFuncAsync(context.Background(), "{{a}}", resolver, mm.Scope{}, mm.CompileOptions{})
		if !errors.Is(compiledErr, sentinel) || !errors.Is(directErr, sentinel) {
			t.Fatalf("compiled=%v direct=%v", compiledErr, directErr)
		}
	})
	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		resolver := func(context.Context, string, mm.Scope) (mm.Value, error) { return "bad", nil }
		renderer := mustCompileRenderer(t, "{{a}}", mm.CompileOptions{})
		_, compiledErr := renderer.RenderFuncAsync(ctx, resolver, mm.Scope{})
		_, directErr := mm.RenderFuncAsync(ctx, "{{a}}", resolver, mm.Scope{}, mm.CompileOptions{})
		if !errors.Is(compiledErr, context.Canceled) || !errors.Is(directErr, context.Canceled) {
			t.Fatalf("compiled=%v direct=%v", compiledErr, directErr)
		}
	})
	t.Run("deadline context", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		resolver := func(context.Context, string, mm.Scope) (mm.Value, error) { return "bad", nil }
		renderer := mustCompileRenderer(t, "{{a}}", mm.CompileOptions{})
		_, compiledErr := renderer.RenderFuncAsync(ctx, resolver, mm.Scope{})
		_, directErr := mm.RenderFuncAsync(ctx, "{{a}}", resolver, mm.Scope{}, mm.CompileOptions{})
		if !errors.Is(compiledErr, context.DeadlineExceeded) || !errors.Is(directErr, context.DeadlineExceeded) {
			t.Fatalf("compiled=%v direct=%v", compiledErr, directErr)
		}
	})
	t.Run("unsupported value", func(t *testing.T) {
		resolver := func(context.Context, string, mm.Scope) (mm.Value, error) { return struct{}{}, nil }
		renderer := mustCompileRenderer(t, "{{a}}", mm.CompileOptions{})
		_, compiledErr := renderer.RenderFuncAsync(context.Background(), resolver, mm.Scope{})
		_, directErr := mm.RenderFuncAsync(context.Background(), "{{a}}", resolver, mm.Scope{}, mm.CompileOptions{})
		if !errors.Is(compiledErr, mm.ErrUnsupportedValue) || !errors.Is(directErr, mm.ErrUnsupportedValue) {
			t.Fatalf("compiled=%v direct=%v", compiledErr, directErr)
		}
	})
	t.Run("validation classification", func(t *testing.T) {
		options := mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}
		renderer, compiledErr := mm.Compile("{{a.}}", options)
		_, directErr := mm.RenderFuncAsync(context.Background(), "{{a.}}", func(context.Context, string, mm.Scope) (mm.Value, error) { return "bad", nil }, mm.Scope{}, options)
		if renderer != nil || !errors.Is(compiledErr, mm.ErrInvalidPath) || !errors.Is(directErr, mm.ErrInvalidPath) {
			t.Fatalf("renderer=%#v compiled=%v direct=%v", renderer, compiledErr, directErr)
		}
	})
}

func TestRendererRenderFuncAsyncQualityAndConcurrentReuse(t *testing.T) {
	t.Run("state and inputs remain unchanged", func(t *testing.T) {
		template := "{{first}}/{{second}}"
		options := mm.CompileOptions{}
		renderer := mustCompileRenderer(t, template, options)
		returned := []any{1, nil, map[string]any{"x": true}}
		wantReturned := []any{1, nil, map[string]any{"x": true}}
		scope := mm.Scope{"marker": "unchanged"}
		wantScope := mm.Scope{"marker": "unchanged"}
		resolver := func(_ context.Context, path string, received mm.Scope) (mm.Value, error) {
			if !reflect.DeepEqual(received, wantScope) {
				return nil, errors.New("scope changed")
			}
			if path == "first" {
				return returned, nil
			}
			return "done", nil
		}
		for iteration := 0; iteration < 100; iteration++ {
			got, err := renderer.RenderFuncAsync(context.Background(), resolver, scope)
			if err != nil || got != "1,,[object Object]/done" {
				t.Fatalf("iteration %d = %q, %v", iteration, got, err)
			}
		}
		if template != "{{first}}/{{second}}" || options != (mm.CompileOptions{}) || !reflect.DeepEqual(scope, wantScope) || !reflect.DeepEqual(returned, wantReturned) {
			t.Fatalf("inputs changed: template=%q options=%#v scope=%#v returned=%#v", template, options, scope, returned)
		}
	})

	t.Run("64 concurrent read-only calls", func(t *testing.T) {
		renderer := mustCompileRenderer(t, "{{prefix}}:{{value}}", mm.CompileOptions{})
		const workers = 64
		errorsFound := make(chan error, workers)
		var group sync.WaitGroup
		for worker := 0; worker < workers; worker++ {
			worker := worker
			group.Add(1)
			go func() {
				defer group.Done()
				scope := mm.Scope{"worker": worker}
				got, err := renderer.RenderFuncAsync(context.Background(), func(_ context.Context, path string, received mm.Scope) (mm.Value, error) {
					return fmt.Sprintf("%s-%d", path, received["worker"]), nil
				}, scope)
				want := fmt.Sprintf("prefix-%d:value-%d", worker, worker)
				if err != nil {
					errorsFound <- err
				} else if got != want {
					errorsFound <- fmt.Errorf("worker %d result %q; want %q", worker, got, want)
				}
			}()
		}
		group.Wait()
		close(errorsFound)
		for err := range errorsFound {
			t.Error(err)
		}
	})
}

func runCompiledAsyncInCompletionOrder(t *testing.T, renderer *mm.Renderer, order []string) string {
	t.Helper()
	gates := map[string]chan struct{}{"a": make(chan struct{}), "b": make(chan struct{}), "c": make(chan struct{})}
	started := make(chan string, 3)
	completed := make(chan string, 3)
	result := make(chan asyncCallResult, 1)
	go func() {
		got, err := renderer.RenderFuncAsync(context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
			started <- path
			<-gates[path]
			completed <- path
			return strings.ToUpper(path), nil
		}, mm.Scope{})
		result <- asyncCallResult{value: got, err: err}
	}()
	for range 3 {
		receiveString(t, started)
	}
	for _, path := range order {
		close(gates[path])
		if got := receiveString(t, completed); got != path {
			t.Fatalf("completion = %q; want %q", got, path)
		}
	}
	call := receiveAsyncResult(t, result)
	if call.err != nil {
		t.Fatalf("Renderer.RenderFuncAsync() error = %v", call.err)
	}
	return call.value
}

func assertRendererAsyncSequence(t *testing.T, firstValue, secondValue mm.Value, firstWant, secondWant string) {
	t.Helper()
	renderer := mustCompileRenderer(t, "{{value}}", mm.CompileOptions{})
	for index, item := range []struct {
		value mm.Value
		want  string
	}{{firstValue, firstWant}, {secondValue, secondWant}} {
		got, err := renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return item.value, nil }, mm.Scope{})
		if err != nil || got != item.want {
			t.Fatalf("call %d = %q, %v; want %q", index, got, err, item.want)
		}
	}
}

func mustCompileRenderer(t *testing.T, template string, options mm.CompileOptions) *mm.Renderer {
	t.Helper()
	renderer, err := mm.Compile(template, options)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return renderer
}

func assertRendererAsyncError(t *testing.T, got string, err, kind error, text string) {
	t.Helper()
	if got != "" || !errors.Is(err, kind) || !strings.Contains(err.Error(), text) {
		t.Fatalf("result = %q, %v; want kind=%v text=%q", got, err, kind, text)
	}
}

func drainStrings(values <-chan string) []string {
	var result []string
	for value := range values {
		result = append(result, value)
	}
	return result
}

type rendererAsyncError struct{ code int }

func (e *rendererAsyncError) Error() string { return "compiled async resolver failure" }
