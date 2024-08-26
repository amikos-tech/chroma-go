package tokenizers

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include "tokenizers.h"

typedef void* (*from_file_func)(const char*);
void* call_from_file(void* f, const char* path) {
    from_file_func func = (from_file_func)f;
    return func(path);
}

typedef void* (*from_bytes_func)(const uint8_t *, uint32_t , const struct TokenizerOptions *);
void* call_from_bytes(void* f,const uint8_t *config, uint32_t len, const struct TokenizerOptions *options) {
    from_bytes_func func = (from_bytes_func)f;
    return func(config,len,options);
}

typedef struct Buffer (*encode_func)(void*, const char*, const struct EncodeOptions*);
struct Buffer call_encode(void* f, void* tokenizer, const char* str, const struct EncodeOptions* opts) {
	encode_func func = (encode_func)f;
	return func(tokenizer, str, opts);
}

typedef void (*free_tokenizer_func)(void*);
void freeTokenizer(void* f, void* tokenizer) {
	free_tokenizer_func func = (free_tokenizer_func)f;
	func(tokenizer);
}

void freeBuffer(struct Buffer *buffer) {
    free(buffer->ids);
    free(buffer->type_ids);
    free(buffer->special_tokens_mask);
    free(buffer->attention_mask);
    free(buffer->tokens);
    free(buffer->offsets);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

var (
	libHandle unsafe.Pointer

	fromBytesFunc               unsafe.Pointer
	fromBytesWithTruncationFunc unsafe.Pointer
	fromFileFunc                unsafe.Pointer
	freeTokenizerFunc           unsafe.Pointer
	encodeFunc                  unsafe.Pointer
	decodeFunc                  unsafe.Pointer
	freeBufferFunc              unsafe.Pointer
	freeStringFunc              unsafe.Pointer
	vocabSizeFunc               unsafe.Pointer
)

func loadSymbol(name string) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	symbol := C.dlsym(libHandle, cName)
	if symbol == nil {
		C.dlclose(libHandle)
		msg := C.GoString(C.dlerror())
		return fmt.Errorf("failed to load %s: %s", name, msg)
	}

	switch name {
	case "from_file":
		fromFileFunc = symbol
	case "from_bytes":
		fromBytesFunc = symbol
	case "from_bytes_with_truncation":
		fromBytesWithTruncationFunc = symbol
	case "free_tokenizer":
		freeTokenizerFunc = symbol
	case "encode":
		encodeFunc = symbol
	case "decode":
		decodeFunc = symbol
	case "free_buffer":
		freeBufferFunc = symbol
	case "free_string":
		freeStringFunc = symbol
	case "vocab_size":
		vocabSizeFunc = symbol
	default:
		return fmt.Errorf("unknown symbol: %s", name)
	}

	return nil
}

// LoadLibrary loads the Tokenizer shared library at the specified path. This must be called first before using any other functions.
func LoadLibrary(path string) error {
	cName := C.CString(path)
	defer C.free(unsafe.Pointer(cName))
	libHandle = C.dlopen(cName, C.RTLD_LAZY)
	if libHandle == nil {
		msg := C.GoString(C.dlerror())
		return fmt.Errorf("error loading tokenizers shared library \"%s\": %s",
			path, msg)
	}

	var funcNames = []string{
		"from_file",
		"from_bytes",
		"from_bytes_with_truncation",
		"free_tokenizer",
		"encode",
		"decode",
		"free_buffer",
		"free_string",
		"vocab_size",
	}

	cFunctionName := C.CString("from_file")
	defer C.free(unsafe.Pointer(cFunctionName))

	for _, name := range funcNames {
		err := loadSymbol(name)
		if err != nil {
			return fmt.Errorf("failed to load from_file symbol: %v", err)
		}
	}
	return nil
}

func (t *Tokenizer) Close() error {
	C.freeTokenizer(freeTokenizerFunc, t.tokenizer)
	t.tokenizer = nil
	C.dlclose(libHandle)
	libHandle = nil
	return nil
}

type Tokenizer struct {
	tokenizer unsafe.Pointer
}

type tokenizerOpts struct {
	encodeSpecialTokens C.bool
}

type TokenizerOption func(to *tokenizerOpts)

func WithEncodeSpecialTokens() TokenizerOption {
	return func(to *tokenizerOpts) {
		to.encodeSpecialTokens = C.bool(true)
	}
}

type TruncationDirection int

const (
	TruncationDirectionLeft TruncationDirection = iota
	TruncationDirectionRight
)

func FromBytes(data []byte, opts ...TokenizerOption) (*Tokenizer, error) {
	allOpts := &tokenizerOpts{
		encodeSpecialTokens: false,
	}
	for _, opt := range opts {
		opt(allOpts)
	}
	dlen := C.uint(len(data))
	cData := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	cOpts := C.struct_TokenizerOptions{
		encode_special_tokens: allOpts.encodeSpecialTokens,
	}
	cOptsPtr := (*C.struct_TokenizerOptions)(unsafe.Pointer(&cOpts))
	tokenizerPtr := C.call_from_bytes(fromBytesFunc, cData, dlen, cOptsPtr)
	return &Tokenizer{tokenizer: tokenizerPtr}, nil
}

func FromBytesWithTruncation(data []byte, maxLen uint32, dir TruncationDirection) (*Tokenizer, error) {
	if fromBytesWithTruncationFunc == nil {
		return nil, fmt.Errorf("library not loaded or fromBytesWithTruncation function not found")
	}

	fromBytesTruncFn := (*[0]byte)(fromBytesWithTruncationFunc)
	tokenizer := unsafe.Pointer(uintptr((*(*func(data *byte, len C.uint, maxLen C.uint, dir C.uchar) unsafe.Pointer)(unsafe.Pointer(&fromBytesTruncFn)))(
		(*byte)(unsafe.Pointer(&data[0])),
		C.uint(len(data)),
		C.uint(maxLen),
		C.uchar(dir),
	)))

	if tokenizer == nil {
		return nil, fmt.Errorf("failed to create tokenizer")
	}

	return &Tokenizer{tokenizer: tokenizer}, nil
}

func FromFile(path string) (*Tokenizer, error) {
	if fromFileFunc == nil {
		return nil, fmt.Errorf("from_file function not loaded")
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// Define the function type corresponding to the C function signature

	// Convert the unsafe.Pointer to a function pointer

	// Call the function
	tokenizerPtr := C.call_from_file(fromFileFunc, cPath)
	if tokenizerPtr == nil {
		return nil, fmt.Errorf("failed to create tokenizer from file: %s", path)
	}

	return &Tokenizer{tokenizer: tokenizerPtr}, nil
}

type Offset [2]uint

type Encoding struct {
	IDs               []uint32
	TypeIDs           []uint32
	SpecialTokensMask []uint32
	AttentionMask     []uint32
	Tokens            []string
	Offsets           []Offset
}

type encodeOpts struct {
	AddSpecialTokens C.bool

	ReturnTypeIDs           C.bool
	ReturnTokens            C.bool
	ReturnSpecialTokensMask C.bool
	ReturnAttentionMask     C.bool
	ReturnOffsets           C.bool
}

type EncodeOption func(eo *encodeOpts)

func uintVecToSlice(arrPtr *C.uint, len int) []uint32 {
	arr := unsafe.Slice(arrPtr, len)
	slice := make([]uint32, len)
	for i, v := range arr {
		slice[i] = uint32(v)
	}
	return slice
}

func offsetVecToSlice(arrPtr *C.size_t, tokenLength int) []Offset {
	arr := unsafe.Slice(arrPtr, tokenLength*2)
	slice := make([]Offset, tokenLength)
	counter := 0
	for i := 0; i < tokenLength; i++ {
		offset := Offset{uint(arr[counter]), uint(arr[counter+1])}
		slice[i] = offset
		counter += 2
	}
	return slice
}

// EncodeResult represents the C struct returned by the encode function
type EncodeResult struct {
	ids               *uint32
	typeIds           *uint32
	tokens            **C.char
	specialTokensMask *uint32
	attentionMask     *uint32
	offsets           *uintptr
	len               uint32
}

func (t *Tokenizer) Encode(str string, addSpecialTokens bool) ([]uint32, []string, error) {
	if encodeFunc == nil || freeBufferFunc == nil {
		return nil, nil, fmt.Errorf("library not loaded or encode/free_buffer functions not found")
	}

	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))

	options := encodeOpts{
		AddSpecialTokens: C.bool(addSpecialTokens),
		ReturnTokens:     C.bool(true),
	}

	encodeFn := (*[0]byte)(encodeFunc)
	result := (*(*func(unsafe.Pointer, *C.char, *encodeOpts) EncodeResult)(unsafe.Pointer(&encodeFn)))(
		t.tokenizer,
		cStr,
		(*encodeOpts)(unsafe.Pointer(&options)),
	)

	if result.len == 0 {
		return nil, nil, nil
	}

	defer func() {
		freeBufferFn := (*[0]byte)(freeBufferFunc)
		(*(*func(*EncodeResult))(unsafe.Pointer(&freeBufferFn)))(&result)
	}()

	len := int(result.len)
	ids := make([]uint32, len)
	for i := 0; i < len; i++ {
		ids[i] = *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(result.ids)) + uintptr(i)*unsafe.Sizeof(uint32(0))))
	}

	var tokens []string
	if result.tokens != nil {
		tokens = make([]string, len)
		for i := 0; i < len; i++ {
			cStr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(result.tokens)) + uintptr(i)*unsafe.Sizeof((*C.char)(nil))))
			tokens[i] = C.GoString(cStr)
		}
	}

	return ids, tokens, nil
}

func WithReturnAllAttributes() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnTypeIDs = C.bool(true)
		eo.ReturnSpecialTokensMask = C.bool(true)
		eo.ReturnAttentionMask = C.bool(true)
		eo.ReturnTokens = C.bool(true)
		eo.ReturnOffsets = C.bool(true)
	}
}

func WithReturnTypeIDs() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnTypeIDs = C.bool(true)
	}
}

func WithReturnSpecialTokensMask() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnSpecialTokensMask = C.bool(true)
	}
}

func WithReturnTokens() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnTokens = C.bool(true)
	}
}

func WithReturnAttentionMask() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnAttentionMask = C.bool(true)
	}
}

func WithReturnOffsets() EncodeOption {
	return func(eo *encodeOpts) {
		eo.ReturnOffsets = C.bool(true)
	}
}

func (t *Tokenizer) EncodeWithOptions(str string, addSpecialTokens bool, opts ...EncodeOption) (Encoding, error) {
	if encodeFunc == nil || freeBufferFunc == nil {
		return Encoding{}, fmt.Errorf("library not loaded or encode/free_buffer functions not found")
	}

	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))

	encOptions := encodeOpts{
		AddSpecialTokens: C.bool(addSpecialTokens),
	}
	for _, opt := range opts {
		opt(&encOptions)
	}

	res := C.call_encode(encodeFunc, t.tokenizer, cStr, (*C.struct_EncodeOptions)(unsafe.Pointer(&encOptions)))
	resLen := int(res.len)
	if resLen == 0 {
		return Encoding{}, nil
	}
	//nolint:gocritic
	defer C.freeBuffer(&res)

	encoding := Encoding{}
	encoding.IDs = uintVecToSlice(res.ids, resLen)

	if encOptions.ReturnTypeIDs && res.type_ids != nil {
		encoding.TypeIDs = uintVecToSlice(res.type_ids, resLen)
	}

	if encOptions.ReturnTokens && res.tokens != nil {
		tokens := make([]string, resLen)
		for i, s := range (*[1 << 30]*C.char)(unsafe.Pointer(res.tokens))[:resLen:resLen] {
			tokens[i] = C.GoString(s)
		}
		encoding.Tokens = tokens
	}

	if encOptions.ReturnSpecialTokensMask && res.special_tokens_mask != nil {
		encoding.SpecialTokensMask = uintVecToSlice(res.special_tokens_mask, resLen)
	}

	if encOptions.ReturnAttentionMask && res.attention_mask != nil {
		encoding.AttentionMask = uintVecToSlice(res.attention_mask, resLen)
	}

	if encOptions.ReturnOffsets && res.offsets != nil {
		encoding.Offsets = offsetVecToSlice(res.offsets, resLen)
	}

	return encoding, nil
}

func (t *Tokenizer) Decode(tokenIDs []uint32, skipSpecialTokens bool) (string, error) {
	if decodeFunc == nil || freeStringFunc == nil {
		return "", fmt.Errorf("library not loaded or decode/free_string functions not found")
	}

	if len(tokenIDs) == 0 {
		return "", nil
	}

	decodeFn := (*[0]byte)(decodeFunc)
	result := (*(*func(unsafe.Pointer, *uint32, C.uint, C.bool) *C.char)(unsafe.Pointer(&decodeFn)))(
		t.tokenizer,
		&tokenIDs[0],
		C.uint(len(tokenIDs)),
		C.bool(skipSpecialTokens),
	)

	if result == nil {
		return "", fmt.Errorf("decode failed")
	}

	defer func() {
		freeStringFn := (*[0]byte)(freeStringFunc)
		(*(*func(*C.char))(unsafe.Pointer(&freeStringFn)))(result)
	}()

	return C.GoString(result), nil
}

// Helper functions

func makeUint32Slice(ptr *uint32, len int) []uint32 {
	slice := make([]uint32, len)
	for i := 0; i < len; i++ {
		slice[i] = *(*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(i)*unsafe.Sizeof(uint32(0))))
	}
	return slice
}

func makeStringSlice(ptr **C.char, len int) []string {
	slice := make([]string, len)
	for i := 0; i < len; i++ {
		cStr := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(i)*unsafe.Sizeof((*C.char)(nil))))
		slice[i] = C.GoString(cStr)
	}
	return slice
}

func makeOffsetSlice(ptr *uintptr, len int) []Offset {
	slice := make([]Offset, len)
	for i := 0; i < len; i++ {
		start := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(i*2)*unsafe.Sizeof(uintptr(0))))
		end := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(i*2+1)*unsafe.Sizeof(uintptr(0))))
		slice[i] = Offset{uint(start), uint(end)}
	}
	return slice
}

func (t *Tokenizer) VocabSize() (uint32, error) {
	if vocabSizeFunc == nil {
		return 0, fmt.Errorf("library not loaded or vocab_size function not found")
	}
	vocabSizeFn := (*[0]byte)(vocabSizeFunc)
	result := (*(*func(unsafe.Pointer) C.uint)(unsafe.Pointer(&vocabSizeFn)))(t.tokenizer)

	return uint32(result), nil
}
