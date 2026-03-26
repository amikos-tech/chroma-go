package embeddings

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultimodalIntentValidation(t *testing.T) {
	require.NoError(t, Content{
		Parts: []Part{NewTextPart("ok")},
	}.Validate())

	require.NoError(t, Content{
		Parts:  []Part{NewTextPart("ok")},
		Intent: IntentRetrievalQuery,
	}.Validate())

	require.NoError(t, Content{
		Parts:  []Part{NewTextPart("ok")},
		Intent: Intent("custom.intent"),
	}.Validate())

	err := Content{
		Parts:  []Part{NewTextPart("ok")},
		Intent: Intent("   "),
	}.Validate()
	require.Error(t, err)

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.NotEmpty(t, validationErr.Issues)
	require.Equal(t, "intent", validationErr.Issues[0].Path)
	require.Equal(t, validationCodeInvalidValue, validationErr.Issues[0].Code)

	err = Content{
		Parts:  []Part{NewTextPart("ok")},
		Intent: Intent(" retrieval_query "),
	}.Validate()
	require.Error(t, err)
	require.ErrorAs(t, err, &validationErr)
	require.NotEmpty(t, validationErr.Issues)
	require.Equal(t, "intent", validationErr.Issues[0].Path)
	require.Equal(t, validationCodeInvalidValue, validationErr.Issues[0].Code)
}

func TestMultimodalValidationErrors(t *testing.T) {
	cases := []struct {
		name          string
		err           error
		wantPath      string
		wantCode      string
		issueCountMin int
	}{
		{
			name:          "empty content",
			err:           Content{}.Validate(),
			wantPath:      "parts",
			wantCode:      validationCodeRequired,
			issueCountMin: 1,
		},
		{
			name:          "empty parts",
			err:           Content{Parts: []Part{}}.Validate(),
			wantPath:      "parts",
			wantCode:      validationCodeRequired,
			issueCountMin: 1,
		},
		{
			name: "invalid source kind mismatch",
			err: Part{
				Modality: ModalityImage,
				Source: &BinarySource{
					Kind:     SourceKindURL,
					FilePath: "/tmp/image.png",
				},
			}.Validate(),
			wantPath:      "source.kind",
			wantCode:      validationCodeMismatch,
			issueCountMin: 1,
		},
		{
			name: "invalid source payload combination",
			err: Part{
				Modality: ModalityImage,
				Source: &BinarySource{
					Kind:     SourceKindFile,
					URL:      "https://example.com/image.png",
					FilePath: "/tmp/image.png",
				},
			}.Validate(),
			wantPath:      "source.payload",
			wantCode:      validationCodeOneOf,
			issueCountMin: 1,
		},
		{
			name: "invalid dimension",
			err: Content{
				Parts:     []Part{NewTextPart("ok")},
				Dimension: func() *int { v := 0; return &v }(),
			}.Validate(),
			wantPath:      "dimension",
			wantCode:      validationCodeOutOfRange,
			issueCountMin: 1,
		},
		{
			name: "dimension exceeds MaxInt32",
			err: Content{
				Parts:     []Part{NewTextPart("ok")},
				Dimension: func() *int { v := math.MaxInt32 + 1; return &v }(),
			}.Validate(),
			wantPath:      "dimension",
			wantCode:      validationCodeOutOfRange,
			issueCountMin: 1,
		},
		{
			name:          "missing modality",
			err:           Part{}.Validate(),
			wantPath:      "modality",
			wantCode:      validationCodeRequired,
			issueCountMin: 1,
		},
		{
			name: "unknown modality",
			err: Part{
				Modality: Modality("hologram"),
				Source:   &BinarySource{Kind: SourceKindURL, URL: "https://example.com/holo"},
			}.Validate(),
			wantPath:      "modality",
			wantCode:      validationCodeInvalidValue,
			issueCountMin: 1,
		},
		{
			name:          "empty text part",
			err:           Part{Modality: ModalityText, Text: ""}.Validate(),
			wantPath:      "text",
			wantCode:      validationCodeRequired,
			issueCountMin: 1,
		},
		{
			name: "text field on non-text part",
			err: Part{
				Modality: ModalityImage,
				Text:     "should not be here",
				Source:   &BinarySource{Kind: SourceKindURL, URL: "https://example.com/image.png"},
			}.Validate(),
			wantPath:      "text",
			wantCode:      validationCodeForbidden,
			issueCountMin: 1,
		},
		{
			name:          "ValidateContents with nil input",
			err:           ValidateContents(nil),
			wantPath:      "contents",
			wantCode:      validationCodeRequired,
			issueCountMin: 1,
		},
		{
			name:          "ValidateContents with empty slice",
			err:           ValidateContents([]Content{}),
			wantPath:      "contents",
			wantCode:      validationCodeRequired,
			issueCountMin: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.err)

			var validationErr *ValidationError
			require.ErrorAs(t, tc.err, &validationErr)
			require.GreaterOrEqual(t, len(validationErr.Issues), tc.issueCountMin)
			require.Equal(t, tc.wantPath, validationErr.Issues[0].Path)
			require.Equal(t, tc.wantCode, validationErr.Issues[0].Code)
			require.NotEmpty(t, validationErr.Issues[0].Message)
		})
	}
}

func TestBinarySourceMIMETypeValidation(t *testing.T) {
	valid := []string{
		"image/png",
		"image/jpeg",
		"audio/mpeg",
		"video/mp4",
		"application/pdf",
		"application/octet-stream",
	}
	for _, mime := range valid {
		t.Run("valid_"+mime, func(t *testing.T) {
			src := BinarySource{Kind: SourceKindBytes, Bytes: []byte("x"), MIMEType: mime}
			require.NoError(t, src.Validate())
		})
	}

	invalid := []struct {
		name string
		mime string
	}{
		{"empty_type", "/png"},
		{"empty_subtype", "image/"},
		{"no_slash", "imagepng"},
		{"newline_injection", "image/png\nX-Evil: true"},
		{"space_in_type", "im age/png"},
		{"semicolon_injection", "image/png;base64,AAAA"},
	}
	for _, tc := range invalid {
		t.Run("invalid_"+tc.name, func(t *testing.T) {
			src := BinarySource{Kind: SourceKindBytes, Bytes: []byte("x"), MIMEType: tc.mime}
			err := src.Validate()
			require.Error(t, err)
			var validationErr *ValidationError
			require.ErrorAs(t, err, &validationErr)
			found := false
			for _, issue := range validationErr.Issues {
				if issue.Path == "mime_type" {
					found = true
					require.Equal(t, validationCodeInvalidValue, issue.Code)
				}
			}
			require.True(t, found, "expected mime_type validation issue")
		})
	}

	t.Run("empty MIMEType is allowed", func(t *testing.T) {
		src := BinarySource{Kind: SourceKindBytes, Bytes: []byte("x")}
		require.NoError(t, src.Validate())
	})
}

func TestNewImagePartFromImageInput(t *testing.T) {
	urlPart, err := NewImagePartFromImageInput(NewImageInputFromURL("https://example.com/image.png"))
	require.NoError(t, err)
	require.Equal(t, ModalityImage, urlPart.Modality)
	require.NotNil(t, urlPart.Source)
	require.Equal(t, SourceKindURL, urlPart.Source.Kind)
	require.Equal(t, "https://example.com/image.png", urlPart.Source.URL)
	require.Empty(t, urlPart.Source.FilePath)
	require.Empty(t, urlPart.Source.Base64)
	require.NoError(t, urlPart.Validate())

	filePart, err := NewImagePartFromImageInput(NewImageInputFromFile("/tmp/image.png"))
	require.NoError(t, err)
	require.NotNil(t, filePart.Source)
	require.Equal(t, SourceKindFile, filePart.Source.Kind)
	require.Equal(t, "/tmp/image.png", filePart.Source.FilePath)
	require.Empty(t, filePart.Source.URL)
	require.Empty(t, filePart.Source.Base64)
	require.NoError(t, filePart.Validate())

	base64Part, err := NewImagePartFromImageInput(NewImageInputFromBase64("Yml0cw=="))
	require.NoError(t, err)
	require.NotNil(t, base64Part.Source)
	require.Equal(t, SourceKindBase64, base64Part.Source.Kind)
	require.Equal(t, "Yml0cw==", base64Part.Source.Base64)
	require.Empty(t, base64Part.Source.URL)
	require.Empty(t, base64Part.Source.FilePath)
	require.NoError(t, base64Part.Validate())

	_, err = NewImagePartFromImageInput(ImageInput{})
	require.Error(t, err)
	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.NotEmpty(t, validationErr.Issues)
	require.Equal(t, "input", validationErr.Issues[0].Path)
	require.Equal(t, validationCodeRequired, validationErr.Issues[0].Code)

	_, err = NewImagePartFromImageInput(ImageInput{URL: "https://example.com/image.png", FilePath: "/tmp/image.png"})
	require.Error(t, err)
	require.ErrorAs(t, err, &validationErr)
	require.NotEmpty(t, validationErr.Issues)
	require.Equal(t, "input", validationErr.Issues[0].Path)
	require.Equal(t, validationCodeOneOf, validationErr.Issues[0].Code)
}
