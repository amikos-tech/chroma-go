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
	uuidV4 := uuid.New()
	return uuidV4.String()
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
	t := time.Now()
	entropy := rand.New(rand.NewSource(t.UnixNano()))
	docULID := ulid.MustNew(ulid.Timestamp(t), entropy)
	return docULID.String()
}

func NewULIDGenerator() *ULIDGenerator {
	return &ULIDGenerator{}
}
