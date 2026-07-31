package micromustache

// Tags defines the opening and closing template delimiters. Empty fields mean
// unspecified; the upstream defaults are "{{" and "}}".
type Tags struct {
	Open  string
	Close string
}

// GetOptions controls reference traversal. MaxRefDepth zero means unspecified;
// the upstream default is 10. Phase 3A records but does not apply defaults.
type GetOptions struct {
	ValidateRef bool
	MaxRefDepth int
}

// RendererOptions controls rendering and embeds reference traversal options.
// Explicit and ValidatePath default to false upstream.
type RendererOptions struct {
	GetOptions
	Explicit     bool
	ValidatePath bool
}

// TokenizeOptions controls template tokenization. MaxPathLen zero means
// unspecified; the upstream default is 1000. Empty Tags are also unspecified.
// Phase 3A records but does not apply defaults.
type TokenizeOptions struct {
	MaxPathLen int
	Tags       Tags
}

// CompileOptions combines the options accepted by tokenization and rendering.
type CompileOptions struct {
	RendererOptions
	TokenizeOptions
}
