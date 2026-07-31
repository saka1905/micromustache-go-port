// Package micromustache defines the public API for a Go port of micromustache.
//
// Phase 3C implements top-level synchronous rendering in addition to template
// tokenization and value lookup. Compilation, renderer methods, resolver
// callbacks, and asynchronous behavior remain unimplemented and return
// ErrNotImplemented. The package has no Node.js runtime dependency.
package micromustache
