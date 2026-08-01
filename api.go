package micromustache

// Compile tokenizes a template and returns a reusable Renderer.
func Compile(template string, options CompileOptions) (*Renderer, error) {
	tokens, err := Tokenize(template, options.TokenizeOptions)
	if err != nil {
		return nil, err
	}
	return NewRenderer(tokens, options.RendererOptions)
}
