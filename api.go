package micromustache

import "context"

// RenderFunc renders a template using a caller-provided synchronous resolver.
func RenderFunc(template string, resolve Resolver, scope Scope, options CompileOptions) (string, error) {
	return "", notImplemented("RenderFunc")
}

// RenderFuncAsync renders a template using a caller-provided asynchronous resolver.
func RenderFuncAsync(ctx context.Context, template string, resolve AsyncResolver, scope Scope, options CompileOptions) (string, error) {
	return "", notImplemented("RenderFuncAsync")
}

// Compile tokenizes a template and returns a reusable Renderer.
func Compile(template string, options CompileOptions) (*Renderer, error) {
	return nil, notImplemented("Compile")
}
