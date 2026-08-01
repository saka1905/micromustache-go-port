package micromustache

import (
	"context"
	"fmt"
)

type asyncResolveResult struct {
	index int
	value Value
	err   error
}

// RenderFuncAsync interpolates a template using concurrently dispatched
// resolver calls and preserves interpolation order in the rendered output.
func RenderFuncAsync(ctx context.Context, template string, resolve AsyncResolver, scope Scope, options CompileOptions) (string, error) {
	renderer, err := Compile(template, options)
	if err != nil {
		return "", err
	}
	if ctx == nil {
		return "", fmt.Errorf("RenderFuncAsync context is nil: %w", ErrInvalidContext)
	}
	if resolve == nil {
		return "", fmt.Errorf("RenderFuncAsync resolver is nil: %w", ErrInvalidResolver)
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("RenderFuncAsync context: %w", err)
	}

	pathCount := len(renderer.tokens.Paths)
	results := make(chan asyncResolveResult, pathCount)
	for index, path := range renderer.tokens.Paths {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("RenderFuncAsync context: %w", err)
		}

		started := make(chan struct{})
		go func(index int, path string) {
			close(started)
			value, err := resolve(ctx, path, scope)
			results <- asyncResolveResult{index: index, value: value, err: err}
		}(index, path)
		<-started
	}

	values := make([]Value, pathCount)
	for received := 0; received < pathCount; received++ {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("RenderFuncAsync context: %w", err)
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("RenderFuncAsync context: %w", ctx.Err())
		case result := <-results:
			if err := ctx.Err(); err != nil {
				return "", fmt.Errorf("RenderFuncAsync context: %w", err)
			}
			if result.err != nil {
				return "", fmt.Errorf("RenderFuncAsync resolver for path %q at index %d: %w", renderer.tokens.Paths[result.index], result.index, result.err)
			}
			values[result.index] = result.value
		}
	}

	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("RenderFuncAsync context: %w", err)
	}
	return stringifyTokens(renderer.tokens, values, renderer.options.Explicit)
}
