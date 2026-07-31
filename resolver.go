package micromustache

import "context"

// Resolver resolves one path for synchronous RenderFunc operations.
type Resolver func(path string, scope Scope) (Value, error)

// AsyncResolver resolves one path for asynchronous RenderFuncAsync operations.
// The context carries cancellation and deadlines without changing upstream
// resolution semantics.
type AsyncResolver func(ctx context.Context, path string, scope Scope) (Value, error)
