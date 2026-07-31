package micromustache

import "context"

// RenderFuncAsync renders a template using a caller-provided asynchronous resolver.
func RenderFuncAsync(ctx context.Context, template string, resolve AsyncResolver, scope Scope, options CompileOptions) (string, error) {
	return "", notImplemented("RenderFuncAsync")
}

// Compile tokenizes a template and returns a reusable Renderer.
func Compile(template string, options CompileOptions) (*Renderer, error) {
	tokens, err := Tokenize(template, options.TokenizeOptions)
	if err != nil {
		return nil, err
	}
	return NewRenderer(tokens, options.RendererOptions)
}
