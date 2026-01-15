package v2

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid"
)

type GenerateOptions struct {
	Document string
}

type IDGeneratorOption func(opts *GenerateOptions)

func WithDocument(document string) IDGeneratorOption {
	return func(opts *GenerateOptions) {
		opts.Document = document
	}
}

type IDGenerator interface {
	Generate(opts ...IDGeneratorOption) string
}

type UUIDGenerator struct{}

func (u *UUIDGenerator) Generate(opts ...IDGeneratorOption) string {
	// uuid.New() uses uuid.Must(uuid.NewRandom()) internally which could panic
	// if the random number generator fails. Add defensive programming.

	// Wrap in a function to handle panics
	generateID := func() (id string) {
		defer func() {
			if r := recover(); r != nil {
				// Fall back to a cryptographically secure random ID if UUID generation fails
				// This is extremely unlikely but ensures the library never panics
				h := sha256.New()
				h.Write([]byte(time.Now().String()))
				// Use crypto/rand for secure random bytes
				randomBytes := make([]byte, 16)
				if _, err := io.ReadFull(crand.Reader, randomBytes); err != nil {
					// If even crypto/rand fails, use timestamp with process info as last resort
					h.Write([]byte(time.Now().Format(time.RFC3339Nano)))
				} else {
					h.Write(randomBytes)
				}
				id = hex.EncodeToString(h.Sum(nil))
			}
		}()

		uuidV4 := uuid.New()
		id = uuidV4.String()
		return id
	}

	return generateID()
}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

type SHA256Generator struct{}

func (s *SHA256Generator) Generate(opts ...IDGeneratorOption) string {
	op := GenerateOptions{}
	for _, opt := range opts {
		opt(&op)
	}
	if op.Document == "" {
		// Generate random input for hashing with panic recovery
		op.Document = func() (doc string) {
			defer func() {
				if r := recover(); r != nil {
					// Fall back to crypto/rand if uuid.New() panics
					randomBytes := make([]byte, 16)
					if _, err := io.ReadFull(crand.Reader, randomBytes); err != nil {
						// Last resort: use timestamp
						doc = time.Now().Format(time.RFC3339Nano)
					} else {
						doc = hex.EncodeToString(randomBytes)
					}
				}
			}()
			doc = uuid.New().String()
			return doc
		}()
	}
	hasher := sha256.New()
	hasher.Write([]byte(op.Document))
	sha256Hash := hex.EncodeToString(hasher.Sum(nil))
	return sha256Hash
}

func NewSHA256Generator() *SHA256Generator {
	return &SHA256Generator{}
}

type ULIDGenerator struct{}

func (u *ULIDGenerator) Generate(opts ...IDGeneratorOption) string {
	// Wrap in a function to handle panics properly
	generateID := func() (id string) {
		defer func() {
			if r := recover(); r != nil {
				// If ULID generation fails, fall back to UUID
				// This ensures the function never panics
				id = uuid.New().String()
			}
		}()

		t := time.Now()
		// Use crypto/rand for secure entropy
		entropy := ulid.Monotonic(crand.Reader, 0)

		// Try using ulid.New() first for safer operation
		docULID, err := ulid.New(ulid.Timestamp(t), entropy)
		if err != nil {
			// Fall back to UUID if ULID generation fails
			id = uuid.New().String()
			return id
		}
		id = docULID.String()
		return id
	}

	return generateID()
}

func NewULIDGenerator() *ULIDGenerator {
	return &ULIDGenerator{}
}
