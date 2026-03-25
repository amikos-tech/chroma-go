package embeddings

// ContentOption configures optional fields on a Content value built by convenience constructors.
type ContentOption func(*Content)

// WithIntent sets the provider-neutral intent on the constructed Content.
func WithIntent(intent Intent) ContentOption {
	return func(c *Content) {
		c.Intent = intent
	}
}

// WithDimension sets the target output dimensionality on the constructed Content.
// Each call allocates a fresh pointer to avoid aliasing between Content values.
func WithDimension(dim int) ContentOption {
	return func(c *Content) {
		d := dim
		c.Dimension = &d
	}
}

// WithProviderHints sets provider-specific hints on the constructed Content.
func WithProviderHints(hints map[string]any) ContentOption {
	return func(c *Content) {
		c.ProviderHints = hints
	}
}

func applyContentOptions(c *Content, opts []ContentOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// NewTextContent creates a Content with a single text part.
func NewTextContent(text string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewTextPart(text)}}
	applyContentOptions(&c, opts)
	return c
}

// NewImageURL creates a Content with a single URL-backed image part.
func NewImageURL(url string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromURL(url))}}
	applyContentOptions(&c, opts)
	return c
}

// NewImageFile creates a Content with a single file-backed image part.
func NewImageFile(path string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromFile(path))}}
	applyContentOptions(&c, opts)
	return c
}

// NewVideoURL creates a Content with a single URL-backed video part.
func NewVideoURL(url string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromURL(url))}}
	applyContentOptions(&c, opts)
	return c
}

// NewVideoFile creates a Content with a single file-backed video part.
func NewVideoFile(path string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromFile(path))}}
	applyContentOptions(&c, opts)
	return c
}

// NewAudioFile creates a Content with a single file-backed audio part.
func NewAudioFile(path string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromFile(path))}}
	applyContentOptions(&c, opts)
	return c
}

// NewPDFFile creates a Content with a single file-backed PDF part.
func NewPDFFile(path string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewPartFromSource(ModalityPDF, NewBinarySourceFromFile(path))}}
	applyContentOptions(&c, opts)
	return c
}

// NewContent creates a Content from pre-built parts with optional configuration.
func NewContent(parts []Part, opts ...ContentOption) Content {
	c := Content{Parts: parts}
	applyContentOptions(&c, opts)
	return c
}
