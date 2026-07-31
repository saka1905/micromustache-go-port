package micromustache

import (
	"errors"
	"fmt"
)

// ErrNotImplemented marks every API operation whose behavior is not implemented yet.
var ErrNotImplemented = errors.New("micromustache: not implemented")

func notImplemented(api string) error {
	return fmt.Errorf("%s: %w", api, ErrNotImplemented)
}
