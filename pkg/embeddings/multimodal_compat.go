package embeddings

// NewTextPart creates a text part for the shared multimodal contract.
func NewTextPart(text string) Part {
	return Part{
		Modality: ModalityText,
		Text:     text,
	}
}

// NewPartFromSource creates a non-text multimodal part from a binary source.
func NewPartFromSource(modality Modality, source BinarySource) Part {
	sourceCopy := source
	if len(source.Bytes) > 0 {
		sourceCopy.Bytes = append([]byte(nil), source.Bytes...)
	}
	return Part{
		Modality: modality,
		Source:   &sourceCopy,
	}
}

// NewBinarySourceFromURL creates a URL-backed binary source without dereferencing it.
func NewBinarySourceFromURL(url string) BinarySource {
	return BinarySource{
		Kind: SourceKindURL,
		URL:  url,
	}
}

// NewBinarySourceFromFile creates a file-backed binary source without opening it.
func NewBinarySourceFromFile(filePath string) BinarySource {
	return BinarySource{
		Kind:     SourceKindFile,
		FilePath: filePath,
	}
}

// NewBinarySourceFromBase64 creates a base64-backed binary source without decoding it.
func NewBinarySourceFromBase64(base64Data string) BinarySource {
	return BinarySource{
		Kind:   SourceKindBase64,
		Base64: base64Data,
	}
}

// NewBinarySourceFromBytes creates an in-memory binary source.
func NewBinarySourceFromBytes(data []byte) BinarySource {
	return BinarySource{
		Kind:  SourceKindBytes,
		Bytes: append([]byte(nil), data...),
	}
}

// NewImagePartFromImageInput bridges the legacy image-only input into the shared part model.
func NewImagePartFromImageInput(input ImageInput) (Part, error) {
	sourceType := input.Type()
	payloadCount := 0
	if input.Base64 != "" {
		payloadCount++
	}
	if input.URL != "" {
		payloadCount++
	}
	if input.FilePath != "" {
		payloadCount++
	}

	if payloadCount == 0 {
		return Part{}, &ValidationError{
			Issues: []ValidationIssue{{
				Path:    "input",
				Code:    validationCodeRequired,
				Message: "image input must set exactly one of Base64, URL, or FilePath",
			}},
		}
	}
	if payloadCount > 1 {
		return Part{}, &ValidationError{
			Issues: []ValidationIssue{{
				Path:    "input",
				Code:    validationCodeOneOf,
				Message: "image input must set exactly one of Base64, URL, or FilePath",
			}},
		}
	}

	var source BinarySource
	switch sourceType {
	case ImageInputTypeBase64:
		source = NewBinarySourceFromBase64(input.Base64)
	case ImageInputTypeURL:
		source = NewBinarySourceFromURL(input.URL)
	case ImageInputTypeFilePath:
		source = NewBinarySourceFromFile(input.FilePath)
	default:
		return Part{}, &ValidationError{
			Issues: []ValidationIssue{{
				Path:    "input",
				Code:    validationCodeInvalidValue,
				Message: "image input must use a supported source type",
			}},
		}
	}

	part := NewPartFromSource(ModalityImage, source)
	if err := part.Validate(); err != nil {
		return Part{}, err
	}

	return part, nil
}
