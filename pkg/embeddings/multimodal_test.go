package embeddings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultimodalContentSupportsAllModalities(t *testing.T) {
	pdfBytes := []byte{0x25, 0x50, 0x44, 0x46}
	pdfSource := NewBinarySourceFromBytes(pdfBytes)
	pdfBytes[0] = 0x00

	cases := []struct {
		name       string
		content    Content
		wantPart   Part
		sourceKind SourceKind
	}{
		{
			name:     "text",
			content:  Content{Parts: []Part{NewTextPart("text payload")}},
			wantPart: Part{Modality: ModalityText, Text: "text payload"},
		},
		{
			name:       "image",
			content:    Content{Parts: []Part{NewPartFromSource(ModalityImage, NewBinarySourceFromURL("https://example.com/image.png"))}},
			wantPart:   Part{Modality: ModalityImage},
			sourceKind: SourceKindURL,
		},
		{
			name:       "audio",
			content:    Content{Parts: []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromFile("/tmp/audio.wav"))}},
			wantPart:   Part{Modality: ModalityAudio},
			sourceKind: SourceKindFile,
		},
		{
			name:       "video",
			content:    Content{Parts: []Part{NewPartFromSource(ModalityVideo, NewBinarySourceFromBase64("Yml0cw=="))}},
			wantPart:   Part{Modality: ModalityVideo},
			sourceKind: SourceKindBase64,
		},
		{
			name:       "pdf",
			content:    Content{Parts: []Part{NewPartFromSource(ModalityPDF, pdfSource)}},
			wantPart:   Part{Modality: ModalityPDF},
			sourceKind: SourceKindBytes,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.content.Validate())
			require.Len(t, tc.content.Parts, 1)

			part := tc.content.Parts[0]
			require.Equal(t, tc.wantPart.Modality, part.Modality)
			require.Equal(t, tc.wantPart.Text, part.Text)

			if tc.wantPart.Modality == ModalityText {
				require.Nil(t, part.Source)
				return
			}

			require.NotNil(t, part.Source)
			require.Equal(t, tc.sourceKind, part.Source.Kind)
		})
	}

	require.Equal(t, []byte{0x25, 0x50, 0x44, 0x46}, cases[4].content.Parts[0].Source.Bytes)
}

func TestMultimodalContentPreservesOrder(t *testing.T) {
	parts := []Part{
		NewTextPart("first"),
		NewPartFromSource(ModalityImage, NewBinarySourceFromURL("https://example.com/image.png")),
		NewPartFromSource(ModalityAudio, NewBinarySourceFromFile("/tmp/audio.wav")),
		NewPartFromSource(ModalityVideo, NewBinarySourceFromBase64("Yml0cw==")),
		NewPartFromSource(ModalityPDF, NewBinarySourceFromBytes([]byte{0x25, 0x50, 0x44, 0x46})),
	}
	content := Content{Parts: append([]Part(nil), parts...)}
	batch := []Content{
		content,
		{Parts: []Part{NewTextPart("second")}},
	}
	originalBatch := append([]Content(nil), batch...)
	wantPartModalities := []Modality{ModalityText, ModalityImage, ModalityAudio, ModalityVideo, ModalityPDF}
	wantTexts := []string{"first", "", "", "", ""}

	require.NoError(t, content.Validate())
	require.NoError(t, ValidateContents(batch))
	require.Equal(t, parts, content.Parts)
	require.Equal(t, originalBatch, batch)

	for i, part := range batch[0].Parts {
		require.Equal(t, wantPartModalities[i], part.Modality)
		require.Equal(t, wantTexts[i], part.Text)
	}

	require.Len(t, batch, 2)
	require.Len(t, batch[1].Parts, 1)
	require.Equal(t, ModalityText, batch[1].Parts[0].Modality)
	require.Equal(t, "second", batch[1].Parts[0].Text)
}

func TestMultimodalRequestOptions(t *testing.T) {
	dimension := 384
	hints := map[string]any{
		"provider":     "gemini",
		"allow_remote": true,
	}
	content := Content{
		Parts: []Part{
			NewTextPart("request"),
		},
		Intent:        IntentSemanticSimilarity,
		Dimension:     &dimension,
		ProviderHints: hints,
	}

	require.NoError(t, content.Validate())
	require.Equal(t, IntentSemanticSimilarity, content.Intent)
	require.Same(t, &dimension, content.Dimension)
	require.Equal(t, hints, content.ProviderHints)

	batch := []Content{content}
	require.NoError(t, ValidateContents(batch))
	require.Equal(t, IntentSemanticSimilarity, batch[0].Intent)
	require.Same(t, &dimension, batch[0].Dimension)
	require.Equal(t, hints, batch[0].ProviderHints)
	require.Equal(t, "gemini", batch[0].ProviderHints["provider"])
	require.Equal(t, true, batch[0].ProviderHints["allow_remote"])
}
