package embeddings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateContentSupportModality(t *testing.T) {
	caps := CapabilityMetadata{Modalities: []Modality{ModalityText, ModalityImage}}
	content := Content{
		Parts: []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromURL("https://example.com/audio.wav"))},
	}
	err := ValidateContentSupport(content, caps)
	requireValidationIssue(t, err, "parts[0].modality", validationCodeUnsupportedModality, "audio")
}

func TestValidateContentSupportIntent(t *testing.T) {
	caps := CapabilityMetadata{Intents: []Intent{IntentRetrievalQuery, IntentSemanticSimilarity}}
	content := Content{
		Parts:  []Part{NewTextPart("text")},
		Intent: IntentClassification,
	}
	err := ValidateContentSupport(content, caps)
	requireValidationIssue(t, err, "intent", validationCodeUnsupportedIntent, "classification")
}

func TestValidateContentSupportDimension(t *testing.T) {
	caps := CapabilityMetadata{RequestOptions: []RequestOption{RequestOptionProviderHints}}
	dim := 128
	content := Content{
		Parts:     []Part{NewTextPart("text")},
		Dimension: &dim,
	}
	err := ValidateContentSupport(content, caps)
	requireValidationIssue(t, err, "dimension", validationCodeUnsupportedDimension, "dimension override")
}

func TestValidateContentSupportPassThrough(t *testing.T) {
	caps := CapabilityMetadata{}
	content := Content{
		Parts:  []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromURL("https://example.com/audio.wav"))},
		Intent: IntentClassification,
	}
	err := ValidateContentSupport(content, caps)
	require.NoError(t, err)
}

func TestValidateContentSupportCustomIntentBypass(t *testing.T) {
	caps := CapabilityMetadata{Intents: []Intent{IntentRetrievalQuery}}
	content := Content{
		Parts:  []Part{NewTextPart("text")},
		Intent: Intent("CUSTOM_TASK"),
	}
	err := ValidateContentSupport(content, caps)
	require.NoError(t, err)
}

func TestValidateContentSupportDimensionPassThrough(t *testing.T) {
	caps := CapabilityMetadata{}
	dim := 128
	content := Content{
		Parts:     []Part{NewTextPart("text")},
		Dimension: &dim,
	}
	err := ValidateContentSupport(content, caps)
	require.NoError(t, err)
}

func TestValidateContentSupportFailOnFirst(t *testing.T) {
	caps := CapabilityMetadata{
		Modalities: []Modality{ModalityText},
		Intents:    []Intent{IntentRetrievalQuery},
	}
	content := Content{
		Parts:  []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromURL("https://example.com/a.wav"))},
		Intent: IntentClassification,
	}
	err := ValidateContentSupport(content, caps)
	requireValidationIssue(t, err, "parts[0].modality", validationCodeUnsupportedModality, "audio")
}

func TestValidateContentsSupportBatch(t *testing.T) {
	caps := CapabilityMetadata{Modalities: []Modality{ModalityText}}
	contents := []Content{
		{Parts: []Part{NewTextPart("first")}},
		{Parts: []Part{NewPartFromSource(ModalityAudio, NewBinarySourceFromURL("https://example.com/audio.wav"))}},
		{Parts: []Part{NewTextPart("third")}},
	}
	err := ValidateContentsSupport(contents, caps)
	requireValidationIssue(t, err, "contents[1].parts[0].modality", validationCodeUnsupportedModality, "audio")
}

func TestValidateContentsSupportEmptyBatch(t *testing.T) {
	caps := CapabilityMetadata{Modalities: []Modality{ModalityText}}
	err := ValidateContentsSupport([]Content{}, caps)
	require.NoError(t, err)
}
