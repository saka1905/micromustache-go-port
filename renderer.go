package micromustache

import "context"

// Renderer is the reusable compiled-template object exposed by the upstream API.
// Its state and behavior remain deliberately absent in Phase 3B.
type Renderer struct{}

// NewRenderer maps the upstream public Renderer constructor.
func NewRenderer(tokens Tokens, options RendererOptions) (*Renderer, error) {
	return nil, notImplemented("NewRenderer")
}

// Render renders a compiled template with the default property resolver.
func (r *Renderer) Render(scope Scope) (string, error) {
	return "", notImplemented("Renderer.Render")
}

// RenderFunc renders a compiled template with a synchronous resolver.
func (r *Renderer) RenderFunc(resolve Resolver, scope Scope) (string, error) {
	return "", notImplemented("Renderer.RenderFunc")
}

// RenderFuncAsync renders a compiled template with an asynchronous resolver.
func (r *Renderer) RenderFuncAsync(ctx context.Context, resolve AsyncResolver, scope Scope) (string, error) {
	return "", notImplemented("Renderer.RenderFuncAsync")
}
