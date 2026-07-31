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

func TestCompileRenderMatchesTopLevelRender(t *testing.T) {
	explicit := mm.CompileOptions{RendererOptions: mm.RendererOptions{Explicit: true}}
	customTags := mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: ">>"}}}
	tests := []struct {
		name     string
		template string
		scope    mm.Scope
		options  mm.CompileOptions
	}{
		{"plain", "plain text", mm.Scope{}, mm.CompileOptions{}},
		{"empty", "", mm.Scope{}, mm.CompileOptions{}},
		{"simple", "Hello {{name}}!", mm.Scope{"name": "Ada"}, mm.CompileOptions{}},
		{"multiple", "{{a}}/{{b}}/{{c}}", mm.Scope{"a": "A", "b": "B", "c": "C"}, mm.CompileOptions{}},
		{"repeated", "{{x}}-{{x}}-{{x}}", mm.Scope{"x": "X"}, mm.CompileOptions{}},
		{"adjacent", "{{a}}{{b}}{{c}}", mm.Scope{"a": 1, "b": 2, "c": 3}, mm.CompileOptions{}},
		{"custom tags", "<<name>> {{name}}", mm.Scope{"name": "tagged"}, customTags},
		{"unicode", "こんにちは {{name}} 🚀", mm.Scope{"name": "世界"}, mm.CompileOptions{}},
		{"nested", "{{user.name}}", mm.Scope{"user": mm.Scope{"name": "Lin"}}, mm.CompileOptions{}},
		{"bracket", "{{obj['a.b']}}", mm.Scope{"obj": mm.Scope{"a.b": "quoted"}}, mm.CompileOptions{}},
		{"array index", "{{items[1]}}", mm.Scope{"items": []any{"zero", "one"}}, mm.CompileOptions{}},
		{"missing default", "A{{missing}}B", mm.Scope{}, mm.CompileOptions{}},
		{"missing explicit", "A{{missing}}B", mm.Scope{}, explicit},
		{"undefined default", "A{{v}}B", mm.Scope{"v": mm.Undefined{}}, mm.CompileOptions{}},
		{"undefined explicit", "A{{v}}B", mm.Scope{"v": mm.Undefined{}}, explicit},
		{"nil default", "A{{v}}B", mm.Scope{"v": nil}, mm.CompileOptions{}},
		{"nil explicit", "A{{v}}B", mm.Scope{"v": nil}, explicit},
		{"true", "{{v}}", mm.Scope{"v": true}, mm.CompileOptions{}},
		{"false", "{{v}}", mm.Scope{"v": false}, mm.CompileOptions{}},
		{"zero", "{{v}}", mm.Scope{"v": float64(0)}, mm.CompileOptions{}},
		{"negative zero", "{{v}}", mm.Scope{"v": math.Copysign(0, -1)}, mm.CompileOptions{}},
		{"integer", "{{v}}", mm.Scope{"v": 42}, mm.CompileOptions{}},
		{"decimal", "{{v}}", mm.Scope{"v": 123.456}, mm.CompileOptions{}},
		{"nan", "{{v}}", mm.Scope{"v": math.NaN()}, mm.CompileOptions{}},
		{"positive infinity", "{{v}}", mm.Scope{"v": math.Inf(1)}, mm.CompileOptions{}},
		{"negative infinity", "{{v}}", mm.Scope{"v": math.Inf(-1)}, mm.CompileOptions{}},
		{"empty string", "A{{v}}B", mm.Scope{"v": ""}, mm.CompileOptions{}},
		{"emoji", "{{v}}", mm.Scope{"v": "🦫✨"}, mm.CompileOptions{}},
		{"array", "{{v}}", mm.Scope{"v": []any{1, nil, "x"}}, mm.CompileOptions{}},
		{"nested array", "{{v}}", mm.Scope{"v": []any{[]any{1, 2}, []any{3}}}, mm.CompileOptions{}},
		{"object", "{{v}}", mm.Scope{"v": map[string]any{"a": 1}}, mm.CompileOptions{}},
		{"array length", "{{items.length}}", mm.Scope{"items": []any{1, 2}}, mm.CompileOptions{}},
		{"quoted unicode", "{{obj['名前']}}", mm.Scope{"obj": mm.Scope{"名前": "世界"}}, mm.CompileOptions{}},
		{"repeated missing explicit", "A{{x}}B{{x}}C", mm.Scope{}, explicit},
		{"validated path", "{{user.name}}", mm.Scope{"user": mm.Scope{"name": "valid"}}, mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}},
		{"depth boundary", "{{a.b}}", mm.Scope{"a": mm.Scope{"b": "ok"}}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: 2}}}},
		{"prototype looking map", "{{v}}", mm.Scope{"v": map[string]any{"__proto__": "own", "constructor": "own"}}, mm.CompileOptions{}},
		{"one e twenty", "{{v}}", mm.Scope{"v": 1e20}, mm.CompileOptions{}},
		{"one e twenty one", "{{v}}", mm.Scope{"v": 1e21}, mm.CompileOptions{}},
		{"nil scope", "A{{missing}}B", nil, mm.CompileOptions{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer, err := mm.Compile(test.template, test.options)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			compiled, err := renderer.Render(test.scope)
			if err != nil {
				t.Fatalf("Renderer.Render() error = %v", err)
			}
			direct, err := mm.Render(test.template, test.scope, test.options)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if compiled != direct {
				t.Fatalf("compiled = %q, direct = %q", compiled, direct)
			}
		})
	}
}

func TestCompileAndRenderErrorStages(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		scope       mm.Scope
		options     mm.CompileOptions
		compileKind error
		renderKind  error
		message     string
	}{
		{"invalid tags", "<<a>>", mm.Scope{}, mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: "<<"}}}, mm.ErrInvalidOption, nil, "open and close symbols"},
		{"unclosed tag", "before {{a", mm.Scope{"a": "x"}, mm.CompileOptions{}, mm.ErrInvalidTemplate, nil, `Missing "}}"`},
		{"max path length", "{{abcd}}", mm.Scope{"abcd": "x"}, mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{MaxPathLen: 3}}, mm.ErrInvalidTemplate, nil, "within 3 characters"},
		{"invalid path eager", "{{a.}}", mm.Scope{}, mm.CompileOptions{RendererOptions: mm.RendererOptions{ValidatePath: true}}, mm.ErrInvalidPath, nil, `Could not parse path: "a."`},
		{"invalid path lazy", "{{a.}}", mm.Scope{}, mm.CompileOptions{}, nil, mm.ErrInvalidPath, `Could not parse path: "a."`},
		{"first invalid path", "{{a.}}{{b.}}", mm.Scope{}, mm.CompileOptions{}, nil, mm.ErrInvalidPath, `Could not parse path: "a."`},
		{"invalid path before data", "{{bad}}{{a.}}", mm.Scope{"bad": struct{}{}}, mm.CompileOptions{}, nil, mm.ErrInvalidPath, `Could not parse path: "a."`},
		{"max ref depth", "{{a.b.c}}", mm.Scope{"a": mm.Scope{"b": mm.Scope{"c": "x"}}}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: 2}}}, nil, mm.ErrReference, "deeper than 2 levels"},
		{"validated missing", "{{missing}}", mm.Scope{}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{ValidateRef: true}}}, nil, mm.ErrReference, "missing is not defined"},
		{"unsupported value", "{{bad}}", mm.Scope{"bad": struct{}{}}, mm.CompileOptions{}, nil, mm.ErrUnsupportedValue, "unsupported value"},
		{"negative max ref depth", "{{a}}", mm.Scope{"a": "x"}, mm.CompileOptions{RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: -1}}}, nil, mm.ErrInvalidOption, "positive number"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer, err := mm.Compile(test.template, test.options)
			if test.compileKind != nil {
				if renderer != nil {
					t.Fatal("Compile() returned renderer on error")
				}
				assertErrorKindAndText(t, err, test.compileKind, test.message)
				direct, directErr := mm.Render(test.template, test.scope, test.options)
				if direct != "" {
					t.Fatalf("Render() result on error = %q", direct)
				}
				assertErrorKindAndText(t, directErr, test.compileKind, test.message)
				return
			}
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			got, err := renderer.Render(test.scope)
			if got != "" {
				t.Fatalf("Renderer.Render() result on error = %q", got)
			}
			assertErrorKindAndText(t, err, test.renderKind, test.message)
			direct, directErr := mm.Render(test.template, test.scope, test.options)
			if direct != "" {
				t.Fatalf("Render() result on error = %q", direct)
			}
			assertErrorKindAndText(t, directErr, test.renderKind, test.message)
		})
	}
}

func TestNewRendererValidatesAndCopiesInputs(t *testing.T) {
	tests := []struct {
		name   string
		tokens mm.Tokens
		valid  bool
	}{
		{"zero tokens", mm.Tokens{}, false},
		{"empty arrays", mm.Tokens{Strings: []string{}, Paths: []string{}}, false},
		{"too few strings", mm.Tokens{Strings: []string{""}, Paths: []string{"a"}}, false},
		{"too many strings", mm.Tokens{Strings: []string{"", "", ""}, Paths: []string{"a"}}, false},
		{"empty template", mm.Tokens{Strings: []string{""}, Paths: []string{}}, true},
		{"one path", mm.Tokens{Strings: []string{"A", "B"}, Paths: []string{"value"}}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			renderer, err := mm.NewRenderer(test.tokens, mm.RendererOptions{})
			if !test.valid {
				if renderer != nil {
					t.Fatal("NewRenderer() returned renderer for invalid tokens")
				}
				assertErrorKindAndText(t, err, mm.ErrInvalidTokens, "Invalid tokens object")
				return
			}
			if err != nil || renderer == nil {
				t.Fatalf("NewRenderer() = %#v, %v", renderer, err)
			}
		})
	}

	tokens := mm.Tokens{Strings: []string{"A", "B"}, Paths: []string{"value"}}
	options := mm.RendererOptions{}
	renderer, err := mm.NewRenderer(tokens, options)
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	tokens.Strings[0], tokens.Strings[1], tokens.Paths[0] = "X", "Y", "other"
	options.Explicit = true
	got, err := renderer.Render(mm.Scope{"value": "x", "other": "wrong"})
	if err != nil || got != "AxB" {
		t.Fatalf("Renderer.Render() after input mutation = %q, %v; want AxB", got, err)
	}

	explicitOptions := mm.RendererOptions{Explicit: false}
	explicitRenderer, err := mm.NewRenderer(mm.Tokens{Strings: []string{"A", "B"}, Paths: []string{"value"}}, explicitOptions)
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	explicitOptions.Explicit = true
	got, err = explicitRenderer.Render(mm.Scope{"value": nil})
	if err != nil || got != "AB" {
		t.Fatalf("Renderer options were not snapshotted: got %q, %v", got, err)
	}

	invalidTokens := mm.Tokens{Strings: []string{"", ""}, Paths: []string{"a."}}
	if invalidRenderer, err := mm.NewRenderer(invalidTokens, mm.RendererOptions{ValidatePath: true}); invalidRenderer != nil || !errors.Is(err, mm.ErrInvalidPath) {
		t.Fatalf("eager NewRenderer() = %#v, %v", invalidRenderer, err)
	}
	lazyRenderer, err := mm.NewRenderer(invalidTokens, mm.RendererOptions{})
	if err != nil {
		t.Fatalf("lazy NewRenderer() error = %v", err)
	}
	if got, err := lazyRenderer.Render(mm.Scope{}); got != "" || !errors.Is(err, mm.ErrInvalidPath) {
		t.Fatalf("lazy Renderer.Render() = %q, %v", got, err)
	}
}

func TestRendererRenderReusesTemplateNotData(t *testing.T) {
	renderer, err := mm.Compile("{{value}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	for index, want := range []string{"A", "B", "C"} {
		got, err := renderer.Render(mm.Scope{"value": want})
		if err != nil || got != want {
			t.Fatalf("render %d = %q, %v; want %q", index, got, err, want)
		}
	}

	changing := mm.Scope{}
	if got, _ := renderer.Render(changing); got != "" {
		t.Fatalf("missing render = %q", got)
	}
	changing["value"] = "present"
	if got, _ := renderer.Render(changing); got != "present" {
		t.Fatalf("present render = %q", got)
	}
	delete(changing, "value")
	if got, _ := renderer.Render(changing); got != "" {
		t.Fatalf("missing-again render = %q", got)
	}

	nestedRenderer, err := mm.Compile("{{user.name}}/{{items[1]}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	nested := mm.Scope{"user": mm.Scope{"name": "Ada"}, "items": []any{"zero", "one"}}
	if got, _ := nestedRenderer.Render(nested); got != "Ada/one" {
		t.Fatalf("initial nested render = %q", got)
	}
	nested["user"].(mm.Scope)["name"] = "Lin"
	nested["items"].([]any)[1] = "changed"
	if got, _ := nestedRenderer.Render(nested); got != "Lin/changed" {
		t.Fatalf("mutated nested render = %q", got)
	}
}

func TestCompileSnapshotsOptionsAndDoesNotMutateInputs(t *testing.T) {
	template := "<<value>>"
	options := mm.CompileOptions{TokenizeOptions: mm.TokenizeOptions{Tags: mm.Tags{Open: "<<", Close: ">>"}}}
	wantOptions := options
	renderer, err := mm.Compile(template, options)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if options != wantOptions {
		t.Fatalf("Compile() mutated options: got %#v, want %#v", options, wantOptions)
	}
	options.Tags = mm.Tags{Open: "{{", Close: "}}"}
	options.Explicit = true
	scope := mm.Scope{"value": nil}
	wantScope := mm.Scope{"value": nil}
	got, err := renderer.Render(scope)
	if err != nil || got != "" {
		t.Fatalf("Renderer.Render() = %q, %v; want empty", got, err)
	}
	if template != "<<value>>" || wantOptions.Tags != (mm.Tags{Open: "<<", Close: ">>"}) || !reflect.DeepEqual(scope, wantScope) {
		t.Fatalf("inputs changed: template=%q originalOptions=%#v scope=%#v", template, wantOptions, scope)
	}
}

func TestRendererRenderRejectsInvalidReceivers(t *testing.T) {
	var nilRenderer *mm.Renderer
	if got, err := nilRenderer.Render(mm.Scope{}); got != "" || !errors.Is(err, mm.ErrInvalidRenderer) {
		t.Fatalf("nil Renderer.Render() = %q, %v", got, err)
	}
	var zeroRenderer mm.Renderer
	if got, err := zeroRenderer.Render(mm.Scope{}); got != "" || !errors.Is(err, mm.ErrInvalidRenderer) {
		t.Fatalf("zero Renderer.Render() = %q, %v", got, err)
	}
}

func TestRendererRenderIsDeterministic(t *testing.T) {
	renderer, err := mm.Compile("{{a}}/{{b}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	scope := mm.Scope{"a": map[string]any{"z": 1, "a": 2}, "b": []any{1, nil, "x"}}
	wantScope := mm.Scope{"a": map[string]any{"z": 1, "a": 2}, "b": []any{1, nil, "x"}}
	first, err := renderer.Render(scope)
	if err != nil {
		t.Fatalf("first Render() error = %v", err)
	}
	for iteration := 0; iteration < 100; iteration++ {
		got, err := renderer.Render(scope)
		if err != nil || got != first {
			t.Fatalf("iteration %d = %q, %v; first %q", iteration, got, err, first)
		}
	}
	if !reflect.DeepEqual(scope, wantScope) {
		t.Fatalf("Renderer.Render mutated scope: %#v", scope)
	}
}

func TestRendererRenderIsSafeForConcurrentReadOnlyCalls(t *testing.T) {
	renderer, err := mm.Compile("{{user.name}}={{items}}", mm.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	const workers = 64
	scope := mm.Scope{"user": mm.Scope{"name": "Ada"}, "items": []any{1, 2, 3}}
	errorsFound := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			got, err := renderer.Render(scope)
			if err != nil {
				errorsFound <- err
			} else if got != "Ada=1,2,3" {
				errorsFound <- errors.New("unexpected concurrent Renderer.Render result: " + got)
			}
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
}

func assertErrorKindAndText(t *testing.T, err, kind error, text string) {
	t.Helper()
	if !errors.Is(err, kind) {
		t.Fatalf("errors.Is(%v, %v) = false", err, kind)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("error = %q, want substring %q", err, text)
	}
}
