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

func TestNewImageURL(t *testing.T) {
	c := NewImageURL("https://example.com/img.png")
	require.Len(t, c.Parts, 1)
	require.Equal(t, ModalityImage, c.Parts[0].Modality)
	require.NotNil(t, c.Parts[0].Source)
	require.Equal(t, SourceKindURL, c.Parts[0].Source.Kind)
	require.Equal(t, "https://example.com/img.png", c.Parts[0].Source.URL)
}

func TestNewImageFile(t *testing.T) {
	c := NewImageFile("/tmp/photo.png")
	require.Len(t, c.Parts, 1)
	require.Equal(t, ModalityImage, c.Parts[0].Modality)
	require.NotNil(t, c.Parts[0].Source)
	require.Equal(t, SourceKindFile, c.Parts[0].Source.Kind)
	require.Equal(t, "/tmp/photo.png", c.Parts[0].Source.FilePath)
}

func TestNewVideoURL(t *testing.T) {
	c := NewVideoURL("https://example.com/vid.mp4")
	require.Len(t, c.Parts, 1)
	require.Equal(t, ModalityVideo, c.Parts[0].Modality)
	require.NotNil(t, c.Parts[0].Source)
	require.Equal(t, SourceKindURL, c.Parts[0].Source.Kind)
	require.Equal(t, "https://example.com/vid.mp4", c.Parts[0].Source.URL)
}

func TestNewVideoFile(t *testing.T) {
	c := NewVideoFile("/tmp/vid.mp4")
	require.Len(t, c.Parts, 1)
	require.Equal(t, ModalityVideo, c.Parts[0].Modality)
	require.NotNil(t, c.Parts[0].Source)
	require.Equal(t, SourceKindFile, c.Parts[0].Source.Kind)
	require.Equal(t, "/tmp/vid.mp4", c.Parts[0].Source.FilePath)
}

func TestNewAudioFile(t *testing.T) {
	c := NewAudioFile("/tmp/audio.wav")
	require.Len(t, c.Parts, 1)
	require.Equal(t, ModalityAudio, c.Parts[0].Modality)
	require.NotNil(t, c.Parts[0].Source)
	require.Equal(t, SourceKindFile, c.Parts[0].Source.Kind)
	require.Equal(t, "/tmp/audio.wav", c.Parts[0].Source.FilePath)
}

func TestNewPDFFile(t *testing.T) {
	c := NewPDFFile("/tmp/doc.pdf")
	require.Len(t, c.Parts, 1)
	require.Equal(t, ModalityPDF, c.Parts[0].Modality)
	require.NotNil(t, c.Parts[0].Source)
	require.Equal(t, SourceKindFile, c.Parts[0].Source.Kind)
	require.Equal(t, "/tmp/doc.pdf", c.Parts[0].Source.FilePath)
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
		{"NewAudioFile", NewAudioFile("/tmp/audio.wav")},
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
