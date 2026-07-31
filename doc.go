// Package micromustache defines the public API for a Go port of micromustache.
//
// Phase 3D implements top-level synchronous resolver rendering in addition to
// ordinary rendering, template tokenization, and value lookup. Compilation,
// renderer methods, and asynchronous behavior remain unimplemented and return
// ErrNotImplemented. The package has no Node.js runtime dependency.
package micromustache
