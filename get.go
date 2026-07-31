package micromustache

import "strings"

const defaultMaxRefDepth = 10

// Get parses a string path and resolves the resulting reference from scope.
func Get(scope Scope, path string, options GetOptions) (Value, error) {
	ref, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	return GetRef(scope, ref, options)
}

// GetRef resolves an already-segmented reference from a scope. A missing value
// returns Undefined unless ValidateRef requests an ErrReference instead.
func GetRef(scope Scope, ref Ref, options GetOptions) (Value, error) {
	maxRefDepth := options.MaxRefDepth
	if maxRefDepth == 0 {
		maxRefDepth = defaultMaxRefDepth
	}
	if maxRefDepth < 0 {
		return nil, compatibleError(
			ErrInvalidOption,
			"Expected a positive number for maxRefDepth. Got %d",
			options.MaxRefDepth,
		)
	}

	joinedRef := func() string { return strings.Join(ref, " > ") }
	if len(ref) > maxRefDepth {
		return nil, compatibleError(
			ErrReference,
			`The ref cannot be deeper than %d levels. Got "%s"`,
			maxRefDepth,
			joinedRef(),
		)
	}

	var current Value = scope
	for _, property := range ref {
		next, found := lookupValue(current, property)
		if !found {
			if options.ValidateRef {
				return nil, compatibleError(
					ErrReference,
					`%s is not defined in the scope at ref: "%s"`,
					property,
					joinedRef(),
				)
			}
			return Undefined{}, nil
		}
		current = next
	}
	return current, nil
}

func lookupValue(value Value, property string) (Value, bool) {
	switch current := value.(type) {
	case Scope:
		next, found := current[property]
		return next, found
	case map[string]any:
		next, found := current[property]
		return next, found
	case []any:
		if property == "length" {
			return len(current), true
		}
		index, ok := canonicalArrayIndex(property)
		if !ok || index >= len(current) {
			return nil, false
		}
		return current[index], true
	default:
		return nil, false
	}
}

func canonicalArrayIndex(property string) (int, bool) {
	if property == "0" {
		return 0, true
	}
	if len(property) == 0 || property[0] < '1' || property[0] > '9' {
		return 0, false
	}

	const maxInt = int(^uint(0) >> 1)
	index := 0
	for i := 0; i < len(property); i++ {
		if property[i] < '0' || property[i] > '9' {
			return 0, false
		}
		digit := int(property[i] - '0')
		if index > (maxInt-digit)/10 {
			return 0, false
		}
		index = index*10 + digit
	}
	return index, true
}
