package micromustache_test

import (
	"context"
	"errors"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

var (
	_ func(string, mm.Scope, mm.CompileOptions) (string, error)                                    = mm.Render
	_ func(string, mm.Resolver, mm.Scope, mm.CompileOptions) (string, error)                       = mm.RenderFunc
	_ func(context.Context, string, mm.AsyncResolver, mm.Scope, mm.CompileOptions) (string, error) = mm.RenderFuncAsync
	_ func(string, mm.CompileOptions) (*mm.Renderer, error)                                        = mm.Compile
	_ func(mm.Scope, string, mm.GetOptions) (mm.Value, error)                                      = mm.Get
	_ func(mm.Scope, mm.Ref, mm.GetOptions) (mm.Value, error)                                      = mm.GetRef
	_ func(string, mm.TokenizeOptions) (mm.Tokens, error)                                          = mm.Tokenize
	_ func(mm.Tokens, mm.RendererOptions) (*mm.Renderer, error)                                    = mm.NewRenderer
)

func TestValueStatesRemainDistinct(t *testing.T) {
	scope := mm.Scope{
		"undefined": mm.Undefined{},
		"null":      nil,
		"false":     false,
		"zero":      0,
		"empty":     "",
	}

	if _, ok := scope["missing"]; ok {
		t.Fatal("missing key must remain absent")
	}
	if _, ok := scope["undefined"].(mm.Undefined); !ok {
		t.Fatal("undefined marker lost its type")
	}
	if value, ok := scope["null"]; !ok || value != nil {
		t.Fatal("null must remain a present nil value")
	}
	if scope["false"] != false || scope["zero"] != 0 || scope["empty"] != "" {
		t.Fatal("ordinary falsy values must remain unchanged")
	}
}

func TestImplementedAPIsDoNotReturnErrNotImplemented(t *testing.T) {
	tests := []struct {
		name string
		call func() error
	}{
		{"Tokenize", func() error { _, err := mm.Tokenize("", mm.TokenizeOptions{}); return err }},
		{"GetRef", func() error { _, err := mm.GetRef(mm.Scope{}, mm.Ref{}, mm.GetOptions{}); return err }},
		{"Get", func() error { _, err := mm.Get(mm.Scope{}, "", mm.GetOptions{}); return err }},
		{"Render", func() error { _, err := mm.Render("", mm.Scope{}, mm.CompileOptions{}); return err }},
		{"RenderFunc", func() error {
			_, err := mm.RenderFunc("", func(string, mm.Scope) (mm.Value, error) { return mm.Undefined{}, nil }, mm.Scope{}, mm.CompileOptions{})
			return err
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err != nil {
				t.Fatalf("implemented API returned error: %v", err)
			} else if errors.Is(err, mm.ErrNotImplemented) {
				t.Fatal("implemented API returned ErrNotImplemented")
			}
		})
	}
}

func TestUnimplementedAPIsReturnErrNotImplemented(t *testing.T) {
	ctx := context.Background()
	scope := mm.Scope{}
	resolver := func(string, mm.Scope) (mm.Value, error) {
		t.Fatal("skeleton must not invoke the resolver")
		return nil, nil
	}
	asyncResolver := func(context.Context, string, mm.Scope) (mm.Value, error) {
		t.Fatal("skeleton must not invoke the async resolver")
		return nil, nil
	}

	var renderer mm.Renderer
	cases := []struct {
		name string
		call func() error
	}{
		{"RenderFuncAsync", func() error {
			value, err := mm.RenderFuncAsync(ctx, "", asyncResolver, scope, mm.CompileOptions{})
			assertEmptyString(t, value)
			return err
		}},
		{"Compile", func() error {
			value, err := mm.Compile("", mm.CompileOptions{})
			if value != nil {
				t.Error("Compile returned a non-nil renderer")
			}
			return err
		}},
		{"NewRenderer", func() error {
			value, err := mm.NewRenderer(mm.Tokens{}, mm.RendererOptions{})
			if value != nil {
				t.Error("NewRenderer returned a non-nil renderer")
			}
			return err
		}},
		{"Renderer.Render", func() error { value, err := renderer.Render(scope); assertEmptyString(t, value); return err }},
		{"Renderer.RenderFunc", func() error {
			value, err := renderer.RenderFunc(resolver, scope)
			assertEmptyString(t, value)
			return err
		}},
		{"Renderer.RenderFuncAsync", func() error {
			value, err := renderer.RenderFuncAsync(ctx, asyncResolver, scope)
			assertEmptyString(t, value)
			return err
		}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, mm.ErrNotImplemented) {
				t.Fatalf("errors.Is(err, ErrNotImplemented) = false; err = %v", err)
			}
		})
	}
}

func assertEmptyString(t *testing.T, value string) {
	t.Helper()
	if value != "" {
		t.Errorf("got %q, want empty string", value)
	}
}
