package gemini

import (
	"context"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genai"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// neutralIntentToTaskType maps the 5 shared neutral intents to Gemini task type strings.
var neutralIntentToTaskType = map[embeddings.Intent]TaskType{
	embeddings.IntentRetrievalQuery:     TaskTypeRetrievalQuery,
	embeddings.IntentRetrievalDocument:  TaskTypeRetrievalDocument,
	embeddings.IntentClassification:     TaskTypeClassification,
	embeddings.IntentClustering:         TaskTypeClustering,
	embeddings.IntentSemanticSimilarity: TaskTypeSemanticSimilarity,
}

// extToMIME maps common file extensions to MIME types for MIME inference fallback.
var extToMIME = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".mp4":  "video/mp4",
	".mov":  "video/mov",
	".pdf":  "application/pdf",
}

// capabilitiesForModel returns the CapabilityMetadata for the given Gemini model.
// gemini-embedding-2-preview supports all 5 modalities; other models are text-only.
func capabilitiesForModel(model string) embeddings.CapabilityMetadata {
	switch model {
	case "gemini-embedding-2-preview":
		return embeddings.CapabilityMetadata{
			Modalities: []embeddings.Modality{
				embeddings.ModalityText,
				embeddings.ModalityImage,
				embeddings.ModalityAudio,
				embeddings.ModalityVideo,
				embeddings.ModalityPDF,
			},
			Intents: []embeddings.Intent{
				embeddings.IntentRetrievalQuery,
				embeddings.IntentRetrievalDocument,
				embeddings.IntentClassification,
				embeddings.IntentClustering,
				embeddings.IntentSemanticSimilarity,
			},
			RequestOptions: []embeddings.RequestOption{
				embeddings.RequestOptionDimension,
				embeddings.RequestOptionProviderHints,
			},
			SupportsBatch:     true,
			SupportsMixedPart: true,
		}
	default:
		return embeddings.CapabilityMetadata{
			Modalities: []embeddings.Modality{
				embeddings.ModalityText,
			},
			Intents: []embeddings.Intent{
				embeddings.IntentRetrievalQuery,
				embeddings.IntentRetrievalDocument,
				embeddings.IntentClassification,
				embeddings.IntentClustering,
				embeddings.IntentSemanticSimilarity,
			},
			RequestOptions: []embeddings.RequestOption{
				embeddings.RequestOptionDimension,
			},
			SupportsBatch:     true,
			SupportsMixedPart: false,
		}
	}
}

// resolveBytes fetches or reads the raw bytes for a binary source.
func resolveBytes(ctx context.Context, source *embeddings.BinarySource, maxFileSize int64) ([]byte, error) {
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
		// Reject before decoding: base64 encodes 3 bytes as 4 chars, so decoded size ≈ len * 3/4.
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
// It uses BinarySource.MIMEType directly if set, then falls back to file extension inference.
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
	case embeddings.ModalityAudio:
		if !strings.HasPrefix(mimeType, "audio/") {
			return errors.Errorf("audio modality requires audio/* MIME type, got %q", mimeType)
		}
	case embeddings.ModalityVideo:
		if !strings.HasPrefix(mimeType, "video/") {
			return errors.Errorf("video modality requires video/* MIME type, got %q", mimeType)
		}
	case embeddings.ModalityPDF:
		if mimeType != "application/pdf" {
			return errors.Errorf("pdf modality requires application/pdf MIME type, got %q", mimeType)
		}
	}
	return nil
}

// convertToGenaiContent converts a shared Content item to a *genai.Content for the Gemini API.
func convertToGenaiContent(ctx context.Context, content embeddings.Content, maxFileSize int64) (*genai.Content, error) {
	parts := make([]*genai.Part, 0, len(content.Parts))
	for i, part := range content.Parts {
		var gPart *genai.Part
		switch {
		case part.Modality == embeddings.ModalityText:
			gPart = genai.NewPartFromText(part.Text)
		case part.Source != nil && part.Source.Kind == embeddings.SourceKindURL:
			mimeType, err := resolveMIME(part.Source)
			if err != nil {
				return nil, errors.Wrapf(err, "part[%d]", i)
			}
			if err := validateMIMEModality(part.Modality, mimeType); err != nil {
				return nil, errors.Wrapf(err, "part[%d]", i)
			}
			gPart = genai.NewPartFromURI(part.Source.URL, mimeType)
		default:
			mimeType, err := resolveMIME(part.Source)
			if err != nil {
				return nil, errors.Wrapf(err, "part[%d]", i)
			}
			if err := validateMIMEModality(part.Modality, mimeType); err != nil {
				return nil, errors.Wrapf(err, "part[%d]", i)
			}
			data, err := resolveBytes(ctx, part.Source, maxFileSize)
			if err != nil {
				return nil, errors.Wrapf(err, "part[%d]", i)
			}
			gPart = genai.NewPartFromBytes(data, mimeType)
		}
		parts = append(parts, gPart)
	}
	return genai.NewContentFromParts(parts, genai.RoleUser), nil
}

// convertToGenaiContents converts a slice of Content items to []*genai.Content.
func convertToGenaiContents(ctx context.Context, contents []embeddings.Content, maxFileSize int64) ([]*genai.Content, error) {
	result := make([]*genai.Content, 0, len(contents))
	for i, content := range contents {
		gc, err := convertToGenaiContent(ctx, content, maxFileSize)
		if err != nil {
			return nil, errors.Wrapf(err, "content[%d]", i)
		}
		result = append(result, gc)
	}
	return result, nil
}

// resolveTaskTypeForContent determines the effective Gemini task type for a single content item.
// Priority: ProviderHints["task_type"] > intent via mapper > defaultTaskType.
func resolveTaskTypeForContent(content embeddings.Content, defaultTaskType TaskType, mapper embeddings.IntentMapper) (TaskType, error) {
	if hints := content.ProviderHints; hints != nil {
		if raw, ok := hints["task_type"]; ok {
			hintStr, ok := raw.(string)
			if !ok || hintStr == "" {
				return "", errors.New("ProviderHints[\"task_type\"] must be a non-empty string")
			}
			tt := TaskType(hintStr)
			if !tt.IsValid() {
				return "", errors.Errorf("invalid Gemini task type in ProviderHints: %q", hintStr)
			}
			return tt, nil
		}
	}

	if content.Intent != "" && mapper != nil {
		mapped, err := mapper.MapIntent(content.Intent)
		if err != nil {
			return "", errors.Wrap(err, "failed to map intent to Gemini task type")
		}
		tt := TaskType(mapped)
		if !tt.IsValid() {
			return "", errors.Errorf("invalid Gemini task type from IntentMapper: %q", mapped)
		}
		return tt, nil
	}

	return defaultTaskType, nil
}
