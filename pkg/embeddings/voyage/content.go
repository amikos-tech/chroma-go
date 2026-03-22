package voyage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	defaultMultimodalModel = "voyage-multimodal-3.5"
	multimodalBaseAPI      = "https://api.voyageai.com/v1/multimodalembeddings"
	maxMultimodalFileSize  = 100 * 1024 * 1024 // 100 MB
)

// MultimodalContentBlock represents a single content block in a Voyage multimodal request.
// The Type field selects which payload field is active:
// "text", "image_base64", "image_url", "video_base64", "video_url".
type MultimodalContentBlock struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	VideoBase64 string `json:"video_base64,omitempty"`
	VideoURL    string `json:"video_url,omitempty"`
}

// MultimodalInput represents one input item containing multiple content blocks.
type MultimodalInput struct {
	Content []MultimodalContentBlock `json:"content"`
}

// CreateMultimodalEmbeddingRequest is the request body for Voyage's /v1/multimodalembeddings endpoint.
type CreateMultimodalEmbeddingRequest struct {
	Model           string            `json:"model"`
	Inputs          []MultimodalInput `json:"inputs"`
	InputType       *InputType        `json:"input_type"`
	Truncation      *bool             `json:"truncation"`
	OutputDimension *int              `json:"output_dimension,omitempty"`
}

// extToMIME maps file extensions to MIME types for Voyage-supported formats.
var extToMIME = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".mp4":  "video/mp4",
}

// capabilitiesForModel returns the CapabilityMetadata for the given Voyage model.
func capabilitiesForModel(model string) embeddings.CapabilityMetadata {
	switch model {
	case defaultMultimodalModel:
		return embeddings.CapabilityMetadata{
			Modalities: []embeddings.Modality{
				embeddings.ModalityText,
				embeddings.ModalityImage,
				embeddings.ModalityVideo,
			},
			Intents: []embeddings.Intent{
				embeddings.IntentRetrievalQuery,
				embeddings.IntentRetrievalDocument,
			},
			RequestOptions: []embeddings.RequestOption{
				embeddings.RequestOptionDimension,
			},
			SupportsBatch:     true,
			SupportsMixedPart: true,
		}
	case "voyage-multimodal-3":
		return embeddings.CapabilityMetadata{
			Modalities: []embeddings.Modality{
				embeddings.ModalityText,
				embeddings.ModalityImage,
			},
			Intents: []embeddings.Intent{
				embeddings.IntentRetrievalQuery,
				embeddings.IntentRetrievalDocument,
			},
			SupportsBatch:     true,
			SupportsMixedPart: true,
		}
	default:
		return embeddings.CapabilityMetadata{
			Modalities:    []embeddings.Modality{embeddings.ModalityText},
			SupportsBatch: true,
		}
	}
}

// resolveBytes fetches or reads the raw bytes for a binary source.
func resolveBytes(_ context.Context, source *embeddings.BinarySource, maxFileSize int64) ([]byte, error) {
	if source == nil {
		return nil, errors.New("source cannot be nil")
	}
	switch source.Kind {
	case embeddings.SourceKindBytes:
		if int64(len(source.Bytes)) > maxFileSize {
			return nil, errors.Errorf("bytes payload size %d exceeds maximum of %d bytes", len(source.Bytes), maxFileSize)
		}
		return source.Bytes, nil
	case embeddings.SourceKindBase64:
		if int64(len(source.Base64))*3/4 > maxFileSize {
			return nil, errors.Errorf("base64 payload too large: estimated decoded size exceeds maximum of %d bytes", maxFileSize)
		}
		data, err := base64.StdEncoding.DecodeString(source.Base64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base64 source")
		}
		return data, nil
	case embeddings.SourceKindFile:
		cleaned := filepath.Clean(source.FilePath)
		if containsDotDot(cleaned) {
			return nil, errors.Errorf("file path %q contains path traversal", source.FilePath)
		}
		f, err := os.Open(cleaned)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open file source")
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file source")
		}
		if int64(len(data)) > maxFileSize {
			return nil, errors.Errorf("file size exceeds maximum of %d bytes", maxFileSize)
		}
		return data, nil
	case embeddings.SourceKindURL:
		return nil, errors.New("URL sources are handled via direct passthrough, not byte resolution")
	default:
		return nil, errors.Errorf("unsupported source kind %q", source.Kind)
	}
}

// containsDotDot reports whether the cleaned path still contains ".." components.
func containsDotDot(path string) bool {
	return slices.Contains(strings.Split(filepath.ToSlash(path), "/"), "..")
}

// resolveMIME determines the MIME type for a binary source.
func resolveMIME(source *embeddings.BinarySource) (string, error) {
	if source == nil {
		return "", errors.New("source cannot be nil")
	}
	if source.MIMEType != "" {
		return source.MIMEType, nil
	}
	if source.FilePath != "" {
		ext := strings.ToLower(filepath.Ext(source.FilePath))
		if mime, ok := extToMIME[ext]; ok {
			return mime, nil
		}
	}
	return "", errors.New("MIME type is required: set BinarySource.MIMEType or use a file with a known extension")
}

// validateMIMEModality ensures the MIME type is consistent with the declared modality.
func validateMIMEModality(modality embeddings.Modality, mimeType string) error {
	switch modality {
	case embeddings.ModalityImage:
		if !strings.HasPrefix(mimeType, "image/") {
			return errors.Errorf("image modality requires image/* MIME type, got %q", mimeType)
		}
	case embeddings.ModalityVideo:
		if !strings.HasPrefix(mimeType, "video/") {
			return errors.Errorf("video modality requires video/* MIME type, got %q", mimeType)
		}
	}
	return nil
}

// convertToVoyageInput converts a shared Content item to a Voyage MultimodalInput.
func convertToVoyageInput(ctx context.Context, content embeddings.Content, maxFileSize int64) (*MultimodalInput, error) {
	blocks := make([]MultimodalContentBlock, 0, len(content.Parts))
	for i, part := range content.Parts {
		block, err := buildContentBlock(ctx, part, maxFileSize)
		if err != nil {
			return nil, errors.Wrapf(err, "part[%d]", i)
		}
		blocks = append(blocks, block)
	}
	return &MultimodalInput{Content: blocks}, nil
}

// buildContentBlock converts a single Part to a Voyage MultimodalContentBlock.
func buildContentBlock(ctx context.Context, part embeddings.Part, maxFileSize int64) (MultimodalContentBlock, error) {
	switch part.Modality {
	case embeddings.ModalityText:
		return MultimodalContentBlock{Type: "text", Text: part.Text}, nil
	case embeddings.ModalityImage:
		return buildBinaryBlock(ctx, part.Source, "image", maxFileSize)
	case embeddings.ModalityVideo:
		return buildBinaryBlock(ctx, part.Source, "video", maxFileSize)
	default:
		return MultimodalContentBlock{}, errors.Errorf("unsupported modality %q", part.Modality)
	}
}

// buildBinaryBlock converts a binary source to a Voyage content block.
// For URL sources, it passes through as image_url/video_url.
// For all other kinds, it resolves bytes and encodes as a data URI.
func buildBinaryBlock(ctx context.Context, source *embeddings.BinarySource, mediaType string, maxFileSize int64) (MultimodalContentBlock, error) {
	if source == nil {
		return MultimodalContentBlock{}, errors.New("binary source is required for non-text parts")
	}
	if source.Kind == embeddings.SourceKindURL {
		block := MultimodalContentBlock{Type: mediaType + "_url"}
		switch mediaType {
		case "image":
			block.ImageURL = source.URL
		case "video":
			block.VideoURL = source.URL
		}
		return block, nil
	}

	mimeType, err := resolveMIME(source)
	if err != nil {
		return MultimodalContentBlock{}, err
	}
	if err := validateMIMEModality(embeddings.Modality(mediaType), mimeType); err != nil {
		return MultimodalContentBlock{}, err
	}
	data, err := resolveBytes(ctx, source, maxFileSize)
	if err != nil {
		return MultimodalContentBlock{}, err
	}

	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
	block := MultimodalContentBlock{Type: mediaType + "_base64"}
	switch mediaType {
	case "image":
		block.ImageBase64 = dataURI
	case "video":
		block.VideoBase64 = dataURI
	}
	return block, nil
}

// MapIntent translates a neutral shared intent to a Voyage input_type string.
// Only retrieval_query and retrieval_document are supported.
func (e *VoyageAIEmbeddingFunction) MapIntent(intent embeddings.Intent) (string, error) {
	if !embeddings.IsNeutralIntent(intent) {
		return "", errors.Errorf("unsupported intent %q: use ProviderHints[\"input_type\"] for Voyage-native input types", intent)
	}
	switch intent {
	case embeddings.IntentRetrievalQuery:
		return string(InputTypeQuery), nil
	case embeddings.IntentRetrievalDocument:
		return string(InputTypeDocument), nil
	default:
		return "", errors.Errorf("intent %q is not supported by Voyage; only retrieval_query and retrieval_document are available", intent)
	}
}

// Capabilities returns the capability metadata for the configured Voyage model.
func (e *VoyageAIEmbeddingFunction) Capabilities() embeddings.CapabilityMetadata {
	return capabilitiesForModel(string(e.apiClient.DefaultModel))
}

// resolveInputTypeForContent determines the effective Voyage input_type for a content request.
// Priority: ProviderHints["input_type"] > intent via mapper > defaultInputType.
func resolveInputTypeForContent(content embeddings.Content, defaultInputType *InputType, mapper embeddings.IntentMapper) (*InputType, error) {
	if hints := content.ProviderHints; hints != nil {
		if raw, ok := hints["input_type"]; ok {
			hintStr, ok := raw.(string)
			if !ok || hintStr == "" {
				return nil, errors.New("ProviderHints[\"input_type\"] must be a non-empty string")
			}
			it := InputType(hintStr)
			return &it, nil
		}
	}
	if content.Intent != "" && mapper != nil {
		mapped, err := mapper.MapIntent(content.Intent)
		if err != nil {
			return nil, errors.Wrap(err, "failed to map intent to Voyage input_type")
		}
		it := InputType(mapped)
		return &it, nil
	}
	return defaultInputType, nil
}

// multimodalURL derives the multimodal endpoint URL from the client's BaseAPI.
// If BaseAPI ends with /v1/embeddings, replaces the path; otherwise uses the default.
func multimodalURL(baseAPI string) string {
	if base, ok := strings.CutSuffix(baseAPI, "/v1/embeddings"); ok {
		return base + "/v1/multimodalembeddings"
	}
	return multimodalBaseAPI
}

// EmbedContent embeds a single multimodal content item using the shared Content API.
func (e *VoyageAIEmbeddingFunction) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	if err := content.Validate(); err != nil {
		return nil, err
	}
	caps := e.Capabilities()
	if err := embeddings.ValidateContentSupport(content, caps); err != nil {
		return nil, err
	}

	inputType, err := resolveInputTypeForContent(content, nil, e)
	if err != nil {
		return nil, err
	}

	var dimension *int
	if content.Dimension != nil {
		dimension = content.Dimension
	}

	input, err := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert content to Voyage format")
	}

	req := &CreateMultimodalEmbeddingRequest{
		Model:           string(e.getModel(ctx)),
		Inputs:          []MultimodalInput{*input},
		InputType:       inputType,
		Truncation:      e.getTruncation(ctx),
		OutputDimension: dimension,
	}

	resp, err := e.apiClient.CreateMultimodalEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding returned from Voyage multimodal API")
	}
	if resp.Data[0].Embedding == nil {
		return nil, errors.New("nil embedding in Voyage multimodal API response")
	}
	return embeddings.NewEmbeddingFromFloat32(resp.Data[0].Embedding.Floats), nil
}

// EmbedContents embeds a batch of multimodal content items using the shared Content API.
func (e *VoyageAIEmbeddingFunction) EmbedContents(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
	if err := embeddings.ValidateContents(contents); err != nil {
		return nil, err
	}
	if len(contents) > e.apiClient.MaxBatchSize {
		return nil, errors.Errorf("number of contents exceeds the maximum batch size %v", e.apiClient.MaxBatchSize)
	}
	caps := e.Capabilities()
	if err := embeddings.ValidateContentsSupport(contents, caps); err != nil {
		return nil, err
	}

	var inputType *InputType
	var dimension *int
	var err error

	if len(contents) == 1 {
		inputType, err = resolveInputTypeForContent(contents[0], nil, e)
		if err != nil {
			return nil, err
		}
		if contents[0].Dimension != nil {
			dimension = contents[0].Dimension
		}
	} else {
		for i, content := range contents {
			if content.Intent != "" {
				return nil, errors.Errorf("contents[%d]: per-item Intent is not supported in batch requests; use context-level overrides for batch-wide input type", i)
			}
			if content.Dimension != nil {
				return nil, errors.Errorf("contents[%d]: per-item Dimension is not supported in batch requests", i)
			}
			if hints := content.ProviderHints; hints != nil {
				if _, ok := hints["input_type"]; ok {
					return nil, errors.Errorf("contents[%d]: per-item ProviderHints[\"input_type\"] is not supported in batch requests; use context-level overrides for batch-wide input type", i)
				}
			}
		}
	}

	inputs := make([]MultimodalInput, 0, len(contents))
	for i, content := range contents {
		input, convErr := convertToVoyageInput(ctx, content, maxMultimodalFileSize)
		if convErr != nil {
			return nil, errors.Wrapf(convErr, "content[%d]", i)
		}
		inputs = append(inputs, *input)
	}

	req := &CreateMultimodalEmbeddingRequest{
		Model:           string(e.getModel(ctx)),
		Inputs:          inputs,
		InputType:       inputType,
		Truncation:      e.getTruncation(ctx),
		OutputDimension: dimension,
	}

	resp, err := e.apiClient.CreateMultimodalEmbedding(ctx, req)
	if err != nil {
		return nil, err
	}

	embs := make([]embeddings.Embedding, 0, len(resp.Data))
	for _, result := range resp.Data {
		if result.Embedding == nil {
			return nil, errors.New("nil embedding in Voyage multimodal API response")
		}
		embs = append(embs, embeddings.NewEmbeddingFromFloat32(result.Embedding.Floats))
	}
	return embs, nil
}
