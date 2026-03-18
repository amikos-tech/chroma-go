package embeddings

// Modality identifies the content type for one multimodal part.
type Modality string

const (
	ModalityText  Modality = "text"
	ModalityImage Modality = "image"
	ModalityAudio Modality = "audio"
	ModalityVideo Modality = "video"
	ModalityPDF   Modality = "pdf"
)

// Intent describes the provider-neutral purpose of a multimodal request.
type Intent string

const (
	IntentRetrievalQuery     Intent = "retrieval_query"
	IntentRetrievalDocument  Intent = "retrieval_document"
	IntentClassification     Intent = "classification"
	IntentClustering         Intent = "clustering"
	IntentSemanticSimilarity Intent = "semantic_similarity"
)

// SourceKind identifies how a binary multimodal input is provided.
type SourceKind string

const (
	SourceKindURL    SourceKind = "url"
	SourceKindFile   SourceKind = "file"
	SourceKindBase64 SourceKind = "base64"
	SourceKindBytes  SourceKind = "bytes"
)

// BinarySource stores a single binary input reference for non-text modalities.
type BinarySource struct {
	Kind     SourceKind
	URL      string
	FilePath string
	Base64   string
	Bytes    []byte
	MIMEType string
}

// Part is one ordered multimodal unit inside a content request.
type Part struct {
	Modality Modality
	Text     string
	Source   *BinarySource
}

// Content is the canonical multimodal request item for one semantic unit.
type Content struct {
	Parts         []Part
	Intent        Intent
	Dimension     *int
	ProviderHints map[string]any
}
