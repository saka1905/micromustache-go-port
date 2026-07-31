package micromustache

import "fmt"

// RenderFunc synchronously interpolates template using one resolver call for
// every raw path occurrence. Paths are parsed only when ValidatePath is set,
// matching the fixed upstream renderFn control flow.
func RenderFunc(template string, resolve Resolver, scope Scope, options CompileOptions) (string, error) {
	tokens, err := Tokenize(template, options.TokenizeOptions)
	if err != nil {
		return "", err
	}

	if options.ValidatePath {
		if _, err := parseTokenPaths(tokens); err != nil {
			return "", err
		}
	}
	if resolve == nil {
		return "", fmt.Errorf("RenderFunc resolver is nil: %w", ErrInvalidResolver)
	}

	values := make([]Value, len(tokens.Paths))
	for index, path := range tokens.Paths {
		value, err := resolve(path, scope)
		if err != nil {
			return "", fmt.Errorf("RenderFunc resolver for path %q at index %d: %w", path, index, err)
		}
		values[index] = value
	}

	return stringifyTokens(tokens, values, options.Explicit)
}
