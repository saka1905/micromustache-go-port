package micromustache

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func parsePath(path string) (Ref, error) {
	if trimJSWhitespace(path) == "" {
		return Ref{}, nil
	}

	ref := make(Ref, 0, 4)
	current := 0
	for {
		var (
			segment string
			next    int
			matched bool
		)

		if segment, next, matched = parseDotSegment(path, current); !matched {
			if segment, next, matched = parseQuotedSegment(path, current); !matched {
				if segment, next, matched = parseNumericSegment(path, current); !matched && current == 0 {
					segment, next, matched = parseInitialSegment(path)
				}
			}
		}

		if !matched {
			break
		}
		ref = append(ref, segment)
		current = next
	}

	if current != len(path) {
		return nil, compatibleError(ErrInvalidPath, `Could not parse path: "%s"`, path)
	}
	return ref, nil
}

func parseDotSegment(path string, start int) (string, int, bool) {
	current := skipJSWhitespace(path, start)
	if current >= len(path) || path[current] != '.' {
		return "", start, false
	}
	current = skipJSWhitespace(path, current+1)
	nameStart := current
	for current < len(path) && isPathNameByte(path[current]) {
		current++
	}
	if current == nameStart {
		return "", start, false
	}
	return path[nameStart:current], skipJSWhitespace(path, current), true
}

func parseQuotedSegment(path string, start int) (string, int, bool) {
	current := skipJSWhitespace(path, start)
	if current >= len(path) || path[current] != '[' {
		return "", start, false
	}
	current = skipJSWhitespace(path, current+1)
	if current >= len(path) || (path[current] != '\'' && path[current] != '"' && path[current] != '`') {
		return "", start, false
	}
	quote := path[current]
	nameStart := current + 1

	for current = nameStart; current < len(path); {
		r, size := utf8.DecodeRuneInString(path[current:])
		if isJSLineTerminator(r) {
			return "", start, false
		}
		if size == 1 && path[current] == quote {
			afterQuote := skipJSWhitespace(path, current+1)
			if afterQuote < len(path) && path[afterQuote] == ']' {
				return path[nameStart:current], skipJSWhitespace(path, afterQuote+1), true
			}
		}
		current += size
	}
	return "", start, false
}

func parseNumericSegment(path string, start int) (string, int, bool) {
	current := skipJSWhitespace(path, start)
	if current >= len(path) || path[current] != '[' {
		return "", start, false
	}
	current = skipJSWhitespace(path, current+1)
	if current < len(path) && path[current] == '+' {
		current = skipJSWhitespace(path, current+1)
	}
	digitStart := current
	for current < len(path) && path[current] >= '0' && path[current] <= '9' {
		current++
	}
	if current == digitStart {
		return "", start, false
	}
	digits := path[digitStart:current]
	afterDigits := skipJSWhitespace(path, current)
	if afterDigits >= len(path) || path[afterDigits] != ']' {
		return "", start, false
	}

	name := strings.TrimLeft(digits, "0")
	if name == "" {
		name = "0"
	}
	if len(name) > 16 {
		return "", start, false
	}
	return name, skipJSWhitespace(path, afterDigits+1), true
}

func parseInitialSegment(path string) (string, int, bool) {
	current := skipJSWhitespace(path, 0)
	nameStart := current
	for current < len(path) && isPathNameByte(path[current]) {
		current++
	}
	if current == nameStart {
		return "", 0, false
	}
	return path[nameStart:current], skipJSWhitespace(path, current), true
}

func isPathNameByte(value byte) bool {
	return value == '$' || value == '_' || value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func trimJSWhitespace(value string) string {
	start := skipJSWhitespace(value, 0)
	end := len(value)
	for end > start {
		r, size := utf8.DecodeLastRuneInString(value[:end])
		if !isJSWhitespace(r) {
			break
		}
		end -= size
	}
	return value[start:end]
}

func skipJSWhitespace(value string, start int) int {
	current := start
	for current < len(value) {
		r, size := utf8.DecodeRuneInString(value[current:])
		if !isJSWhitespace(r) {
			break
		}
		current += size
	}
	return current
}

func isJSWhitespace(r rune) bool {
	return r == '\t' || r == '\v' || r == '\f' || r == ' ' || r == '\u00a0' ||
		r == '\ufeff' || unicode.Is(unicode.Zs, r) || isJSLineTerminator(r)
}

func isJSLineTerminator(r rune) bool {
	return r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029'
}
