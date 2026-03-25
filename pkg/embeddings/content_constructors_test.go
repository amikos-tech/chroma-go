package embeddings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTextContent(t *testing.T) {
	c := NewTextContent("hello")
	require.Len(t, c.Parts, 1)
	require.Equal(t, ModalityText, c.Parts[0].Modality)
	require.Equal(t, "hello", c.Parts[0].Text)
	require.Nil(t, c.Parts[0].Source)
}

func TestBinaryURLConstructors(t *testing.T) {
	tests := []struct {
		name     string
		ctor     func(string, ...ContentOption) Content
		modality Modality
		url      string
	}{
		{"NewImageURL", NewImageURL, ModalityImage, "https://example.com/img.png"},
		{"NewVideoURL", NewVideoURL, ModalityVideo, "https://example.com/vid.mp4"},
		{"NewAudioURL", NewAudioURL, ModalityAudio, "https://example.com/audio.mp3"},
		{"NewPDFURL", NewPDFURL, ModalityPDF, "https://example.com/doc.pdf"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.ctor(tc.url)
			require.Len(t, c.Parts, 1)
			require.Equal(t, tc.modality, c.Parts[0].Modality)
			require.NotNil(t, c.Parts[0].Source)
			require.Equal(t, SourceKindURL, c.Parts[0].Source.Kind)
			require.Equal(t, tc.url, c.Parts[0].Source.URL)
		})
	}
}

func TestBinaryFileConstructors(t *testing.T) {
	tests := []struct {
		name     string
		ctor     func(string, ...ContentOption) Content
		modality Modality
		path     string
	}{
		{"NewImageFile", NewImageFile, ModalityImage, "/tmp/photo.png"},
		{"NewVideoFile", NewVideoFile, ModalityVideo, "/tmp/vid.mp4"},
		{"NewAudioFile", NewAudioFile, ModalityAudio, "/tmp/audio.wav"},
		{"NewPDFFile", NewPDFFile, ModalityPDF, "/tmp/doc.pdf"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.ctor(tc.path)
			require.Len(t, c.Parts, 1)
			require.Equal(t, tc.modality, c.Parts[0].Modality)
			require.NotNil(t, c.Parts[0].Source)
			require.Equal(t, SourceKindFile, c.Parts[0].Source.Kind)
			require.Equal(t, tc.path, c.Parts[0].Source.FilePath)
		})
	}
}

func TestNewContent(t *testing.T) {
	parts := []Part{
		NewTextPart("a"),
		NewPartFromSource(ModalityImage, NewBinarySourceFromURL("u")),
	}
	c := NewContent(parts)
	require.Len(t, c.Parts, 2)
	require.Equal(t, ModalityText, c.Parts[0].Modality)
	require.Equal(t, "a", c.Parts[0].Text)
	require.Equal(t, ModalityImage, c.Parts[1].Modality)
	require.Equal(t, "u", c.Parts[1].Source.URL)
}

func TestNewContentWithOptions(t *testing.T) {
	parts := []Part{NewTextPart("a"), NewPartFromSource(ModalityImage, NewBinarySourceFromURL("u"))}
	c := NewContent(parts, WithIntent(IntentClustering), WithDimension(128))
	require.Len(t, c.Parts, 2)
	require.Equal(t, IntentClustering, c.Intent)
	require.NotNil(t, c.Dimension)
	require.Equal(t, 128, *c.Dimension)
}

func TestNewContentNilPartsFailsValidation(t *testing.T) {
	c := NewContent(nil)
	require.Error(t, c.Validate())
}

func TestNewContentEmptyPartsFailsValidation(t *testing.T) {
	c := NewContent([]Part{})
	require.Error(t, c.Validate())
}

func TestWithIntent(t *testing.T) {
	c := NewTextContent("q", WithIntent(IntentRetrievalQuery))
	require.Equal(t, IntentRetrievalQuery, c.Intent)
}

func TestWithDimension(t *testing.T) {
	c := NewTextContent("q", WithDimension(256))
	require.NotNil(t, c.Dimension)
	require.Equal(t, 256, *c.Dimension)
}

func TestWithDimensionNoAlias(t *testing.T) {
	c1 := NewTextContent("a", WithDimension(128))
	c2 := NewTextContent("b", WithDimension(256))
	require.NotSame(t, c1.Dimension, c2.Dimension)
	require.Equal(t, 128, *c1.Dimension)
	require.Equal(t, 256, *c2.Dimension)
}

func TestWithProviderHints(t *testing.T) {
	c := NewTextContent("q", WithProviderHints(map[string]any{"task_type": "CLASSIFICATION"}))
	require.NotNil(t, c.ProviderHints)
	require.Equal(t, "CLASSIFICATION", c.ProviderHints["task_type"])
}

func TestWithProviderHintsNoAlias(t *testing.T) {
	hints := map[string]any{"task_type": "CLASSIFICATION"}
	c := NewTextContent("q", WithProviderHints(hints))
	hints["task_type"] = "CLUSTERING"
	require.Equal(t, "CLASSIFICATION", c.ProviderHints["task_type"], "mutating original map must not affect Content")
}

func TestConstructorContentValidates(t *testing.T) {
	constructors := []struct {
		name    string
		content Content
	}{
		{"NewTextContent", NewTextContent("hello")},
		{"NewImageURL", NewImageURL("https://example.com/img.png")},
		{"NewImageFile", NewImageFile("/tmp/photo.png")},
		{"NewVideoURL", NewVideoURL("https://example.com/vid.mp4")},
		{"NewVideoFile", NewVideoFile("/tmp/vid.mp4")},
		{"NewAudioURL", NewAudioURL("https://example.com/audio.mp3")},
		{"NewAudioFile", NewAudioFile("/tmp/audio.wav")},
		{"NewPDFURL", NewPDFURL("https://example.com/doc.pdf")},
		{"NewPDFFile", NewPDFFile("/tmp/doc.pdf")},
		{"NewContent", NewContent([]Part{NewTextPart("a")})},
	}

	for _, tc := range constructors {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.content.Validate())
		})
	}
}

func TestMultipleOptions(t *testing.T) {
	c := NewImageFile("p", WithIntent(IntentClustering), WithDimension(512))
	require.Equal(t, IntentClustering, c.Intent)
	require.NotNil(t, c.Dimension)
	require.Equal(t, 512, *c.Dimension)
}
