package embeddings

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// mimeTypeRe matches a basic RFC 2045 type/subtype MIME format.
//
//nolint:gocritic // regexp.Compile used intentionally to avoid panic per project guidelines.
var mimeTypeRe, _ = regexp.Compile(`^[a-zA-Z0-9][a-zA-Z0-9!#$&\-^_.+]*/[a-zA-Z0-9][a-zA-Z0-9!#$&\-^_.+]*$`)

func isValidMIMEType(mime string) bool {
	if mimeTypeRe != nil {
		return mimeTypeRe.MatchString(mime)
	}
	parts := strings.SplitN(mime, "/", 2)
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

const (
	validationCodeForbidden            = "forbidden"
	validationCodeInvalidValue         = "invalid_value"
	validationCodeMismatch             = "mismatch"
	validationCodeOneOf                = "one_of"
	validationCodeOutOfRange           = "out_of_range"
	validationCodeRequired             = "required"
	validationCodeUnsupportedModality  = "unsupported_modality"
	validationCodeUnsupportedIntent    = "unsupported_intent"
	validationCodeUnsupportedDimension = "unsupported_dimension"
)

// ValidationIssue describes one structural multimodal validation failure.
type ValidationIssue struct {
	Path    string
	Code    string
	Message string
}

// ValidationError collects one or more multimodal validation issues.
type ValidationError struct {
	Issues []ValidationIssue
}

// Error implements error.
func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "multimodal validation failed"
	}

	first := e.Issues[0]
	summary := first.Message
	if first.Path != "" {
		summary = fmt.Sprintf("%s: %s", first.Path, first.Message)
	}

	if len(e.Issues) == 1 {
		return fmt.Sprintf("multimodal validation failed: %s", summary)
	}

	return fmt.Sprintf(
		"multimodal validation failed: %s (and %d more issue(s))",
		summary,
		len(e.Issues)-1,
	)
}

// Validate checks that a content item is structurally valid.
func (c Content) Validate() error {
	validationErr := &ValidationError{}

	if len(c.Parts) == 0 {
		validationErr.addIssue("parts", validationCodeRequired, "content must include at least one part")
	}

	for i, part := range c.Parts {
		if err := part.Validate(); err != nil {
			validationErr.addIssues(prefixValidationIssues(fmt.Sprintf("parts[%d].", i), err)...)
		}
	}

	if c.Dimension != nil {
		switch {
		case *c.Dimension <= 0:
			validationErr.addIssue("dimension", validationCodeOutOfRange, "dimension must be greater than zero")
		case *c.Dimension > math.MaxInt32:
			validationErr.addIssue("dimension", validationCodeOutOfRange, "dimension must be less than or equal to math.MaxInt32")
		}
	}

	intentValue := string(c.Intent)
	if intentValue != "" {
		trimmed := strings.TrimSpace(intentValue)
		switch {
		case trimmed == "":
			validationErr.addIssue("intent", validationCodeInvalidValue, "intent cannot be whitespace only")
		case trimmed != intentValue:
			validationErr.addIssue("intent", validationCodeInvalidValue, "intent must not contain leading or trailing whitespace")
		}
	}

	return validationErr.orNil()
}

// Validate checks that a multimodal part is structurally valid.
func (p Part) Validate() error {
	validationErr := &ValidationError{}

	if p.Modality == "" {
		validationErr.addIssue("modality", validationCodeRequired, "modality is required")
		return validationErr.orNil()
	}

	switch p.Modality {
	case ModalityText:
		if p.Text == "" {
			validationErr.addIssue("text", validationCodeRequired, "text parts must include non-empty text")
		}
		if p.Source != nil {
			validationErr.addIssue("source", validationCodeForbidden, "text parts must not include a binary source")
		}
	case ModalityImage, ModalityAudio, ModalityVideo, ModalityPDF:
		if p.Text != "" {
			validationErr.addIssue("text", validationCodeForbidden, "non-text parts must not include text")
		}
		if p.Source == nil {
			validationErr.addIssue("source", validationCodeRequired, "non-text parts must include a binary source")
		} else if err := p.Source.Validate(); err != nil {
			validationErr.addIssues(prefixValidationIssues("source.", err)...)
		}
	default:
		validationErr.addIssue("modality", validationCodeInvalidValue, fmt.Sprintf("unsupported modality %q", p.Modality))
	}

	return validationErr.orNil()
}

// Validate checks that a binary source selects exactly one payload for the declared kind.
func (s BinarySource) Validate() error {
	validationErr := &ValidationError{}

	if !isKnownSourceKind(s.Kind) {
		validationErr.addIssue("kind", validationCodeInvalidValue, fmt.Sprintf("unsupported source kind %q", s.Kind))
	}

	payloadCount := 0
	if s.URL != "" {
		payloadCount++
	}
	if s.FilePath != "" {
		payloadCount++
	}
	if s.Base64 != "" {
		payloadCount++
	}
	if len(s.Bytes) > 0 {
		payloadCount++
	}

	switch payloadCount {
	case 0:
		validationErr.addIssue("payload", validationCodeRequired, "binary source must include exactly one payload")
	case 1:
	default:
		validationErr.addIssue("payload", validationCodeOneOf, "binary source must include exactly one payload")
	}

	switch s.Kind {
	case SourceKindURL:
		if s.URL == "" {
			validationErr.addIssue("kind", validationCodeMismatch, "source kind \"url\" requires the URL field")
		}
	case SourceKindFile:
		if s.FilePath == "" {
			validationErr.addIssue("kind", validationCodeMismatch, "source kind \"file\" requires the FilePath field")
		}
	case SourceKindBase64:
		if s.Base64 == "" {
			validationErr.addIssue("kind", validationCodeMismatch, "source kind \"base64\" requires the Base64 field")
		}
	case SourceKindBytes:
		if len(s.Bytes) == 0 {
			validationErr.addIssue("kind", validationCodeMismatch, "source kind \"bytes\" requires the Bytes field")
		}
	}

	if s.MIMEType != "" && !isValidMIMEType(s.MIMEType) {
		validationErr.addIssue("mime_type", validationCodeInvalidValue, fmt.Sprintf("MIME type %q is not a valid type/subtype format", s.MIMEType))
	}

	return validationErr.orNil()
}

// ValidateContents validates a batch of multimodal content items.
func ValidateContents(contents []Content) error {
	validationErr := &ValidationError{}

	if len(contents) == 0 {
		validationErr.addIssue("contents", validationCodeRequired, "at least one content item is required")
		return validationErr.orNil()
	}

	for i, content := range contents {
		if err := content.Validate(); err != nil {
			validationErr.addIssues(prefixValidationIssues(fmt.Sprintf("contents[%d].", i), err)...)
		}
	}

	return validationErr.orNil()
}

// ValidateContentSupport checks content against declared provider capabilities.
// Returns on the first unsupported check, consistent with batch fail-on-first behavior.
// Empty capability fields are treated as undeclared and pass through without validation.
func ValidateContentSupport(content Content, caps CapabilityMetadata) error {
	if len(caps.Modalities) > 0 {
		for i, part := range content.Parts {
			if !caps.SupportsModality(part.Modality) {
				return compatibilityError(
					fmt.Sprintf("parts[%d].modality", i),
					validationCodeUnsupportedModality,
					fmt.Sprintf("provider does not support %q modality", part.Modality),
				)
			}
		}
	}

	if content.Intent != "" && IsNeutralIntent(content.Intent) && len(caps.Intents) > 0 {
		if !caps.SupportsIntent(content.Intent) {
			return compatibilityError(
				"intent",
				validationCodeUnsupportedIntent,
				fmt.Sprintf("provider does not support %q intent", content.Intent),
			)
		}
	}

	if content.Dimension != nil && len(caps.RequestOptions) > 0 && !caps.SupportsRequestOption(RequestOptionDimension) {
		return compatibilityError(
			"dimension",
			validationCodeUnsupportedDimension,
			"provider does not support output dimension override",
		)
	}

	return nil
}

// ValidateContentsSupport validates a batch of content items against declared provider capabilities.
// Returns on the first unsupported item, consistent with existing batch validation behavior.
func ValidateContentsSupport(contents []Content, caps CapabilityMetadata) error {
	for i, content := range contents {
		if err := ValidateContentSupport(content, caps); err != nil {
			return prefixBatchCompatibilityError(i, err)
		}
	}
	return nil
}

func isKnownSourceKind(kind SourceKind) bool {
	switch kind {
	case SourceKindURL, SourceKindFile, SourceKindBase64, SourceKindBytes:
		return true
	default:
		return false
	}
}

func prefixValidationIssues(prefix string, err error) []ValidationIssue {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) && validationErr != nil {
		issues := make([]ValidationIssue, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			prefixed := issue
			if prefixed.Path == "" {
				prefixed.Path = strings.TrimSuffix(prefix, ".")
			} else {
				prefixed.Path = prefix + prefixed.Path
			}
			issues = append(issues, prefixed)
		}
		return issues
	}

	return []ValidationIssue{{
		Path:    strings.TrimSuffix(prefix, "."),
		Code:    validationCodeInvalidValue,
		Message: err.Error(),
	}}
}

func (e *ValidationError) addIssue(path, code, message string) {
	if e == nil {
		return
	}
	e.Issues = append(e.Issues, ValidationIssue{
		Path:    path,
		Code:    code,
		Message: message,
	})
}

func (e *ValidationError) addIssues(issues ...ValidationIssue) {
	if e == nil || len(issues) == 0 {
		return
	}
	e.Issues = append(e.Issues, issues...)
}

func (e *ValidationError) orNil() error {
	if e == nil || len(e.Issues) == 0 {
		return nil
	}
	return e
}
