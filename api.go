package micromustache

import "context"

// Render renders a template with the default property resolver.
func Render(template string, scope Scope, options CompileOptions) (string, error) {
	return "", notImplemented("Render")
}

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

// Get resolves a string path from a scope.
func Get(scope Scope, path string, options GetOptions) (Value, error) {
	return nil, notImplemented("Get")
}

// GetRef resolves an already-segmented reference from a scope.
func GetRef(scope Scope, ref Ref, options GetOptions) (Value, error) {
	return nil, notImplemented("GetRef")
}

// Tokenize separates a template into literal strings and property paths.
func Tokenize(template string, options TokenizeOptions) (Tokens, error) {
	return Tokens{}, notImplemented("Tokenize")
}
