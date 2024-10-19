package defaultef

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"sync/atomic"

	ort "github.com/yalue/onnxruntime_go"

	tokenizers "github.com/amikos-tech/chroma-go/pkg/tokenizers/libtokenizers"
	"github.com/amikos-tech/chroma-go/types"
)

type Option func(p *DefaultEmbeddingFunction) error

var _ types.EmbeddingFunction = (*DefaultEmbeddingFunction)(nil)

type DefaultEmbeddingFunction struct {
	tokenizer *tokenizers.Tokenizer
	closed    int32
	closeOnce sync.Once
}

var (
	initLock sync.Mutex
)

func NewDefaultEmbeddingFunction(opts ...Option) (*DefaultEmbeddingFunction, func() error, error) {
	initLock.Lock()
	defer initLock.Unlock()
	err := EnsureLibTokenizersSharedLibrary()
	if err != nil {
		return nil, nil, err
	}
	err = EnsureOnnxRuntimeSharedLibrary()
	if err != nil {
		return nil, nil, err
	}
	err = EnsureDefaultEmbeddingFunctionModel()
	if err != nil {
		return nil, nil, err
	}
	err = tokenizers.LoadLibrary(libTokenizersLibPath)
	if err != nil {
		return nil, nil, err
	}
	updatedConfigBytes, err := updateConfig(onnxModelTokenizerConfigPath)
	if err != nil {
		return nil, nil, err
	}
	tk, err := tokenizers.FromBytes(updatedConfigBytes)
	if err != nil {
		return nil, nil, err
	}
	ef := &DefaultEmbeddingFunction{tokenizer: tk}
	if !ort.IsInitialized() {
		ort.SetSharedLibraryPath(onnxLibPath)
		err = ort.InitializeEnvironment()
		if err != nil {
			errc := ef.Close()
			if errc != nil {
				fmt.Printf("error while closing embedding function %v", errc.Error())
			}
			return nil, nil, err
		}
	}

	return ef, ef.Close, nil
}

type EmbeddingInput struct {
	shape           *ort.Shape
	inputTensor     *ort.Tensor[int64]
	attentionTensor *ort.Tensor[int64]
	typeIDSTensor   *ort.Tensor[int64]
}

func NewEmbeddingInput(inputIDs []int64, attnMask []int64, typeIDs []int64, numInputs, vlen int64) (*EmbeddingInput, error) {
	inputShape := ort.NewShape(numInputs, vlen)
	inputTensor, err := ort.NewTensor(inputShape, inputIDs)
	if err != nil {
		return nil, err
	}
	attentionTensor, err := ort.NewTensor(inputShape, attnMask)
	if err != nil {
		derr := inputTensor.Destroy()
		if derr != nil {
			fmt.Printf("potential memory leak. Failed to destroy input tensor %e", derr)
		}
		return nil, err
	}
	typeTensor, err := ort.NewTensor(inputShape, typeIDs)
	if err != nil {
		derr := inputTensor.Destroy()
		if derr != nil {
			fmt.Printf("potential memory leak. Failed to destroy input tensor %e", derr)
		}
		derr = attentionTensor.Destroy()
		if derr != nil {
			fmt.Printf("potential memory leak. Failed to destroy attention tensor %e", derr)
		}
		return nil, err
	}
	return &EmbeddingInput{
		shape:           &inputShape,
		inputTensor:     inputTensor,
		attentionTensor: attentionTensor,
		typeIDSTensor:   typeTensor,
	}, nil
}

func (ei *EmbeddingInput) Close() error {
	var errOut []error
	err1 := ei.inputTensor.Destroy()
	if err1 != nil {
		errOut = append(errOut, err1)
	}
	err2 := ei.attentionTensor.Destroy()
	if err2 != nil {
		errOut = append(errOut, err2)
	}

	err3 := ei.typeIDSTensor.Destroy()
	if err3 != nil {
		errOut = append(errOut, err3)
	}
	if len(errOut) > 0 {
		return fmt.Errorf("errors: %v", errOut)
	}
	return nil
}

func (e *DefaultEmbeddingFunction) tokenize(documents []string) (*EmbeddingInput, error) {
	var tensorSize int64 = 0
	var numInputs = int64(len(documents))
	var vlen int64 = 0
	inputIDs := make([]int64, tensorSize)
	attnMask := make([]int64, tensorSize)
	typeIDs := make([]int64, tensorSize)
	for _, doc := range documents {
		res1, err := e.tokenizer.EncodeWithOptions(doc, true, tokenizers.WithReturnAttentionMask(), tokenizers.WithReturnTypeIDs())
		if err != nil {
			return nil, err
		}
		for i := range res1.IDs {
			inputIDs = append(inputIDs, int64(res1.IDs[i]))
			attnMask = append(attnMask, int64(res1.AttentionMask[i]))
			typeIDs = append(typeIDs, int64(res1.TypeIDs[i]))
		}
		vlen = int64(math.Max(float64(vlen), float64(len(res1.IDs))))
		tensorSize += int64(len(res1.IDs))
	}
	return NewEmbeddingInput(inputIDs, attnMask, typeIDs, numInputs, vlen)
}

func (e *DefaultEmbeddingFunction) encode(embeddingInput *EmbeddingInput) ([]*types.Embedding, error) {
	outputShape := ort.NewShape(append(*embeddingInput.shape, 384)...)
	shapeInt32 := make([]int, len(outputShape))

	for i, v := range outputShape {
		shapeInt32[i] = int(v)
	}
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, err
	}
	defer func(outputTensor *ort.Tensor[float32]) {
		err := outputTensor.Destroy()
		if err != nil {
			fmt.Printf("potential memory leak. Failed to destory outputTensor %v", err)
		}
	}(outputTensor)
	session, err := ort.NewAdvancedSession(onnxModelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"}, []string{"last_hidden_state"},
		[]ort.Value{embeddingInput.inputTensor, embeddingInput.attentionTensor, embeddingInput.typeIDSTensor}, []ort.Value{outputTensor}, nil)
	if err != nil {
		return nil, err
	}
	defer func(session *ort.AdvancedSession) {
		err := session.Destroy()
		if err != nil {
			fmt.Printf("potential memory leak. Failed to destory ORT session %v", err)
		}
	}(session)

	err = session.Run()
	if err != nil {
		return nil, err
	}
	outputData := outputTensor.GetData()
	t, err := ReshapeFlattenedTensor(outputData, shapeInt32)
	if err != nil {
		return nil, err
	}

	expandedMask := BroadcastTo(ExpandDims(embeddingInput.attentionTensor.GetData(), *embeddingInput.shape), [3]int(shapeInt32))
	mtpl, err := multiply(t.(Tensor3D[float32]), expandedMask)
	if err != nil {
		return nil, err
	}

	summed, err := mtpl.Sum(1)
	if err != nil {
		return nil, err
	}
	summedExpandedMask, err := expandedMask.Sum(1)
	if err != nil {
		return nil, err
	}
	summedExpandedMaskF32 := ConvertTensor2D[int64, float32](summedExpandedMask)
	clippedSummed := clip(summedExpandedMaskF32, 1e-9, math.MaxFloat32)
	embeddings := divide(summed, clippedSummed)
	normalizedEmbeddings := normalize(embeddings)
	out := types.NewEmbeddingsFromFloat32(normalizedEmbeddings)
	return out, nil
}

func (e *DefaultEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]*types.Embedding, error) {
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil, fmt.Errorf("embedding function is closed")
	}
	embeddingInputs, err := e.tokenize(documents)
	if err != nil {
		return nil, err
	}
	return e.encode(embeddingInputs)
}

func (e *DefaultEmbeddingFunction) EmbedQuery(ctx context.Context, document string) (*types.Embedding, error) {
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil, fmt.Errorf("embedding function is closed")
	}
	embeddingInputs, err := e.tokenize([]string{document})
	if err != nil {
		return nil, err
	}
	embeddings, err := e.encode(embeddingInputs)
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func (e *DefaultEmbeddingFunction) EmbedRecords(ctx context.Context, records []*types.Record, force bool) error {
	if atomic.LoadInt32(&e.closed) == 1 {
		return fmt.Errorf("embedding function is closed")
	}
	return types.EmbedRecordsDefaultImpl(e, ctx, records, force)
}

func updateConfig(filename string) ([]byte, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Unmarshal JSON into a map
	var jsonMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	// Update truncation.max_length
	if truncation, ok := jsonMap["truncation"]; ok {
		var truncationMap map[string]interface{}
		if err := json.Unmarshal(truncation, &truncationMap); err != nil {
			return nil, fmt.Errorf("error unmarshaling truncation: %v", err)
		}
		truncationMap["max_length"] = 256
		updatedTruncation, err := json.Marshal(truncationMap)
		if err != nil {
			return nil, fmt.Errorf("error marshaling updated truncation: %v", err)
		}
		jsonMap["truncation"] = updatedTruncation
	}

	// Update padding.strategy.Fixed
	if padding, ok := jsonMap["padding"]; ok {
		var paddingMap map[string]json.RawMessage
		if err := json.Unmarshal(padding, &paddingMap); err != nil {
			return nil, fmt.Errorf("error unmarshaling padding: %v", err)
		}
		if strategy, ok := paddingMap["strategy"]; ok {
			var strategyMap map[string]int
			if err := json.Unmarshal(strategy, &strategyMap); err != nil {
				return nil, fmt.Errorf("error unmarshaling strategy: %v", err)
			}
			strategyMap["Fixed"] = 256
			updatedStrategy, err := json.Marshal(strategyMap)
			if err != nil {
				return nil, fmt.Errorf("error marshaling updated strategy: %v", err)
			}
			paddingMap["strategy"] = updatedStrategy
		}
		updatedPadding, err := json.Marshal(paddingMap)
		if err != nil {
			return nil, fmt.Errorf("error marshaling updated padding: %v", err)
		}
		jsonMap["padding"] = updatedPadding
	}

	// Marshal the updated map back to JSON
	updatedData, err := json.MarshalIndent(jsonMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling updated JSON: %v", err)
	}

	return updatedData, nil
}

func (e *DefaultEmbeddingFunction) Close() error {
	if atomic.LoadInt32(&e.closed) == 1 {
		return nil
	}
	var closeErr error
	e.closeOnce.Do(func() {
		var errs []error
		if e.tokenizer != nil {
			err := e.tokenizer.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
		if ort.IsInitialized() { // skip destroying the environment if it is not initialized
			err := ort.DestroyEnvironment()
			if err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			closeErr = fmt.Errorf("errors: %v", errs)
		}
		atomic.StoreInt32(&e.closed, 1)
	})
	return closeErr
}
