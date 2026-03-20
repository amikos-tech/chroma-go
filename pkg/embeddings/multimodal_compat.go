package embeddings

import (
	"context"
	"fmt"
)

var (
	_ ContentEmbeddingFunction = (*embeddingFunctionContentAdapter)(nil)
	_ ContentEmbeddingFunction = (*multimodalEmbeddingFunctionContentAdapter)(nil)
	_ CapabilityAware          = (*embeddingFunctionContentAdapter)(nil)
	_ CapabilityAware          = (*multimodalEmbeddingFunctionContentAdapter)(nil)
	_ Closeable                = (*embeddingFunctionContentAdapter)(nil)
	_ Closeable                = (*multimodalEmbeddingFunctionContentAdapter)(nil)
)

type embeddingFunctionContentAdapter struct {
	ef   EmbeddingFunction
	caps CapabilityMetadata
}

type multimodalEmbeddingFunctionContentAdapter struct {
	ef   MultimodalEmbeddingFunction
	caps CapabilityMetadata
}

type compatibleContent struct {
	modality Modality
	text     string
	image    ImageInput
}

// AdaptEmbeddingFunctionToContent bridges a legacy text-only embedding function into the shared content interface.
func AdaptEmbeddingFunctionToContent(ef EmbeddingFunction, caps CapabilityMetadata) ContentEmbeddingFunction {
	if ef == nil {
		return nil
	}
	return &embeddingFunctionContentAdapter{ef: ef, caps: caps}
}

// AdaptMultimodalEmbeddingFunctionToContent bridges a legacy text/image embedding function into the shared content interface.
func AdaptMultimodalEmbeddingFunctionToContent(ef MultimodalEmbeddingFunction, caps CapabilityMetadata) ContentEmbeddingFunction {
	if ef == nil {
		return nil
	}
	return &multimodalEmbeddingFunctionContentAdapter{ef: ef, caps: caps}
}

func (a *embeddingFunctionContentAdapter) Capabilities() CapabilityMetadata {
	return a.caps
}

func (a *embeddingFunctionContentAdapter) Close() error {
	if c, ok := a.ef.(Closeable); ok {
		return c.Close()
	}
	return nil
}

func (a *embeddingFunctionContentAdapter) EmbedContent(ctx context.Context, content Content) (Embedding, error) {
	compatible, err := textContentFromSharedContent(content, a.caps)
	if err != nil {
		return nil, err
	}
	embeddings, err := a.ef.EmbedDocuments(ctx, []string{compatible.text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("EmbedDocuments returned empty result for single text input")
	}
	return embeddings[0], nil
}

func (a *embeddingFunctionContentAdapter) EmbedContents(ctx context.Context, contents []Content) ([]Embedding, error) {
	if err := validateBatchCompatibility(contents, a.caps); err != nil {
		return nil, err
	}

	texts := make([]string, len(contents))
	for i, content := range contents {
		compatible, err := textContentFromSharedContent(content, a.caps)
		if err != nil {
			return nil, prefixBatchCompatibilityError(i, err)
		}
		texts[i] = compatible.text
	}

	return a.ef.EmbedDocuments(ctx, texts)
}

func (a *multimodalEmbeddingFunctionContentAdapter) Capabilities() CapabilityMetadata {
	return a.caps
}

func (a *multimodalEmbeddingFunctionContentAdapter) Close() error {
	if c, ok := a.ef.(Closeable); ok {
		return c.Close()
	}
	return nil
}

func (a *multimodalEmbeddingFunctionContentAdapter) EmbedContent(ctx context.Context, content Content) (Embedding, error) {
	compatible, err := multimodalContentFromSharedContent(content, a.caps)
	if err != nil {
		return nil, err
	}

	switch compatible.modality {
	case ModalityText:
		embeddings, qErr := a.ef.EmbedDocuments(ctx, []string{compatible.text})
		if qErr != nil {
			return nil, qErr
		}
		if len(embeddings) == 0 {
			return nil, fmt.Errorf("EmbedDocuments returned empty result for single text input")
		}
		return embeddings[0], nil
	case ModalityImage:
		return a.ef.EmbedImage(ctx, compatible.image)
	default:
		return nil, compatibilityError("parts[0].modality", validationCodeInvalidValue, fmt.Sprintf("legacy multimodal adapter does not support %q modality", compatible.modality))
	}
}

func (a *multimodalEmbeddingFunctionContentAdapter) EmbedContents(ctx context.Context, contents []Content) ([]Embedding, error) {
	if err := validateBatchCompatibility(contents, a.caps); err != nil {
		return nil, err
	}

	compatible := make([]compatibleContent, len(contents))
	allText := true
	allImage := true
	for i, content := range contents {
		item, err := multimodalContentFromSharedContent(content, a.caps)
		if err != nil {
			return nil, prefixBatchCompatibilityError(i, err)
		}
		compatible[i] = item
		allText = allText && item.modality == ModalityText
		allImage = allImage && item.modality == ModalityImage
	}

	if allText {
		texts := make([]string, len(compatible))
		for i, item := range compatible {
			texts[i] = item.text
		}
		return a.ef.EmbedDocuments(ctx, texts)
	}

	if allImage {
		images := make([]ImageInput, len(compatible))
		for i, item := range compatible {
			images[i] = item.image
		}
		return a.ef.EmbedImages(ctx, images)
	}

	result := make([]Embedding, len(compatible))
	for i, item := range compatible {
		switch item.modality {
		case ModalityText:
			embeddings, qErr := a.ef.EmbedDocuments(ctx, []string{item.text})
			if qErr != nil {
				return nil, prefixBatchCompatibilityError(i, qErr)
			}
			if len(embeddings) == 0 {
				return nil, prefixBatchCompatibilityError(i, fmt.Errorf("EmbedDocuments returned empty result for single text input"))
			}
			result[i] = embeddings[0]
		case ModalityImage:
			embedding, qErr := a.ef.EmbedImage(ctx, item.image)
			if qErr != nil {
				return nil, prefixBatchCompatibilityError(i, qErr)
			}
			result[i] = embedding
		default:
			return nil, compatibilityError("contents", validationCodeInvalidValue, fmt.Sprintf("legacy multimodal adapter does not support %q modality", item.modality))
		}
	}

	return result, nil
}

func validateBatchCompatibility(contents []Content, caps CapabilityMetadata) error {
	if err := ValidateContents(contents); err != nil {
		return err
	}
	if len(contents) > 1 && !caps.SupportsBatch {
		return compatibilityError("contents", validationCodeForbidden, "provider capabilities do not support batched shared content requests")
	}
	return nil
}

func textContentFromSharedContent(content Content, caps CapabilityMetadata) (compatibleContent, error) {
	part, err := validateCompatibleContent(content, caps, ModalityText)
	if err != nil {
		return compatibleContent{}, err
	}

	return compatibleContent{
		modality: ModalityText,
		text:     part.Text,
	}, nil
}

func multimodalContentFromSharedContent(content Content, caps CapabilityMetadata) (compatibleContent, error) {
	part, err := validateCompatibleContent(content, caps, ModalityText, ModalityImage)
	if err != nil {
		return compatibleContent{}, err
	}

	switch part.Modality {
	case ModalityText:
		return compatibleContent{
			modality: ModalityText,
			text:     part.Text,
		}, nil
	case ModalityImage:
		image, err := imageInputFromPart(part)
		if err != nil {
			return compatibleContent{}, err
		}
		return compatibleContent{
			modality: ModalityImage,
			image:    image,
		}, nil
	default:
		return compatibleContent{}, compatibilityError("parts[0].modality", validationCodeInvalidValue, fmt.Sprintf("legacy multimodal adapter does not support %q modality", part.Modality))
	}
}

func validateCompatibleContent(content Content, caps CapabilityMetadata, allowedModalities ...Modality) (Part, error) {
	if err := content.Validate(); err != nil {
		return Part{}, err
	}

	compatErr := &ValidationError{}
	if len(content.Parts) != 1 {
		compatErr.addIssue("parts", validationCodeOneOf, "legacy compatibility adapters require exactly one Part")
	}
	if content.Intent != "" {
		compatErr.addIssue("intent", validationCodeForbidden, "legacy compatibility adapters do not support Intent")
	}
	if content.Dimension != nil {
		compatErr.addIssue("dimension", validationCodeForbidden, "legacy compatibility adapters do not support Dimension")
	}
	if len(content.ProviderHints) > 0 {
		compatErr.addIssue("providerHints", validationCodeForbidden, "legacy compatibility adapters do not support ProviderHints")
	}
	if err := compatErr.orNil(); err != nil {
		return Part{}, err
	}

	part := content.Parts[0]
	if !isAllowedCompatibilityModality(part.Modality, allowedModalities) {
		return Part{}, compatibilityError("parts[0].modality", validationCodeInvalidValue, fmt.Sprintf("legacy compatibility adapters do not support %q modality", part.Modality))
	}
	if len(caps.Modalities) > 0 && !caps.SupportsModality(part.Modality) {
		return Part{}, compatibilityError("parts[0].modality", validationCodeForbidden, fmt.Sprintf("provider capabilities do not advertise %q modality support", part.Modality))
	}
	return part, nil
}

func isAllowedCompatibilityModality(modality Modality, allowedModalities []Modality) bool {
	for _, allowed := range allowedModalities {
		if modality == allowed {
			return true
		}
	}
	return false
}

func imageInputFromPart(part Part) (ImageInput, error) {
	if part.Source == nil {
		return ImageInput{}, compatibilityError("parts[0].source", validationCodeRequired, "image parts must include a binary source")
	}

	switch part.Source.Kind {
	case SourceKindBase64:
		return NewImageInputFromBase64(part.Source.Base64), nil
	case SourceKindURL:
		return NewImageInputFromURL(part.Source.URL), nil
	case SourceKindFile:
		return NewImageInputFromFile(part.Source.FilePath), nil
	case SourceKindBytes:
		return ImageInput{}, compatibilityError("parts[0].source.kind", validationCodeForbidden, "legacy image adapters do not support bytes-backed binary sources")
	default:
		return ImageInput{}, compatibilityError("parts[0].source.kind", validationCodeInvalidValue, fmt.Sprintf("legacy image adapters do not support %q binary sources", part.Source.Kind))
	}
}

func compatibilityError(path, code, message string) error {
	return &ValidationError{
		Issues: []ValidationIssue{{
			Path:    path,
			Code:    code,
			Message: message,
		}},
	}
}

func prefixBatchCompatibilityError(index int, err error) error {
	return &ValidationError{
		Issues: prefixValidationIssues(fmt.Sprintf("contents[%d].", index), err),
	}
}

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
