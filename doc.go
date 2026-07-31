// Package micromustache defines the public API for a Go port of micromustache.
//
// Phase 3E implements reusable compiled templates and synchronous data
// rendering in addition to top-level rendering, template tokenization, and
// value lookup. Resolver renderer methods and asynchronous behavior remain
// unimplemented and return ErrNotImplemented. The package has no Node.js
// runtime dependency.
package micromustache
