package embeddings

// RequestOption identifies a shared request-time option that a provider can support.
type RequestOption string

const (
	// RequestOptionDimension indicates support for caller-provided output dimensions.
	RequestOptionDimension RequestOption = "dimension"
	// RequestOptionProviderHints indicates support for provider-specific request hints.
	RequestOptionProviderHints RequestOption = "provider_hints"
)

// CapabilityMetadata describes the shared capability surface exposed by an embedding provider.
type CapabilityMetadata struct {
	Modalities     []Modality
	Intents        []Intent
	RequestOptions []RequestOption

	SupportsBatch     bool
	SupportsMixedPart bool
}

// SupportsModality reports whether the capability metadata includes the given modality.
func (m CapabilityMetadata) SupportsModality(modality Modality) bool {
	for _, supported := range m.Modalities {
		if supported == modality {
			return true
		}
	}
	return false
}

// SupportsIntent reports whether the capability metadata includes the given intent.
func (m CapabilityMetadata) SupportsIntent(intent Intent) bool {
	for _, supported := range m.Intents {
		if supported == intent {
			return true
		}
	}
	return false
}

// SupportsRequestOption reports whether the capability metadata includes the given request option.
func (m CapabilityMetadata) SupportsRequestOption(option RequestOption) bool {
	for _, supported := range m.RequestOptions {
		if supported == option {
			return true
		}
	}
	return false
}
