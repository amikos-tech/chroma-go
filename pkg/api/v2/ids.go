package v2

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
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
				// Fall back to a timestamp-based ID if UUID generation somehow fails
				// This is extremely unlikely but ensures the library never panics
				h := sha256.New()
				h.Write([]byte(time.Now().String()))
				h.Write([]byte(string(rune(rand.Int()))))
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
		op.Document = uuid.New().String()
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
		entropy := rand.New(rand.NewSource(t.UnixNano()))

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
