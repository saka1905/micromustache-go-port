package micromustache

// Tokens is the public token representation exposed by the fixed upstream API.
// The upstream type exposes only string fragments and paths; it exposes no
// source offsets, raw token text, or parsed references.
type Tokens struct {
	Strings []string
	Paths   []string
}
