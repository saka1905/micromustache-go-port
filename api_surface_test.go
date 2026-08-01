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
	_ func(*mm.Renderer, mm.Scope) (string, error)                                                 = (*mm.Renderer).Render
	_ func(*mm.Renderer, mm.Resolver, mm.Scope) (string, error)                                    = (*mm.Renderer).RenderFunc
	_ func(*mm.Renderer, context.Context, mm.AsyncResolver, mm.Scope) (string, error)              = (*mm.Renderer).RenderFuncAsync
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
		{"RenderFuncAsync", func() error {
			_, err := mm.RenderFuncAsync(context.Background(), "", func(context.Context, string, mm.Scope) (mm.Value, error) { return mm.Undefined{}, nil }, mm.Scope{}, mm.CompileOptions{})
			return err
		}},
		{"Compile", func() error { _, err := mm.Compile("", mm.CompileOptions{}); return err }},
		{"NewRenderer", func() error {
			_, err := mm.NewRenderer(mm.Tokens{Strings: []string{""}}, mm.RendererOptions{})
			return err
		}},
		{"Renderer.Render", func() error {
			renderer, err := mm.NewRenderer(mm.Tokens{Strings: []string{""}}, mm.RendererOptions{})
			if err != nil {
				return err
			}
			_, err = renderer.Render(mm.Scope{})
			return err
		}},
		{"Renderer.RenderFunc", func() error {
			renderer, err := mm.NewRenderer(mm.Tokens{Strings: []string{""}}, mm.RendererOptions{})
			if err != nil {
				return err
			}
			_, err = renderer.RenderFunc(func(string, mm.Scope) (mm.Value, error) { return mm.Undefined{}, nil }, mm.Scope{})
			return err
		}},
		{"Renderer.RenderFuncAsync", func() error {
			renderer, err := mm.NewRenderer(mm.Tokens{Strings: []string{""}}, mm.RendererOptions{})
			if err != nil {
				return err
			}
			_, err = renderer.RenderFuncAsync(context.Background(), func(context.Context, string, mm.Scope) (mm.Value, error) { return mm.Undefined{}, nil }, mm.Scope{})
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
