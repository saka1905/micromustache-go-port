package micromustache

import (
	"errors"
	"fmt"
)

var (
	// ErrNotImplemented marks every API operation whose behavior is not implemented yet.
	ErrNotImplemented = errors.New("micromustache: not implemented")
	// ErrInvalidTemplate marks template tokenization syntax errors.
	ErrInvalidTemplate = errors.New("micromustache: invalid template")
	// ErrInvalidPath marks reference path syntax errors.
	ErrInvalidPath = errors.New("micromustache: invalid path")
	// ErrInvalidOption marks an option value outside its supported range.
	ErrInvalidOption = errors.New("micromustache: invalid option")
	// ErrReference marks a missing or over-depth reference when validation is required.
	ErrReference = errors.New("micromustache: reference error")
	// ErrUnsupportedValue marks a Go value that has no deterministic JavaScript string coercion mapping.
	ErrUnsupportedValue = errors.New("micromustache: unsupported value")
	// ErrInvalidResolver marks a nil synchronous resolver supplied to a RenderFunc operation.
	ErrInvalidResolver = errors.New("micromustache: invalid resolver")
	// ErrInvalidTokens marks a Tokens value that cannot represent a Renderer template.
	ErrInvalidTokens = errors.New("micromustache: invalid tokens")
	// ErrInvalidRenderer marks a nil or zero-value Renderer receiver.
	ErrInvalidRenderer = errors.New("micromustache: invalid renderer")
)

type compatibilityError struct {
	kind    error
	message string
}

func (e *compatibilityError) Error() string {
	return e.message
}

func (e *compatibilityError) Unwrap() error {
	return e.kind
}

func compatibleError(kind error, format string, args ...any) error {
	return &compatibilityError{kind: kind, message: fmt.Sprintf(format, args...)}
}

func notImplemented(api string) error {
	return fmt.Errorf("%s: %w", api, ErrNotImplemented)
}
