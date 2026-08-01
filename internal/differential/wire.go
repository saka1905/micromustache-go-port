package differential

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	mm "github.com/saka1905/micromustache-go-port"
)

type Case struct {
	ID           string          `json:"id"`
	Op           string          `json:"op"`
	Args         json.RawMessage `json:"args"`
	Category     string          `json:"category,omitempty"`
	DifferenceID string          `json:"differenceId,omitempty"`
	SkipReason   string          `json:"skipReason,omitempty"`
}

type Request struct {
	ID   string          `json:"id"`
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args"`
}

type Response struct {
	ID    *string         `json:"id"`
	OK    bool            `json:"ok"`
	Value json.RawMessage `json:"value,omitempty"`
	Error *ErrorEnvelope  `json:"error,omitempty"`
}

type ErrorEnvelope struct {
	Name         string `json:"name"`
	Message      string `json:"message"`
	Category     string `json:"category,omitempty"`
	CauseName    string `json:"causeName,omitempty"`
	CauseMessage string `json:"causeMessage,omitempty"`
}

type valueEnvelope struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value,omitempty"`
}

func DecodeValue(raw json.RawMessage) (any, error) {
	var node valueEnvelope
	if err := decodeStrict(raw, &node); err != nil {
		return nil, fmt.Errorf("invalid encoded value: %w", err)
	}
	switch node.Type {
	case "undefined":
		return mm.Undefined{}, requireNoValue(node)
	case "null":
		return nil, requireNoValue(node)
	case "boolean":
		var value bool
		return value, decodeRequired(node.Value, &value)
	case "number":
		var value float64
		if err := decodeRequired(node.Value, &value); err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			if err == nil {
				err = errors.New("number must be finite")
			}
			return nil, err
		}
		return value, nil
	case "nan":
		return math.NaN(), requireNoValue(node)
	case "infinity":
		return math.Inf(1), requireNoValue(node)
	case "negative-infinity":
		return math.Inf(-1), requireNoValue(node)
	case "negative-zero":
		return math.Copysign(0, -1), requireNoValue(node)
	case "string":
		var value string
		return value, decodeRequired(node.Value, &value)
	case "array":
		var values []json.RawMessage
		if err := decodeRequired(node.Value, &values); err != nil {
			return nil, err
		}
		result := make([]any, len(values))
		for index, value := range values {
			decoded, err := DecodeValue(value)
			if err != nil {
				return nil, fmt.Errorf("array index %d: %w", index, err)
			}
			result[index] = decoded
		}
		return result, nil
	case "object":
		var values map[string]json.RawMessage
		if err := decodeRequired(node.Value, &values); err != nil {
			return nil, err
		}
		result := make(map[string]any, len(values))
		for key, value := range values {
			decoded, err := DecodeValue(value)
			if err != nil {
				return nil, fmt.Errorf("object key %q: %w", key, err)
			}
			result[key] = decoded
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unknown type %q", node.Type)
	}
}

func EncodeValue(value any) (json.RawMessage, error) {
	node, err := encodedNode(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(node)
}

func encodedNode(value any) (any, error) {
	switch current := value.(type) {
	case mm.Undefined:
		return map[string]any{"type": "undefined"}, nil
	case nil:
		return map[string]any{"type": "null"}, nil
	case bool:
		return map[string]any{"type": "boolean", "value": current}, nil
	case float64:
		switch {
		case math.IsNaN(current):
			return map[string]any{"type": "nan"}, nil
		case math.IsInf(current, 1):
			return map[string]any{"type": "infinity"}, nil
		case math.IsInf(current, -1):
			return map[string]any{"type": "negative-infinity"}, nil
		case current == 0 && math.Signbit(current):
			return map[string]any{"type": "negative-zero"}, nil
		default:
			return map[string]any{"type": "number", "value": current}, nil
		}
	case float32:
		return encodedNode(float64(current))
	case int:
		return map[string]any{"type": "number", "value": current}, nil
	case int64:
		return map[string]any{"type": "number", "value": current}, nil
	case string:
		return map[string]any{"type": "string", "value": current}, nil
	case []string:
		values := make([]any, len(current))
		for index, item := range current {
			values[index] = item
		}
		return encodedNode(values)
	case []any:
		values := make([]any, len(current))
		for index, item := range current {
			encoded, err := encodedNode(item)
			if err != nil {
				return nil, fmt.Errorf("array index %d: %w", index, err)
			}
			values[index] = encoded
		}
		return map[string]any{"type": "array", "value": values}, nil
	case mm.Scope:
		values := make(map[string]any, len(current))
		for key, item := range current {
			values[key] = item
		}
		return encodedNode(values)
	case map[string]any:
		values := make(map[string]any, len(current))
		for key, item := range current {
			encoded, err := encodedNode(item)
			if err != nil {
				return nil, fmt.Errorf("object key %q: %w", key, err)
			}
			values[key] = encoded
		}
		return map[string]any{"type": "object", "value": values}, nil
	case mm.Tokens:
		return encodedNode(map[string]any{"strings": current.Strings, "paths": current.Paths})
	default:
		return nil, fmt.Errorf("unsupported Go value type %T", value)
	}
}

func DecodeScope(raw json.RawMessage) (mm.Scope, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return mm.Scope{}, nil
	}
	value, err := DecodeValue(raw)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("scope must decode to an object")
	}
	result := make(mm.Scope, len(object))
	for key, item := range object {
		result[key] = item
	}
	return result, nil
}

type wireOptions struct {
	ValidateRef  bool     `json:"validateRef"`
	MaxRefDepth  int      `json:"maxRefDepth"`
	Explicit     bool     `json:"explicit"`
	ValidatePath bool     `json:"validatePath"`
	MaxPathLen   int      `json:"maxPathLen"`
	Tags         []string `json:"tags"`
}

func DecodeCompileOptions(raw json.RawMessage) (mm.CompileOptions, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return mm.CompileOptions{}, nil
	}
	var options wireOptions
	if err := decodeStrict(raw, &options); err != nil {
		return mm.CompileOptions{}, err
	}
	if len(options.Tags) != 0 && len(options.Tags) != 2 {
		return mm.CompileOptions{}, errors.New("tags must contain exactly two strings")
	}
	result := mm.CompileOptions{
		RendererOptions: mm.RendererOptions{
			GetOptions:   mm.GetOptions{ValidateRef: options.ValidateRef, MaxRefDepth: options.MaxRefDepth},
			Explicit:     options.Explicit,
			ValidatePath: options.ValidatePath,
		},
		TokenizeOptions: mm.TokenizeOptions{MaxPathLen: options.MaxPathLen},
	}
	if len(options.Tags) == 2 {
		result.Tags = mm.Tags{Open: options.Tags[0], Close: options.Tags[1]}
	}
	return result, nil
}

func DecodeRendererOptions(raw json.RawMessage) (mm.RendererOptions, error) {
	options, err := DecodeCompileOptions(raw)
	return options.RendererOptions, err
}

func DecodeGetOptions(raw json.RawMessage) (mm.GetOptions, error) {
	options, err := DecodeCompileOptions(raw)
	return options.GetOptions, err
}

func DecodeTokenizeOptions(raw json.RawMessage) (mm.TokenizeOptions, error) {
	options, err := DecodeCompileOptions(raw)
	return options.TokenizeOptions, err
}

type wireTokens struct {
	Strings []string `json:"strings"`
	Paths   []string `json:"paths"`
}

func (tokens wireTokens) Go() mm.Tokens {
	return mm.Tokens{Strings: append([]string(nil), tokens.Strings...), Paths: append([]string(nil), tokens.Paths...)}
}

func decodeStrict(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

func decodeRequired(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return errors.New("missing value")
	}
	return decodeStrict(raw, target)
}

func requireNoValue(node valueEnvelope) error {
	if len(node.Value) != 0 {
		return errors.New("unexpected value")
	}
	return nil
}
