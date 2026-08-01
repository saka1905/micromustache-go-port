// Package micromustache defines the public API for a Go port of micromustache.
//
// Phase 3F implements reusable compiled templates with synchronous data and
// resolver rendering in addition to top-level rendering, tokenization, and
// value lookup. Asynchronous operations remain unimplemented and return
// ErrNotImplemented. The package has no Node.js runtime dependency.
package micromustache
