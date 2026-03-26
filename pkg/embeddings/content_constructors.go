package embeddings

import "maps"

// ContentOption configures optional fields on a Content value built by convenience constructors.
type ContentOption func(*Content)

// WithIntent sets the provider-neutral intent on the constructed Content.
func WithIntent(intent Intent) ContentOption {
	return func(c *Content) {
		c.Intent = intent
	}
}

// WithDimension sets the target output dimensionality on the constructed Content.
func WithDimension(dim int) ContentOption {
	return func(c *Content) {
		d := dim
		c.Dimension = &d
	}
}

// WithProviderHints sets provider-specific hints on the constructed Content.
func WithProviderHints(hints map[string]any) ContentOption {
	return func(c *Content) {
		c.ProviderHints = maps.Clone(hints)
	}
}

func applyContentOptions(c *Content, opts []ContentOption) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
}

// TODO(#469): URL constructors produce BinarySource without MIMEType, which causes
// resolveMIME to fail in Gemini and VoyageAI providers. Fix by adding URL extension
// inference to resolveMIME.
func newBinaryContent(modality Modality, source BinarySource, opts []ContentOption) Content {
	c := Content{Parts: []Part{NewPartFromSource(modality, source)}}
	applyContentOptions(&c, opts)
	return c
}

// NewTextContent creates a Content with a single text part.
func NewTextContent(text string, opts ...ContentOption) Content {
	c := Content{Parts: []Part{NewTextPart(text)}}
	applyContentOptions(&c, opts)
	return c
}

// NewImageURL creates a Content with a single URL-backed image part.
func NewImageURL(url string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityImage, NewBinarySourceFromURL(url), opts)
}

// NewImageFile creates a Content with a single file-backed image part.
func NewImageFile(path string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityImage, NewBinarySourceFromFile(path), opts)
}

// NewVideoURL creates a Content with a single URL-backed video part.
func NewVideoURL(url string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityVideo, NewBinarySourceFromURL(url), opts)
}

// NewVideoFile creates a Content with a single file-backed video part.
func NewVideoFile(path string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityVideo, NewBinarySourceFromFile(path), opts)
}

// NewAudioURL creates a Content with a single URL-backed audio part.
func NewAudioURL(url string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityAudio, NewBinarySourceFromURL(url), opts)
}

// NewAudioFile creates a Content with a single file-backed audio part.
func NewAudioFile(path string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityAudio, NewBinarySourceFromFile(path), opts)
}

// NewPDFURL creates a Content with a single URL-backed PDF part.
func NewPDFURL(url string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityPDF, NewBinarySourceFromURL(url), opts)
}

// NewPDFFile creates a Content with a single file-backed PDF part.
func NewPDFFile(path string, opts ...ContentOption) Content {
	return newBinaryContent(ModalityPDF, NewBinarySourceFromFile(path), opts)
}

// NewContent creates a Content from pre-built parts with optional configuration.
func NewContent(parts []Part, opts ...ContentOption) Content {
	var partsCopy []Part
	if parts != nil {
		partsCopy = make([]Part, len(parts))
		copy(partsCopy, parts)
	}
	c := Content{Parts: partsCopy}
	applyContentOptions(&c, opts)
	return c
}
