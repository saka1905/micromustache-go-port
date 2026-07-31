package micromustache

// Value represents a JavaScript-compatible value at the public API boundary.
// A nil Value represents JavaScript null. Undefined is represented explicitly
// by Undefined so that it remains distinct from a missing Scope key.
type Value any

// Scope maps property names to values. An absent key represents a missing
// property; a present key may independently contain nil or Undefined{}.
type Scope map[string]Value

// Ref is the segmented form of a property path accepted by GetRef.
type Ref []string

// Undefined is the explicit marker for a JavaScript undefined value.
type Undefined struct{}
