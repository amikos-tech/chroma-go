package twelvelabs

import (
	"context"
	"encoding/base64"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/amikos-tech/chroma-go/pkg/internal/pathutil"
)

const maxMediaSourceSize int64 = 100 * 1024 * 1024 // 100 MB

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
		data, err := io.ReadAll(io.LimitReader(f, maxMediaSourceSize+1))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read file source")
		}
		if int64(len(data)) > maxMediaSourceSize {
			return nil, errors.Errorf("file size exceeds maximum of %d bytes", maxMediaSourceSize)
		}
		return data, nil
	case embeddings.SourceKindBase64:
		if source.Base64 == "" {
			return nil, errors.New("base64 source must include non-empty data")
		}
		if int64(len(source.Base64))*3/4 > maxMediaSourceSize {
			return nil, errors.Errorf("base64 payload too large: estimated decoded size exceeds maximum of %d bytes", maxMediaSourceSize)
		}
		data, err := base64.StdEncoding.DecodeString(source.Base64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base64 source")
		}
		if int64(len(data)) > maxMediaSourceSize {
			return nil, errors.Errorf("base64 payload exceeds maximum of %d bytes", maxMediaSourceSize)
		}
		return data, nil
	case embeddings.SourceKindBytes:
		if len(source.Bytes) == 0 {
			return nil, errors.New("bytes source must include non-empty bytes")
		}
		if int64(len(source.Bytes)) > maxMediaSourceSize {
			return nil, errors.Errorf("bytes payload size %d exceeds maximum of %d bytes", len(source.Bytes), maxMediaSourceSize)
		}
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
		if part.Text == "" {
			return nil, errors.New("text part must have non-empty text")
		}
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

// embedContent sends a single Content item to the API and returns the embedding.
// When asyncPollingEnabled is true and the modality is audio or video, the
// content routes through the tasks endpoint + polling loop (CONTEXT.md D-07).
// All other cases use the sync /embed-v2 path (D-08 — zero change for non-opt-in).
func (e *TwelveLabsEmbeddingFunction) embedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	if e.apiClient.asyncPollingEnabled && len(content.Parts) == 1 {
		switch content.Parts[0].Modality {
		case embeddings.ModalityAudio, embeddings.ModalityVideo:
			return e.createTaskAndPoll(ctx, content)
		}
	}
	req, err := contentToRequest(content, e.resolveModel(ctx), e.apiClient.AudioEmbeddingOption)
	if err != nil {
		return nil, err
	}
	resp, err := e.doPost(ctx, *req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed content")
	}
	return embeddingFromResponse(resp)
}

// EmbedContent embeds a single multimodal Content item.
func (e *TwelveLabsEmbeddingFunction) EmbedContent(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	caps := e.Capabilities()
	if err := content.Validate(); err != nil {
		return nil, errors.Wrap(err, "Twelve Labs content validation failed")
	}
	if err := embeddings.ValidateContentSupport(content, caps); err != nil {
		return nil, errors.Wrap(err, "Twelve Labs content validation failed")
	}
	return e.embedContent(ctx, content)
}

// EmbedContents embeds multiple Content items (one API call per item).
func (e *TwelveLabsEmbeddingFunction) EmbedContents(ctx context.Context, contents []embeddings.Content) ([]embeddings.Embedding, error) {
	caps := e.Capabilities()
	if err := embeddings.ValidateContents(contents); err != nil {
		return nil, errors.Wrap(err, "Twelve Labs content validation failed")
	}
	if err := embeddings.ValidateContentsSupport(contents, caps); err != nil {
		return nil, errors.Wrap(err, "Twelve Labs content validation failed")
	}

	result := make([]embeddings.Embedding, 0, len(contents))
	for i, c := range contents {
		emb, err := e.embedContent(ctx, c)
		if err != nil {
			return nil, errors.Wrapf(err, "contents[%d]", i)
		}
		result = append(result, emb)
	}
	return result, nil
}

// Capabilities returns the provider capability metadata.
// The Twelve Labs embed-v2 API does not support intent differentiation;
// Marengo operates in a unified embedding space.
func (e *TwelveLabsEmbeddingFunction) Capabilities() embeddings.CapabilityMetadata {
	return embeddings.CapabilityMetadata{
		Modalities: []embeddings.Modality{
			embeddings.ModalityText,
			embeddings.ModalityImage,
			embeddings.ModalityAudio,
			embeddings.ModalityVideo,
		},
		SupportsBatch:     false,
		SupportsMixedPart: false,
	}
}
