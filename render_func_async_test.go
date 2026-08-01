package micromustache_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestRenderFuncAsyncMatchesImmediateSyncCases(t *testing.T) {
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
		{"string", "{{v}}", map[string]mm.Value{"v": "text"}, mm.CompileOptions{}, "text", []string{"v"}},
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
			scopeMarker := &struct{ name string }{name: test.name}
			scope := mm.Scope{"marker": scopeMarker}
			var mutex sync.Mutex
			var asyncPaths []string
			asyncResolver := func(ctx context.Context, path string, received mm.Scope) (mm.Value, error) {
				if ctx == nil || received["marker"] != scopeMarker {
					return nil, errors.New("resolver arguments were not preserved")
				}
				mutex.Lock()
				asyncPaths = append(asyncPaths, path)
				mutex.Unlock()
				if value, ok := test.values[path]; ok {
					return value, nil
				}
				return mm.Undefined{}, nil
			}

			got, err := mm.RenderFuncAsync(context.Background(), test.template, asyncResolver, scope, test.options)
			if err != nil || got != test.want {
				t.Fatalf("RenderFuncAsync() = %q, %v; want %q", got, err, test.want)
			}
			var syncPaths []string
			syncGot, err := mm.RenderFunc(test.template, func(path string, received mm.Scope) (mm.Value, error) {
				syncPaths = append(syncPaths, path)
				return asyncResolver(context.Background(), path, received)
			}, scope, test.options)
			if err != nil || syncGot != test.want {
				t.Fatalf("RenderFunc() = %q, %v; want %q", syncGot, err, test.want)
			}
			if len(asyncPaths) < len(test.wantPaths) || !sameStringCounts(asyncPaths[:len(test.wantPaths)], test.wantPaths) || !reflect.DeepEqual(syncPaths, test.wantPaths) {
				t.Fatalf("paths async=%#v sync=%#v want=%#v", asyncPaths, syncPaths, test.wantPaths)
			}
		})
	}
}

func TestRenderFuncAsyncConcurrencyContract(t *testing.T) {
	t.Run("all start before completion", func(t *testing.T) {
		started := make(chan string, 3)
		release := make(chan struct{})
		result := make(chan asyncCallResult, 1)
		go func() {
			got, err := mm.RenderFuncAsync(context.Background(), "{{a}}/{{b}}/{{c}}", func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
				started <- path
				<-release
				return strings.ToUpper(path), nil
			}, mm.Scope{}, mm.CompileOptions{})
			result <- asyncCallResult{value: got, err: err}
		}()
		paths := []string{receiveString(t, started), receiveString(t, started), receiveString(t, started)}
		if !sameStringCounts(paths, []string{"a", "b", "c"}) {
			t.Fatalf("invoked paths = %#v", paths)
		}
		close(release)
		assertAsyncResult(t, result, "A/B/C", nil)
	})

	t.Run("reverse completion keeps template order", func(t *testing.T) {
		gates := map[string]chan struct{}{"a": make(chan struct{}), "b": make(chan struct{}), "c": make(chan struct{})}
		started := make(chan string, 3)
		completed := make(chan string, 3)
		result := make(chan asyncCallResult, 1)
		go func() {
			got, err := mm.RenderFuncAsync(context.Background(), "{{a}}/{{b}}/{{c}}", func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
				started <- path
				<-gates[path]
				completed <- path
				return strings.ToUpper(path), nil
			}, mm.Scope{}, mm.CompileOptions{})
			result <- asyncCallResult{value: got, err: err}
		}()
		for range 3 {
			receiveString(t, started)
		}
		for _, path := range []string{"c", "b", "a"} {
			close(gates[path])
			if got := receiveString(t, completed); got != path {
				t.Fatalf("completion = %q, want %q", got, path)
			}
		}
		assertAsyncResult(t, result, "A/B/C", nil)
	})

	t.Run("repeated paths are independent occurrences", func(t *testing.T) {
		var mutex sync.Mutex
		calls := 0
		got, err := mm.RenderFuncAsync(context.Background(), "{{x}}/{{x}}/{{x}}", func(context.Context, string, mm.Scope) (mm.Value, error) {
			mutex.Lock()
			calls++
			mutex.Unlock()
			return "X", nil
		}, mm.Scope{}, mm.CompileOptions{})
		if err != nil || got != "X/X/X" || calls != 3 {
			t.Fatalf("RenderFuncAsync() = %q, %v; calls=%d", got, err, calls)
		}
	})

	t.Run("shared release still preserves output order", func(t *testing.T) {
		started := make(chan struct{}, 3)
		release := make(chan struct{})
		result := make(chan asyncCallResult, 1)
		go func() {
			got, err := mm.RenderFuncAsync(context.Background(), "{{a}}/{{b}}/{{c}}", func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
				started <- struct{}{}
				<-release
				return strings.ToUpper(path), nil
			}, mm.Scope{}, mm.CompileOptions{})
			result <- asyncCallResult{value: got, err: err}
		}()
		for range 3 {
			receiveSignal(t, started)
		}
		close(release)
		assertAsyncResult(t, result, "A/B/C", nil)
	})
}

func TestRenderFuncAsyncErrorsAndStages(t *testing.T) {
	sentinel := &asyncResolverError{code: 42}
	tests := []struct {
		name     string
		template string
		options  mm.CompileOptions
		ctx      context.Context
		resolver mm.AsyncResolver
		kind     error
		text     string
		calls    int
	}{
		{"nil context", "{{a}}", mm.CompileOptions{}, nil, func(context.Context, string, mm.Scope) (mm.Value, error) { return "A", nil }, mm.ErrInvalidContext, "context is nil", 0},
		{"nil resolver", "{{a}}", mm.CompileOptions{}, context.Background(), nil, mm.ErrInvalidResolver, "resolver is nil", 0},
		{"resolver error", "{{a}}", mm.CompileOptions{}, context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return nil, sentinel }, sentinel, "async resolver failure", 1},
		{"later resolver error", "{{a}}{{b}}{{c}}", mm.CompileOptions{}, context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
			if path == "b" {
				return nil, sentinel
			}
			return path, nil
		}, sentinel, "async resolver failure", -1},
		{"unsupported value after all calls", "{{a}}{{b}}{{c}}", mm.CompileOptions{}, context.Background(), func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
			if path == "a" {
				return struct{}{}, nil
			}
			return path, nil
		}, mm.ErrUnsupportedValue, "unsupported value", 3},
		{"validated invalid path", "{{a.}}", mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}, context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "unreached", nil }, mm.ErrInvalidPath, `Could not parse path: "a."`, 0},
		{"max path length", "{{abcd}}", mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{MaxPathLen: 3}}, context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "unreached", nil }, mm.ErrInvalidTemplate, "within 3 characters", 0},
		{"unclosed tag", "before {{a", mm.CompileOptions{}, context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return "unreached", nil }, mm.ErrInvalidTemplate, `Missing "}}"`, 0},
		{"resolver returns context error", "{{a}}", mm.CompileOptions{}, context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return nil, context.Canceled }, context.Canceled, "path \"a\" at index 0", 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var callsMutex sync.Mutex
			resolver := test.resolver
			if resolver != nil {
				resolver = func(ctx context.Context, path string, scope mm.Scope) (mm.Value, error) {
					callsMutex.Lock()
					calls++
					callsMutex.Unlock()
					return test.resolver(ctx, path, scope)
				}
			}
			got, err := mm.RenderFuncAsync(test.ctx, test.template, resolver, mm.Scope{}, test.options)
			callsMutex.Lock()
			gotCalls := calls
			callsMutex.Unlock()
			if got != "" || !errors.Is(err, test.kind) || !strings.Contains(err.Error(), test.text) || test.calls >= 0 && gotCalls != test.calls {
				t.Fatalf("RenderFuncAsync() = %q, %v; calls=%d, want kind=%v text=%q calls=%d", got, err, gotCalls, test.kind, test.text, test.calls)
			}
			if test.kind == sentinel {
				var typed *asyncResolverError
				if !errors.As(err, &typed) || typed != sentinel {
					t.Fatalf("errors.As(%v) did not preserve resolver error", err)
				}
			}
		})
	}
}

func TestRenderFuncAsyncReturnsFirstObservedResolverError(t *testing.T) {
	gates := map[string]chan struct{}{"a": make(chan struct{}), "b": make(chan struct{}), "c": make(chan struct{})}
	started := make(chan string, 3)
	finished := make(chan struct{}, 3)
	result := make(chan asyncCallResult, 1)
	go func() {
		got, err := mm.RenderFuncAsync(context.Background(), "{{a}}{{b}}{{c}}", func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
			started <- path
			<-gates[path]
			finished <- struct{}{}
			return nil, errors.New(path + " failure")
		}, mm.Scope{}, mm.CompileOptions{})
		result <- asyncCallResult{value: got, err: err}
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
	if call.value != "" || call.err == nil || !strings.Contains(call.err.Error(), `path "b" at index 1`) || !strings.Contains(call.err.Error(), "b failure") {
		t.Fatalf("RenderFuncAsync() = %q, %v", call.value, call.err)
	}
}

func TestRenderFuncAsyncContextCancellation(t *testing.T) {
	t.Run("canceled before call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		calls := 0
		got, err := mm.RenderFuncAsync(ctx, "{{a}}", func(context.Context, string, mm.Scope) (mm.Value, error) { calls++; return "A", nil }, mm.Scope{}, mm.CompileOptions{})
		if got != "" || !errors.Is(err, context.Canceled) || calls != 0 {
			t.Fatalf("RenderFuncAsync() = %q, %v; calls=%d", got, err, calls)
		}
	})

	t.Run("cancel during calls returns without waiting for ignoring resolvers", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{}, 3)
		release := make(chan struct{})
		finished := make(chan struct{}, 3)
		result := make(chan asyncCallResult, 1)
		go func() {
			got, err := mm.RenderFuncAsync(ctx, "{{a}}{{b}}{{c}}", func(context.Context, string, mm.Scope) (mm.Value, error) {
				started <- struct{}{}
				<-release
				finished <- struct{}{}
				return "value", nil
			}, mm.Scope{}, mm.CompileOptions{})
			result <- asyncCallResult{value: got, err: err}
		}()
		for range 3 {
			receiveSignal(t, started)
		}
		cancel()
		call := receiveAsyncResult(t, result)
		if call.value != "" || !errors.Is(call.err, context.Canceled) {
			t.Fatalf("RenderFuncAsync() = %q, %v", call.value, call.err)
		}
		close(release)
		for range 3 {
			receiveSignal(t, finished)
		}
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		got, err := mm.RenderFuncAsync(ctx, "{{a}}", func(ctx context.Context, _ string, _ mm.Scope) (mm.Value, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}, mm.Scope{}, mm.CompileOptions{})
		if got != "" || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("RenderFuncAsync() = %q, %v", got, err)
		}
	})

	t.Run("compile error precedes nil context", func(t *testing.T) {
		got, err := mm.RenderFuncAsync(nil, "before {{a", func(context.Context, string, mm.Scope) (mm.Value, error) { return "unreached", nil }, mm.Scope{}, mm.CompileOptions{})
		if got != "" || !errors.Is(err, mm.ErrInvalidTemplate) {
			t.Fatalf("RenderFuncAsync() = %q, %v", got, err)
		}
	})

	t.Run("nil resolver precedes canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got, err := mm.RenderFuncAsync(ctx, "{{a}}", nil, mm.Scope{}, mm.CompileOptions{})
		if got != "" || !errors.Is(err, mm.ErrInvalidResolver) {
			t.Fatalf("RenderFuncAsync() = %q, %v", got, err)
		}
	})
}

func TestRenderFuncAsyncDoesNotMutateInputsAndIsDeterministic(t *testing.T) {
	template := "{{first}}/{{second}}"
	returned := []any{1, nil, map[string]any{"x": true}}
	wantReturned := []any{1, nil, map[string]any{"x": true}}
	scope := mm.Scope{"marker": "unchanged"}
	wantScope := mm.Scope{"marker": "unchanged"}
	resolver := func(_ context.Context, path string, received mm.Scope) (mm.Value, error) {
		if !reflect.DeepEqual(received, wantScope) {
			return nil, errors.New("resolver scope changed")
		}
		if path == "first" {
			return returned, nil
		}
		return "done", nil
	}
	for iteration := 0; iteration < 100; iteration++ {
		got, err := mm.RenderFuncAsync(context.Background(), template, resolver, scope, mm.CompileOptions{})
		if err != nil || got != "1,,[object Object]/done" {
			t.Fatalf("iteration %d = %q, %v", iteration, got, err)
		}
	}
	if template != "{{first}}/{{second}}" || !reflect.DeepEqual(scope, wantScope) || !reflect.DeepEqual(returned, wantReturned) {
		t.Fatalf("inputs mutated: template=%q scope=%#v returned=%#v", template, scope, returned)
	}
}

type asyncCallResult struct {
	value string
	err   error
}

func receiveAsyncResult(t *testing.T, values <-chan asyncCallResult) asyncCallResult {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for RenderFuncAsync")
		return asyncCallResult{}
	}
}

func assertAsyncResult(t *testing.T, values <-chan asyncCallResult, want string, wantErr error) {
	t.Helper()
	got := receiveAsyncResult(t, values)
	if got.value != want || !errors.Is(got.err, wantErr) {
		t.Fatalf("RenderFuncAsync() = %q, %v; want %q, %v", got.value, got.err, want, wantErr)
	}
}

func receiveString(t *testing.T, values <-chan string) string {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for resolver event")
		return ""
	}
}

func receiveSignal(t *testing.T, values <-chan struct{}) {
	t.Helper()
	select {
	case <-values:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for resolver signal")
	}
}

func sameStringCounts(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}

type asyncResolverError struct{ code int }

func (e *asyncResolverError) Error() string { return "async resolver failure" }
