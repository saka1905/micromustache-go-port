package micromustache

// Tags defines the opening and closing template delimiters. Empty fields mean
// unspecified; the upstream defaults are "{{" and "}}".
type Tags struct {
	Open  string
	Close string
}

// GetOptions controls reference traversal. MaxRefDepth zero selects the
// upstream default of 10 for Get and GetRef.
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

// TokenizeOptions controls template tokenization. MaxPathLen zero selects the
// upstream default of 1000. Empty Tags select the upstream default delimiters.
type TokenizeOptions struct {
	MaxPathLen int
	Tags       Tags
}

// CompileOptions combines the options accepted by tokenization and rendering.
type CompileOptions struct {
	RendererOptions
	TokenizeOptions
}
