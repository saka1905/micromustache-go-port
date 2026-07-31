package micromustache_test

import (
	"errors"
	"reflect"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		template string
		options  mm.TokenizeOptions
		want     mm.Tokens
	}{
		{"no interpolation", "Hello", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"Hello"}, Paths: []string{}}},
		{"empty template", "", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{""}, Paths: []string{}}},
		{"multiple paths", "A{{x}}B{{y}}C", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"A", "B", "C"}, Paths: []string{"x", "y"}}},
		{"dot path stays raw", "{{ user.name }}", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"", ""}, Paths: []string{"user.name"}}},
		{"bracket path stays raw", "{{obj['a.b']}}", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"", ""}, Paths: []string{"obj['a.b']"}}},
		{"invalid path stays raw", "{{a.}}", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"", ""}, Paths: []string{"a."}}},
		{"custom tags", "<% x %>", mm.TokenizeOptions{Tags: mm.Tags{Open: "<%", Close: "%>"}}, mm.Tokens{Strings: []string{"", ""}, Paths: []string{"x"}}},
		{"close without open", "A}}B", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"A}}B"}, Paths: []string{}}},
		{"unicode path", "A{{ ['😀'] }}B", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"A", "B"}, Paths: []string{"['😀']"}}},
		{"javascript whitespace", "A{{\u00a0x\u00a0}}B", mm.TokenizeOptions{}, mm.Tokens{Strings: []string{"A", "B"}, Paths: []string{"x"}}},
		{"emoji is two UTF-16 units", "{{😀}}", mm.TokenizeOptions{MaxPathLen: 2}, mm.Tokens{Strings: []string{"", ""}, Paths: []string{"😀"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			optionsBefore := test.options
			for run := 0; run < 2; run++ {
				got, err := mm.Tokenize(test.template, test.options)
				if err != nil {
					t.Fatalf("Tokenize() error = %v", err)
				}
				if !reflect.DeepEqual(got, test.want) {
					t.Fatalf("Tokenize() = %#v, want %#v", got, test.want)
				}
			}
			if test.options != optionsBefore {
				t.Fatalf("Tokenize() mutated options: got %#v, want %#v", test.options, optionsBefore)
			}
		})
	}
}

func TestTokenizeErrors(t *testing.T) {
	tests := []struct {
		name     string
		template string
		options  mm.TokenizeOptions
		kind     error
		message  string
	}{
		{"missing close", "Hi {{", mm.TokenizeOptions{}, mm.ErrInvalidTemplate, `Missing "}}" in the template for the "{{" at position 3 within 1000 characters`},
		{"empty path", "{{   }}", mm.TokenizeOptions{}, mm.ErrInvalidTemplate, `Unexpected "}}" tag found at position 0`},
		{"nested open", "{{a{{b}}", mm.TokenizeOptions{}, mm.ErrInvalidTemplate, `Path cannot have "{{". But at position 0 got "a{{b"`},
		{"ASCII maxPathLen", "{{abcd}}", mm.TokenizeOptions{MaxPathLen: 3}, mm.ErrInvalidTemplate, `Missing "}}" in the template for the "{{" at position 0 within 3 characters`},
		{"emoji maxPathLen", "{{😀}}", mm.TokenizeOptions{MaxPathLen: 1}, mm.ErrInvalidTemplate, `Missing "}}" in the template for the "{{" at position 0 within 1 characters`},
		{"UTF-16 position", "😀{{x", mm.TokenizeOptions{}, mm.ErrInvalidTemplate, `Missing "}}" in the template for the "{{" at position 2 within 1000 characters`},
		{"equal tags", "", mm.TokenizeOptions{Tags: mm.Tags{Open: "|", Close: "|"}}, mm.ErrInvalidOption, `The open and close symbols should be two distinct non-empty strings which don't contain each other. Got "|" and "|"`},
		{"open contains close", "", mm.TokenizeOptions{Tags: mm.Tags{Open: "{{", Close: "{"}}, mm.ErrInvalidOption, `The open and close symbols should be two distinct non-empty strings which don't contain each other. Got "{{" and "{"`},
		{"one empty tag", "", mm.TokenizeOptions{Tags: mm.Tags{Open: "{{"}}, mm.ErrInvalidOption, `The open and close symbols should be two distinct non-empty strings which don't contain each other. Got "{{" and ""`},
		{"negative maxPathLen", "", mm.TokenizeOptions{MaxPathLen: -1}, mm.ErrInvalidOption, `Expected a positive number for maxPathLen. Got -1`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := mm.Tokenize(test.template, test.options)
			if !reflect.DeepEqual(got, mm.Tokens{}) {
				t.Fatalf("Tokenize() result = %#v, want zero Tokens", got)
			}
			if !errors.Is(err, test.kind) {
				t.Fatalf("errors.Is(%v, %v) = false", err, test.kind)
			}
			if err.Error() != test.message {
				t.Fatalf("error = %q, want %q", err, test.message)
			}
		})
	}
}
