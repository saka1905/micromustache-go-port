package micromustache

import "strings"

const defaultMaxPathLen = 1000

var defaultTags = Tags{Open: "{{", Close: "}}"}

// Tokenize separates a template into literal strings and unparsed property
// paths. Paths are trimmed but their dot or bracket syntax is handled by Get.
func Tokenize(template string, options TokenizeOptions) (Tokens, error) {
	tags := options.Tags
	if tags.Open == "" && tags.Close == "" {
		tags = defaultTags
	}
	if tags.Open == "" || tags.Close == "" || tags.Open == tags.Close ||
		strings.Contains(tags.Open, tags.Close) || strings.Contains(tags.Close, tags.Open) {
		return Tokens{}, compatibleError(
			ErrInvalidOption,
			`The open and close symbols should be two distinct non-empty strings which don't contain each other. Got "%s" and "%s"`,
			tags.Open,
			tags.Close,
		)
	}

	maxPathLen := options.MaxPathLen
	if maxPathLen == 0 {
		maxPathLen = defaultMaxPathLen
	}
	if maxPathLen < 0 {
		return Tokens{}, compatibleError(
			ErrInvalidOption,
			"Expected a positive number for maxPathLen. Got %d",
			options.MaxPathLen,
		)
	}

	stringsResult := make([]string, 0, 4)
	paths := make([]string, 0, 3)
	current := 0
	lastClose := 0

	for current < len(template) {
		openRelative := strings.Index(template[current:], tags.Open)
		if openRelative < 0 {
			break
		}
		openIndex := current + openRelative
		refStart := openIndex + len(tags.Open)
		closeRelative := strings.Index(template[refStart:], tags.Close)
		openPosition := utf16Length(template[:openIndex])
		if closeRelative < 0 || utf16Length(template[refStart:refStart+closeRelative]) > maxPathLen {
			return Tokens{}, compatibleError(
				ErrInvalidTemplate,
				`Missing "%s" in the template for the "%s" at position %d within %d characters`,
				tags.Close,
				tags.Open,
				openPosition,
				maxPathLen,
			)
		}

		closeIndex := refStart + closeRelative
		path := trimJSWhitespace(template[refStart:closeIndex])
		if path == "" {
			return Tokens{}, compatibleError(
				ErrInvalidTemplate,
				`Unexpected "%s" tag found at position %d`,
				tags.Close,
				openPosition,
			)
		}
		if strings.Contains(path, tags.Open) {
			return Tokens{}, compatibleError(
				ErrInvalidTemplate,
				`Path cannot have "%s". But at position %d got "%s"`,
				tags.Open,
				openPosition,
				path,
			)
		}

		stringsResult = append(stringsResult, template[current:openIndex])
		paths = append(paths, path)
		lastClose = closeIndex + len(tags.Close)
		current = lastClose
	}

	stringsResult = append(stringsResult, template[lastClose:])
	return Tokens{Strings: stringsResult, Paths: paths}, nil
}

func utf16Length(value string) int {
	length := 0
	for _, r := range value {
		if r > 0xffff {
			length += 2
		} else {
			length++
		}
	}
	return length
}
