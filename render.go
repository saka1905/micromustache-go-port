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

	refs := make([]Ref, len(tokens.Paths))
	for index, path := range tokens.Paths {
		refs[index], err = parsePath(path)
		if err != nil {
			return "", err
		}
	}

	values := make([]Value, len(refs))
	for index, ref := range refs {
		values[index], err = GetRef(scope, ref, options.GetOptions)
		if err != nil {
			return "", err
		}
	}

	var result strings.Builder
	for index, value := range values {
		result.WriteString(tokens.Strings[index])
		if err := appendJSValue(&result, value, options.Explicit); err != nil {
			return "", err
		}
	}
	result.WriteString(tokens.Strings[len(tokens.Strings)-1])
	return result.String(), nil
}
