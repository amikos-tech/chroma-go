package twelvelabs

import (
	"context"
	"encoding/base64"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/internal/pathutil"
)

var extToMIME = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
	".webp": "image/webp",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".mp4":  "video/mp4",
	".mpeg": "video/mpeg",
	".mov":  "video/quicktime",
	".webm": "video/webm",
	".avi":  "video/x-msvideo",
}

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

func resolveBytes(source *embeddings.BinarySource) ([]byte, error) {
	if source == nil {
		return nil, errors.New("source cannot be nil")
	}
	switch source.Kind {
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
		data, err := io.ReadAll(f)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file source")
		}
		return data, nil
	case embeddings.SourceKindBase64:
		data, err := base64.StdEncoding.DecodeString(source.Base64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base64 source")
		}
		return data, nil
	case embeddings.SourceKindBytes:
		return source.Bytes, nil
	case embeddings.SourceKindURL:
		return nil, errors.New("URL sources are handled via direct passthrough, not byte resolution")
	default:
		return nil, errors.Errorf("unsupported source kind %q", source.Kind)
	}
}

func contentToRequest(content embeddings.Content, model string, audioOpt string) (*EmbedV2Request, error) {
	if len(content.Parts) != 1 {
		return nil, errors.Errorf("Twelve Labs requires exactly one part per Content item, got %d", len(content.Parts))
	}
	part := content.Parts[0]
	req := &EmbedV2Request{ModelName: model}

	switch part.Modality {
	case embeddings.ModalityText:
		req.InputType = "text"
		req.Text = &TextInput{InputText: part.Text}

	case embeddings.ModalityImage:
		req.InputType = "image"
		ms, err := buildMediaSource(part.Source)
		if err != nil {
			return nil, errors.Wrap(err, "image source")
		}
		req.Image = &ImageInput{MediaSource: ms}

	case embeddings.ModalityAudio:
		req.InputType = "audio"
		ms, err := buildMediaSource(part.Source)
		if err != nil {
			return nil, errors.Wrap(err, "audio source")
		}
		req.Audio = &AudioInput{MediaSource: ms, EmbeddingOption: audioOpt}

	case embeddings.ModalityVideo:
		req.InputType = "video"
		ms, err := buildMediaSource(part.Source)
		if err != nil {
			return nil, errors.Wrap(err, "video source")
		}
		req.Video = &VideoInput{MediaSource: ms}

	default:
		return nil, errors.Errorf("unsupported modality %q: Twelve Labs supports text, image, audio, and video", part.Modality)
	}
	return req, nil
}

func buildMediaSource(source *embeddings.BinarySource) (MediaSource, error) {
	if source == nil {
		return MediaSource{}, errors.New("binary source is required for non-text parts")
	}
	if source.Kind == embeddings.SourceKindURL {
		return MediaSource{URL: source.URL}, nil
	}
	data, err := resolveBytes(source)
	if err != nil {
		return MediaSource{}, err
	}
	return MediaSource{Base64String: base64.StdEncoding.EncodeToString(data)}, nil
}

// EmbedContent embeds a single multimodal Content item.
func (e *TwelveLabsEmbeddingFunction) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	if err := content.Validate(); err != nil {
		return nil, err
	}
	caps := e.Capabilities()
	if err := embeddings.ValidateContentSupport(content, caps); err != nil {
		return nil, err
	}

	req, err := contentToRequest(content, e.resolveModel(ctx), e.apiClient.AudioEmbeddingOption)
	if err != nil {
		return nil, err
	}
	resp, err := e.doPost(ctx, *req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed content")
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding returned from Twelve Labs API")
	}
	return embeddings.NewEmbeddingFromFloat32(float64sToFloat32s(resp.Data[0].Embedding)), nil
}

// EmbedContents embeds multiple Content items (one API call per item).
func (e *TwelveLabsEmbeddingFunction) EmbedContents(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
	if err := embeddings.ValidateContents(contents); err != nil {
		return nil, err
	}
	caps := e.Capabilities()
	if err := embeddings.ValidateContentsSupport(contents, caps); err != nil {
		return nil, err
	}

	result := make([]embeddings.Embedding, 0, len(contents))
	for i, c := range contents {
		emb, err := e.EmbedContent(ctx, c)
		if err != nil {
			return nil, errors.Wrapf(err, "contents[%d]", i)
		}
		result = append(result, emb)
	}
	return result, nil
}

// Capabilities returns the provider capability metadata.
func (e *TwelveLabsEmbeddingFunction) Capabilities() embeddings.CapabilityMetadata {
	return embeddings.CapabilityMetadata{
		Modalities: []embeddings.Modality{
			embeddings.ModalityText,
			embeddings.ModalityImage,
			embeddings.ModalityAudio,
			embeddings.ModalityVideo,
		},
		Intents: []embeddings.Intent{
			embeddings.IntentRetrievalQuery,
			embeddings.IntentRetrievalDocument,
		},
		SupportsBatch:     false,
		SupportsMixedPart: false,
	}
}

// MapIntent translates a neutral intent to a Twelve Labs input_type hint.
func (e *TwelveLabsEmbeddingFunction) MapIntent(intent embeddings.Intent) (string, error) {
	switch intent {
	case embeddings.IntentRetrievalQuery:
		return "query", nil
	case embeddings.IntentRetrievalDocument:
		return "document", nil
	default:
		if embeddings.IsNeutralIntent(intent) {
			return "", errors.Errorf("intent %q is not supported by Twelve Labs; only retrieval_query and retrieval_document are available", intent)
		}
		return string(intent), nil
	}
}
