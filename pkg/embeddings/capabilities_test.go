package embeddings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type capabilityAwareStubEmbeddingFunction struct {
	caps CapabilityMetadata
}

func (s *capabilityAwareStubEmbeddingFunction) EmbedDocuments(_ context.Context, texts []string) ([]Embedding, error) {
	result := make([]Embedding, len(texts))
	for i := range texts {
		result[i] = NewEmbeddingFromFloat32([]float32{float32(i + 1)})
	}
	return result, nil
}

func (s *capabilityAwareStubEmbeddingFunction) EmbedQuery(_ context.Context, text string) (Embedding, error) {
	return NewEmbeddingFromFloat32([]float32{float32(len(text))}), nil
}

func (s *capabilityAwareStubEmbeddingFunction) Name() string {
	return "capability-aware-stub"
}

func (s *capabilityAwareStubEmbeddingFunction) GetConfig() EmbeddingFunctionConfig {
	return EmbeddingFunctionConfig{"provider": "stub"}
}

func (s *capabilityAwareStubEmbeddingFunction) DefaultSpace() DistanceMetric {
	return COSINE
}

func (s *capabilityAwareStubEmbeddingFunction) SupportedSpaces() []DistanceMetric {
	return []DistanceMetric{COSINE}
}

func (s *capabilityAwareStubEmbeddingFunction) Capabilities() CapabilityMetadata {
	return s.caps
}

type recordingEmbeddingFunction struct {
	queries       []string
	documentBatch [][]string
}

func (r *recordingEmbeddingFunction) EmbedDocuments(_ context.Context, texts []string) ([]Embedding, error) {
	r.documentBatch = append(r.documentBatch, append([]string(nil), texts...))

	result := make([]Embedding, len(texts))
	for i := range texts {
		result[i] = NewEmbeddingFromFloat32([]float32{float32(i + 1)})
	}
	return result, nil
}

func (r *recordingEmbeddingFunction) EmbedQuery(_ context.Context, text string) (Embedding, error) {
	r.queries = append(r.queries, text)
	return NewEmbeddingFromFloat32([]float32{float32(len(text))}), nil
}

func (r *recordingEmbeddingFunction) Name() string {
	return "recording-text"
}

func (r *recordingEmbeddingFunction) GetConfig() EmbeddingFunctionConfig {
	return EmbeddingFunctionConfig{"provider": "recording-text"}
}

func (r *recordingEmbeddingFunction) DefaultSpace() DistanceMetric {
	return COSINE
}

func (r *recordingEmbeddingFunction) SupportedSpaces() []DistanceMetric {
	return []DistanceMetric{COSINE}
}

type recordingMultimodalEmbeddingFunction struct {
	recordingEmbeddingFunction
	images       []ImageInput
	imageBatches [][]ImageInput
}

func (r *recordingMultimodalEmbeddingFunction) EmbedImages(_ context.Context, images []ImageInput) ([]Embedding, error) {
	r.imageBatches = append(r.imageBatches, cloneImageInputs(images))

	result := make([]Embedding, len(images))
	for i := range images {
		result[i] = NewEmbeddingFromFloat32([]float32{float32(i + 10)})
	}
	return result, nil
}

func (r *recordingMultimodalEmbeddingFunction) EmbedImage(_ context.Context, image ImageInput) (Embedding, error) {
	r.images = append(r.images, image)
	return NewEmbeddingFromFloat32([]float32{42}), nil
}

func TestCapabilityMetadata(t *testing.T) {
	provider := &capabilityAwareStubEmbeddingFunction{
		caps: CapabilityMetadata{
			Modalities: []Modality{ModalityText, ModalityImage},
			Intents: []Intent{
				IntentRetrievalQuery,
				IntentSemanticSimilarity,
			},
			RequestOptions: []RequestOption{
				RequestOptionDimension,
				RequestOptionProviderHints,
			},
			SupportsBatch:     true,
			SupportsMixedPart: false,
		},
	}

	var aware CapabilityAware = provider
	caps := aware.Capabilities()

	require.True(t, caps.SupportsModality(ModalityText))
	require.True(t, caps.SupportsModality(ModalityImage))
	require.False(t, caps.SupportsModality(ModalityAudio))
	require.True(t, caps.SupportsIntent(IntentRetrievalQuery))
	require.True(t, caps.SupportsIntent(IntentSemanticSimilarity))
	require.False(t, caps.SupportsIntent(IntentClassification))
	require.True(t, caps.SupportsRequestOption(RequestOptionDimension))
	require.True(t, caps.SupportsRequestOption(RequestOptionProviderHints))
	require.True(t, caps.SupportsBatch)
	require.False(t, caps.SupportsMixedPart)
}

func TestLegacyTextCompatibility(t *testing.T) {
	legacy := &recordingEmbeddingFunction{}
	adapter := AdaptEmbeddingFunctionToContent(legacy, CapabilityMetadata{
		Modalities:    []Modality{ModalityText},
		SupportsBatch: true,
	})

	embedding, err := adapter.EmbedContent(context.Background(), Content{
		Parts: []Part{NewTextPart("query text")},
	})
	require.NoError(t, err)
	require.NotNil(t, embedding)
	require.Empty(t, legacy.queries)
	require.Len(t, legacy.documentBatch, 1)
	require.Equal(t, []string{"query text"}, legacy.documentBatch[0])

	embeddings, err := adapter.EmbedContents(context.Background(), []Content{
		{Parts: []Part{NewTextPart("first document")}},
		{Parts: []Part{NewTextPart("second document")}},
	})
	require.NoError(t, err)
	require.Len(t, embeddings, 2)
	require.Len(t, legacy.documentBatch, 2)
	require.Equal(t, []string{"first document", "second document"}, legacy.documentBatch[1])
}

func TestLegacyImageCompatibility(t *testing.T) {
	legacy := &recordingMultimodalEmbeddingFunction{}
	adapter := AdaptMultimodalEmbeddingFunctionToContent(legacy, CapabilityMetadata{
		Modalities:    []Modality{ModalityText, ModalityImage},
		SupportsBatch: true,
	})

	textEmbedding, err := adapter.EmbedContent(context.Background(), Content{
		Parts: []Part{NewTextPart("query")},
	})
	require.NoError(t, err)
	require.NotNil(t, textEmbedding)
	require.Empty(t, legacy.queries)
	require.Len(t, legacy.documentBatch, 1)
	require.Equal(t, []string{"query"}, legacy.documentBatch[0])

	imageURL := "https://example.com/image.png"
	imageEmbedding, err := adapter.EmbedContent(context.Background(), Content{
		Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromURL(imageURL))},
	})
	require.NoError(t, err)
	require.NotNil(t, imageEmbedding)
	require.Len(t, legacy.images, 1)
	require.Equal(t, imageURL, legacy.images[0].URL)

	textBatch, err := adapter.EmbedContents(context.Background(), []Content{
		{Parts: []Part{NewTextPart("first")}},
		{Parts: []Part{NewTextPart("second")}},
	})
	require.NoError(t, err)
	require.Len(t, textBatch, 2)
	require.Len(t, legacy.documentBatch, 2)
	require.Equal(t, []string{"first", "second"}, legacy.documentBatch[1])

	imageBatch, err := adapter.EmbedContents(context.Background(), []Content{
		{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromBase64("aW1hZ2Ux"))}},
		{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromFile("/tmp/image-2.png"))}},
	})
	require.NoError(t, err)
	require.Len(t, imageBatch, 2)
	require.Len(t, legacy.imageBatches, 1)
	require.Equal(t, "aW1hZ2Ux", legacy.imageBatches[0][0].Base64)
	require.Equal(t, "/tmp/image-2.png", legacy.imageBatches[0][1].FilePath)
}

func TestCompatibilityAdapterRejectsUnsupportedContent(t *testing.T) {
	textAdapter := AdaptEmbeddingFunctionToContent(&recordingEmbeddingFunction{}, CapabilityMetadata{
		Modalities:    []Modality{ModalityText},
		SupportsBatch: true,
	})
	multimodalAdapter := AdaptMultimodalEmbeddingFunctionToContent(&recordingMultimodalEmbeddingFunction{}, CapabilityMetadata{
		Modalities:    []Modality{ModalityText, ModalityImage},
		SupportsBatch: true,
	})

	dimension := 128
	cases := []struct {
		name        string
		adapter     ContentEmbeddingFunction
		content     Content
		path        string
		code        string
		messagePart string
	}{
		{
			name:        "mixed parts",
			adapter:     multimodalAdapter,
			content:     Content{Parts: []Part{NewTextPart("first"), NewTextPart("second")}},
			path:        "parts",
			code:        validationCodeOneOf,
			messagePart: "exactly one Part",
		},
		{
			name:        "audio modality",
			adapter:     multimodalAdapter,
			content:     Content{Parts: []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromURL("https://example.com/audio.wav"))}},
			path:        "parts[0].modality",
			code:        validationCodeInvalidValue,
			messagePart: "audio",
		},
		{
			name:        "video modality",
			adapter:     multimodalAdapter,
			content:     Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromURL("https://example.com/video.mp4"))}},
			path:        "parts[0].modality",
			code:        validationCodeInvalidValue,
			messagePart: "video",
		},
		{
			name:        "pdf modality",
			adapter:     multimodalAdapter,
			content:     Content{Parts: []Part{NewPartFromSource(ModalityPDF, NewBinarySourceFromBase64("JVBERg=="))}},
			path:        "parts[0].modality",
			code:        validationCodeInvalidValue,
			messagePart: "pdf",
		},
		{
			name:        "intent request option",
			adapter:     textAdapter,
			content:     Content{Parts: []Part{NewTextPart("query")}, Intent: IntentRetrievalQuery},
			path:        "intent",
			code:        validationCodeForbidden,
			messagePart: "Intent",
		},
		{
			name:        "dimension request option",
			adapter:     textAdapter,
			content:     Content{Parts: []Part{NewTextPart("query")}, Dimension: &dimension},
			path:        "dimension",
			code:        validationCodeForbidden,
			messagePart: "Dimension",
		},
		{
			name:        "provider hints request option",
			adapter:     textAdapter,
			content:     Content{Parts: []Part{NewTextPart("query")}, ProviderHints: map[string]any{"provider": "roboflow"}},
			path:        "providerHints",
			code:        validationCodeForbidden,
			messagePart: "ProviderHints",
		},
		{
			name:        "bytes-backed image source",
			adapter:     multimodalAdapter,
			content:     Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromBytes([]byte("img")))}},
			path:        "parts[0].source.kind",
			code:        validationCodeForbidden,
			messagePart: "bytes-backed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.adapter.EmbedContent(context.Background(), tc.content)
			requireValidationIssue(t, err, tc.path, tc.code, tc.messagePart)
		})
	}
}

func cloneImageInputs(images []ImageInput) []ImageInput {
	cloned := make([]ImageInput, len(images))
	copy(cloned, images)
	return cloned
}

func requireValidationIssue(t *testing.T, err error, path, code, messagePart string) {
	t.Helper()

	require.Error(t, err)

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.NotEmpty(t, validationErr.Issues)
	require.Equal(t, path, validationErr.Issues[0].Path)
	require.Equal(t, code, validationErr.Issues[0].Code)
	require.Contains(t, validationErr.Issues[0].Message, messagePart)
}
