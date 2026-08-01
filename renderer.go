package micromustache

import (
	"context"
	"fmt"
	"sync"
)

// Renderer is the reusable compiled-template object exposed by the upstream API.
// It caches template-scoped parsed references but never data or render results.
type Renderer struct {
	tokens      Tokens
	options     RendererOptions
	refsOnce    sync.Once
	refs        []Ref
	refsErr     error
	initialized bool
}

// NewRenderer maps the upstream public Renderer constructor.
func NewRenderer(tokens Tokens, options RendererOptions) (*Renderer, error) {
	if len(tokens.Strings) != len(tokens.Paths)+1 {
		return nil, compatibleError(ErrInvalidTokens, "Invalid tokens object")
	}

	renderer := &Renderer{
		tokens: Tokens{
			Strings: append([]string(nil), tokens.Strings...),
			Paths:   append([]string(nil), tokens.Paths...),
		},
		options:     options,
		initialized: true,
	}
	if options.ValidatePath {
		if _, err := renderer.parsedRefs(); err != nil {
			return nil, err
		}
	}
	return renderer, nil
}

// Render renders a compiled template with the default property resolver.
func (r *Renderer) Render(scope Scope) (string, error) {
	refs, err := r.parsedRefs()
	if err != nil {
		return "", err
	}

	values := make([]Value, len(refs))
	for index, ref := range refs {
		values[index], err = GetRef(scope, ref, r.options.GetOptions)
		if err != nil {
			return "", err
		}
	}
	return stringifyTokens(r.tokens, values, r.options.Explicit)
}

func (r *Renderer) parsedRefs() ([]Ref, error) {
	if r == nil || !r.initialized {
		return nil, fmt.Errorf("Renderer: %w", ErrInvalidRenderer)
	}
	r.refsOnce.Do(func() {
		r.refs, r.refsErr = parseTokenPaths(r.tokens)
	})
	return r.refs, r.refsErr
}

// RenderFunc renders a compiled template with a synchronous resolver.
func (r *Renderer) RenderFunc(resolve Resolver, scope Scope) (string, error) {
	return r.renderFunc(resolve, scope, "Renderer.RenderFunc")
}

// RenderFuncAsync renders a compiled template with an asynchronous resolver.
func (r *Renderer) RenderFuncAsync(ctx context.Context, resolve AsyncResolver, scope Scope) (string, error) {
	return "", notImplemented("Renderer.RenderFuncAsync")
}
