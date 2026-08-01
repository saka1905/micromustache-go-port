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
	return renderer.renderFuncAsync(ctx, resolve, scope, "RenderFuncAsync")
}

func (r *Renderer) renderFuncAsync(ctx context.Context, resolve AsyncResolver, scope Scope, api string) (string, error) {
	if r == nil || !r.initialized {
		return "", fmt.Errorf("Renderer: %w", ErrInvalidRenderer)
	}
	if ctx == nil {
		return "", fmt.Errorf("%s context is nil: %w", api, ErrInvalidContext)
	}
	if resolve == nil {
		return "", fmt.Errorf("%s resolver is nil: %w", api, ErrInvalidResolver)
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("%s context: %w", api, err)
	}

	pathCount := len(r.tokens.Paths)
	results := make(chan asyncResolveResult, pathCount)
	for index, path := range r.tokens.Paths {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("%s context: %w", api, err)
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
			return "", fmt.Errorf("%s context: %w", api, err)
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("%s context: %w", api, ctx.Err())
		case result := <-results:
			if err := ctx.Err(); err != nil {
				return "", fmt.Errorf("%s context: %w", api, err)
			}
			if result.err != nil {
				return "", fmt.Errorf("%s resolver for path %q at index %d: %w", api, r.tokens.Paths[result.index], result.index, result.err)
			}
			values[result.index] = result.value
		}
	}

	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("%s context: %w", api, err)
	}
	return stringifyTokens(r.tokens, values, r.options.Explicit)
}
