package micromustache

import "fmt"

// RenderFunc synchronously interpolates template using one resolver call for
// every raw path occurrence. Paths are parsed only when ValidatePath is set,
// matching the fixed upstream renderFn control flow.
func RenderFunc(template string, resolve Resolver, scope Scope, options CompileOptions) (string, error) {
	renderer, err := Compile(template, options)
	if err != nil {
		return "", err
	}
	return renderer.renderFunc(resolve, scope, "RenderFunc")
}

func (r *Renderer) renderFunc(resolve Resolver, scope Scope, api string) (string, error) {
	if r == nil || !r.initialized {
		return "", fmt.Errorf("Renderer: %w", ErrInvalidRenderer)
	}
	if resolve == nil {
		return "", fmt.Errorf("%s resolver is nil: %w", api, ErrInvalidResolver)
	}

	values := make([]Value, len(r.tokens.Paths))
	for index, path := range r.tokens.Paths {
		value, err := resolve(path, scope)
		if err != nil {
			return "", fmt.Errorf("%s resolver for path %q at index %d: %w", api, path, index, err)
		}
		values[index] = value
	}

	return stringifyTokens(r.tokens, values, r.options.Explicit)
}
