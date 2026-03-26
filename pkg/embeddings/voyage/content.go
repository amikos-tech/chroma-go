package voyage

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/internal/pathutil"
)

const (
	defaultMultimodalModel = "voyage-multimodal-3.5"
	maxMultimodalFileSize  = 100 * 1024 * 1024 // 100 MB
)

// MultimodalContentBlock represents a single block in a Voyage multimodal request.
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
func resolveBytes(source *embeddings.BinarySource, maxFileSize int64) ([]byte, error) {
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
		cleaned, err := pathutil.ValidateFilePath(source.FilePath)
		if err != nil {
			return nil, errors.Wrap(err, "invalid file source path")
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

// resolveMIME determines the MIME type for a binary source.
// It uses BinarySource.MIMEType directly if set, then falls back to file extension,
// then to URL path extension inference.
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
	if source.URL != "" {
		u, err := url.Parse(source.URL)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse source URL for MIME inference")
		}
		ext := strings.ToLower(filepath.Ext(u.Path))
		if mime, ok := extToMIME[ext]; ok {
			return mime, nil
		}
	}
	return "", errors.New("MIME type is required: set BinarySource.MIMEType or use a file/URL with a known extension")
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
	default:
		return errors.Errorf("MIME validation not implemented for modality %q", modality)
	}
	return nil
}

// convertToVoyageInput converts a shared Content item to a Voyage MultimodalInput.
func convertToVoyageInput(content embeddings.Content, maxFileSize int64) (*MultimodalInput, error) {
	blocks := make([]MultimodalContentBlock, 0, len(content.Parts))
	for i, part := range content.Parts {
		block, err := buildContentBlock(part, maxFileSize)
		if err != nil {
			return nil, errors.Wrapf(err, "part[%d]", i)
		}
		blocks = append(blocks, block)
	}
	return &MultimodalInput{Content: blocks}, nil
}

// buildContentBlock converts a single Part to a Voyage MultimodalContentBlock.
func buildContentBlock(part embeddings.Part, maxFileSize int64) (MultimodalContentBlock, error) {
	switch part.Modality {
	case embeddings.ModalityText:
		return MultimodalContentBlock{Type: "text", Text: part.Text}, nil
	case embeddings.ModalityImage:
		return buildBinaryBlock(part.Source, embeddings.ModalityImage, maxFileSize)
	case embeddings.ModalityVideo:
		return buildBinaryBlock(part.Source, embeddings.ModalityVideo, maxFileSize)
	default:
		return MultimodalContentBlock{}, errors.Errorf("unsupported modality %q", part.Modality)
	}
}

// buildBinaryBlock converts a binary source to a Voyage content block.
// For URL sources, it passes through as image_url/video_url.
// For all other kinds, it resolves bytes and encodes as a data URI.
func buildBinaryBlock(source *embeddings.BinarySource, modality embeddings.Modality, maxFileSize int64) (MultimodalContentBlock, error) {
	if source == nil {
		return MultimodalContentBlock{}, errors.New("binary source is required for non-text parts")
	}
	if source.Kind == embeddings.SourceKindURL {
		block := MultimodalContentBlock{Type: string(modality) + "_url"}
		switch modality {
		case embeddings.ModalityImage:
			block.ImageURL = source.URL
		case embeddings.ModalityVideo:
			block.VideoURL = source.URL
		default:
			return MultimodalContentBlock{}, errors.Errorf("unsupported modality %q for URL source", modality)
		}
		return block, nil
	}

	mimeType, err := resolveMIME(source)
	if err != nil {
		return MultimodalContentBlock{}, err
	}
	if err := validateMIMEModality(modality, mimeType); err != nil {
		return MultimodalContentBlock{}, err
	}
	data, err := resolveBytes(source, maxFileSize)
	if err != nil {
		return MultimodalContentBlock{}, err
	}

	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
	block := MultimodalContentBlock{Type: string(modality) + "_base64"}
	switch modality {
	case embeddings.ModalityImage:
		block.ImageBase64 = dataURI
	case embeddings.ModalityVideo:
		block.VideoBase64 = dataURI
	default:
		return MultimodalContentBlock{}, errors.Errorf("unsupported modality %q for binary source", modality)
	}
	return block, nil
}

// MapIntent translates a neutral shared intent to a Voyage input_type string.
func (e *VoyageAIEmbeddingFunction) MapIntent(intent embeddings.Intent) (string, error) {
	switch intent {
	case embeddings.IntentRetrievalQuery:
		return string(InputTypeQuery), nil
	case embeddings.IntentRetrievalDocument:
		return string(InputTypeDocument), nil
	default:
		return "", errors.Errorf("intent %q is not supported by Voyage; only retrieval_query and retrieval_document are available; for Voyage-native types use ProviderHints[\"input_type\"]", intent)
	}
}

// Capabilities returns the capability metadata for the configured Voyage model.
func (e *VoyageAIEmbeddingFunction) Capabilities() embeddings.CapabilityMetadata {
	return capabilitiesForModel(string(e.apiClient.DefaultModel))
}

// capabilitiesForContext returns capabilities for the effective model,
// honoring any model override set in the context.
func (e *VoyageAIEmbeddingFunction) capabilitiesForContext(ctx context.Context) embeddings.CapabilityMetadata {
	return capabilitiesForModel(string(e.getModel(ctx)))
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
func multimodalURL(baseAPI string) (string, error) {
	if base, ok := strings.CutSuffix(baseAPI, "/v1/embeddings"); ok {
		return base + "/v1/multimodalembeddings", nil
	}
	return "", errors.Errorf("cannot derive multimodal endpoint from BaseAPI %q: expected path ending in /v1/embeddings", baseAPI)
}

// contextInputType extracts the context-level input type if set, otherwise returns nil.
func contextInputType(ctx context.Context) *InputType {
	if it, ok := ctx.Value(inputTypeContextKey).(*InputType); ok {
		return it
	}
	return nil
}

// EmbedContent embeds a single multimodal content item using the shared Content API.
func (e *VoyageAIEmbeddingFunction) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	if err := content.Validate(); err != nil {
		return nil, err
	}
	caps := e.capabilitiesForContext(ctx)
	if err := embeddings.ValidateContentSupport(content, caps); err != nil {
		return nil, err
	}

	inputType, err := resolveInputTypeForContent(content, contextInputType(ctx), e)
	if err != nil {
		return nil, err
	}

	input, err := convertToVoyageInput(content, maxMultimodalFileSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert content to Voyage format")
	}

	req := &CreateMultimodalEmbeddingRequest{
		Model:           string(e.getModel(ctx)),
		Inputs:          []MultimodalInput{*input},
		InputType:       inputType,
		Truncation:      e.getTruncation(ctx),
		OutputDimension: content.Dimension,
	}

	resp, err := e.apiClient.CreateMultimodalEmbedding(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed multimodal content")
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding returned from Voyage multimodal API")
	}
	if resp.Data[0].Embedding == nil {
		return nil, errors.New("nil embedding in Voyage multimodal API response")
	}
	if len(resp.Data[0].Embedding.Floats) == 0 {
		return nil, errors.New("empty embedding vector in Voyage multimodal API response")
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
	caps := e.capabilitiesForContext(ctx)
	if err := embeddings.ValidateContentsSupport(contents, caps); err != nil {
		return nil, err
	}

	var inputType *InputType
	var dimension *int
	var err error

	if len(contents) == 1 {
		inputType, err = resolveInputTypeForContent(contents[0], contextInputType(ctx), e)
		if err != nil {
			return nil, err
		}
		dimension = contents[0].Dimension
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
		inputType = contextInputType(ctx)
	}

	inputs := make([]MultimodalInput, 0, len(contents))
	for i, content := range contents {
		input, convErr := convertToVoyageInput(content, maxMultimodalFileSize)
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
		return nil, errors.Wrap(err, "failed to embed multimodal contents batch")
	}
	if len(resp.Data) != len(contents) {
		return nil, errors.Errorf("expected %d embeddings from Voyage multimodal API, got %d", len(contents), len(resp.Data))
	}

	embs := make([]embeddings.Embedding, 0, len(resp.Data))
	for _, result := range resp.Data {
		if result.Embedding == nil {
			return nil, errors.New("nil embedding in Voyage multimodal API response")
		}
		if len(result.Embedding.Floats) == 0 {
			return nil, errors.New("empty embedding vector in Voyage multimodal API response")
		}
		embs = append(embs, embeddings.NewEmbeddingFromFloat32(result.Embedding.Floats))
	}
	return embs, nil
}
