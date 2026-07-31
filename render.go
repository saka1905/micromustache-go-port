package micromustache

import "strings"

// Render synchronously interpolates every path in template using values from
// scope. It mirrors the fixed upstream order: tokenize, parse all paths,
// resolve every reference, then stringify and concatenate.
func Render(template string, scope Scope, options CompileOptions) (string, error) {
	tokens, err := Tokenize(template, options.TokenizeOptions)
	if err != nil {
		return "", err
	}

	refs, err := parseTokenPaths(tokens)
	if err != nil {
		return "", err
	}

	values := make([]Value, len(refs))
	for index, ref := range refs {
		values[index], err = GetRef(scope, ref, options.GetOptions)
		if err != nil {
			return "", err
		}
	}

	return stringifyTokens(tokens, values, options.Explicit)
}

func parseTokenPaths(tokens Tokens) ([]Ref, error) {
	refs := make([]Ref, len(tokens.Paths))
	for index, path := range tokens.Paths {
		ref, err := parsePath(path)
		if err != nil {
			return nil, err
		}
		refs[index] = ref
	}
	return refs, nil
}

func stringifyTokens(tokens Tokens, values []Value, explicit bool) (string, error) {
	var result strings.Builder
	for index, value := range values {
		result.WriteString(tokens.Strings[index])
		if err := appendJSValue(&result, value, explicit); err != nil {
			return "", err
		}
	}
	result.WriteString(tokens.Strings[len(tokens.Strings)-1])
	return result.String(), nil
}
