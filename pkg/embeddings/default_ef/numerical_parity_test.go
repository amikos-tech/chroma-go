package defaultef

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"

	ort "github.com/amikos-tech/pure-onnx/ort"
	puretokenizers "github.com/amikos-tech/pure-tokenizers"
	"github.com/stretchr/testify/require"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	paritySequenceLength = 256
	parityEmbeddingDim   = 384
	poolingEpsilon       = float32(1e-9)
	normalizationEpsilon = float32(1e-12)

	legacyParityTolerance   = float32(1e-6)
	chromaPythonTolerance   = float32(1e-5)
	chromaPythonParityEnv   = "RUN_CHROMA_PYTHON_PARITY"
	chromaPythonParityValue = "1"
)

func TestDefaultEF_NumericalParityWithLegacyReferenceMath(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)

	documents := []string{
		"Hello Chroma!",
		"A second document with punctuation, numbers (123), and symbols #!?.",
		"Multiline input should stay stable.\nLine two has extra whitespace.",
		"",
	}

	ef, closeEF, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = closeEF()
	})

	gotEmbeddings, err := ef.EmbedDocuments(context.Background(), documents)
	require.NoError(t, err)
	got := embeddingsToFloat32Rows(gotEmbeddings)

	want, err := embedDocumentsWithLegacyReferenceMath(documents)
	require.NoError(t, err)

	maxDiff := maxAbsDiff2D(got, want)
	require.LessOrEqualf(
		t,
		maxDiff,
		legacyParityTolerance,
		"default_ef diverged from legacy reference math; max_abs_diff=%0.10f tolerance=%0.10f",
		maxDiff,
		legacyParityTolerance,
	)
}

func TestDefaultEF_NumericalParityWithChromaPythonReference(t *testing.T) {
	setOfflineRuntimePathOrSkip(t)

	if os.Getenv(chromaPythonParityEnv) != chromaPythonParityValue {
		t.Skipf("%s=%s is required for Chroma Python parity check", chromaPythonParityEnv, chromaPythonParityValue)
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skipf("python3 not found: %v", err)
	}

	documents := []string{
		"Hello Chroma!",
		"Cross-language parity should remain stable over time.",
		"Tokenizer + ONNX pooling parity check.",
	}

	ef, closeEF, err := NewDefaultEmbeddingFunction()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = closeEF()
	})
	gotEmbeddings, err := ef.EmbedDocuments(context.Background(), documents)
	require.NoError(t, err)
	got := embeddingsToFloat32Rows(gotEmbeddings)

	pythonRows, skipReason, err := embedWithChromaPython(documents)
	if skipReason != "" {
		t.Skip(skipReason)
	}
	require.NoError(t, err)

	maxDiff := maxAbsDiff2D(got, pythonRows)
	require.LessOrEqualf(
		t,
		maxDiff,
		chromaPythonTolerance,
		"default_ef diverged from Chroma Python reference; max_abs_diff=%0.10f tolerance=%0.10f",
		maxDiff,
		chromaPythonTolerance,
	)
}

func embedDocumentsWithLegacyReferenceMath(documents []string) ([][]float32, error) {
	cfg := getConfig()
	if err := EnsureOnnxRuntimeSharedLibrary(); err != nil {
		return nil, err
	}
	if err := EnsureDefaultEmbeddingFunctionModel(); err != nil {
		return nil, err
	}

	bootstrapOpts := []ort.BootstrapOption{
		ort.WithBootstrapCacheDir(cfg.OnnxCacheDir),
	}
	if cfg.LibOnnxRuntimeVersion == "custom" {
		bootstrapOpts = append(bootstrapOpts, ort.WithBootstrapLibraryPath(cfg.OnnxLibPath))
	} else {
		bootstrapOpts = append(bootstrapOpts, ort.WithBootstrapVersion(cfg.LibOnnxRuntimeVersion))
	}
	if err := ort.InitializeEnvironmentWithBootstrap(bootstrapOpts...); err != nil {
		return nil, err
	}
	defer func() {
		_ = ort.DestroyEnvironment()
	}()

	tokenizerOptions := []puretokenizers.TokenizerOption{
		puretokenizers.WithTruncation(
			paritySequenceLength,
			puretokenizers.TruncationDirectionRight,
			puretokenizers.TruncationStrategyLongestFirst,
		),
		puretokenizers.WithPadding(true, puretokenizers.PaddingStrategy{
			Tag:       puretokenizers.PaddingStrategyFixed,
			FixedSize: paritySequenceLength,
		}),
	}
	tokenizer, err := puretokenizers.FromFile(cfg.OnnxModelTokenizerConfigPath, tokenizerOptions...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tokenizer.Close()
	}()

	batchSize := len(documents)
	shape := ort.Shape{int64(batchSize), paritySequenceLength}
	outputShape := ort.Shape{int64(batchSize), paritySequenceLength, parityEmbeddingDim}

	inputIDs := make([]int64, batchSize*paritySequenceLength)
	attentionMask := make([]int64, batchSize*paritySequenceLength)
	typeIDs := make([]int64, batchSize*paritySequenceLength)

	for row, doc := range documents {
		encoding, err := tokenizer.Encode(
			doc,
			puretokenizers.WithAddSpecialTokens(),
			puretokenizers.WithReturnAttentionMask(),
			puretokenizers.WithReturnTypeIDs(),
		)
		if err != nil {
			return nil, err
		}
		rowStart := row * paritySequenceLength
		fillUint32AsInt64(inputIDs[rowStart:rowStart+paritySequenceLength], encoding.IDs)
		fillUint32AsInt64(attentionMask[rowStart:rowStart+paritySequenceLength], encoding.AttentionMask)
		fillUint32AsInt64(typeIDs[rowStart:rowStart+paritySequenceLength], encoding.TypeIDs)
	}

	inputTensor, err := ort.NewTensor[int64](shape, inputIDs)
	if err != nil {
		return nil, err
	}
	attentionTensor, err := ort.NewTensor[int64](shape, attentionMask)
	if err != nil {
		_ = inputTensor.Destroy()
		return nil, err
	}
	typeTensor, err := ort.NewTensor[int64](shape, typeIDs)
	if err != nil {
		_ = attentionTensor.Destroy()
		_ = inputTensor.Destroy()
		return nil, err
	}
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		_ = typeTensor.Destroy()
		_ = attentionTensor.Destroy()
		_ = inputTensor.Destroy()
		return nil, err
	}

	session, err := ort.NewAdvancedSession(
		cfg.OnnxModelPath,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		[]ort.Value{inputTensor, attentionTensor, typeTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		_ = outputTensor.Destroy()
		_ = typeTensor.Destroy()
		_ = attentionTensor.Destroy()
		_ = inputTensor.Destroy()
		return nil, err
	}
	defer func() {
		_ = session.Destroy()
		_ = outputTensor.Destroy()
		_ = typeTensor.Destroy()
		_ = attentionTensor.Destroy()
		_ = inputTensor.Destroy()
	}()

	if err := session.Run(); err != nil {
		return nil, err
	}

	return meanPoolAndNormalizeLegacy(outputTensor.GetData(), attentionMask, batchSize), nil
}

func meanPoolAndNormalizeLegacy(lastHiddenState []float32, attentionMask []int64, batchSize int) [][]float32 {
	embeddings := make([][]float32, batchSize)

	for row := 0; row < batchSize; row++ {
		embedding := make([]float32, parityEmbeddingDim)
		maskOffset := row * paritySequenceLength

		denominator := float32(0)
		for tokenIdx := 0; tokenIdx < paritySequenceLength; tokenIdx++ {
			mask := float32(attentionMask[maskOffset+tokenIdx])
			denominator += mask
			if mask == 0 {
				continue
			}

			hiddenOffset := (maskOffset + tokenIdx) * parityEmbeddingDim
			for d := 0; d < parityEmbeddingDim; d++ {
				embedding[d] += lastHiddenState[hiddenOffset+d] * mask
			}
		}

		if denominator < poolingEpsilon {
			denominator = poolingEpsilon
		}
		invDenominator := float32(1.0) / denominator
		for d := 0; d < parityEmbeddingDim; d++ {
			embedding[d] *= invDenominator
		}

		normSquared := float32(0)
		for _, v := range embedding {
			normSquared += v * v
		}
		norm := float32(math.Sqrt(float64(normSquared)))
		if norm < normalizationEpsilon {
			norm = normalizationEpsilon
		}
		invNorm := float32(1.0) / norm
		for d := 0; d < parityEmbeddingDim; d++ {
			embedding[d] *= invNorm
		}

		embeddings[row] = embedding
	}

	return embeddings
}

func fillUint32AsInt64(dst []int64, src []uint32) {
	if len(dst) == 0 || len(src) == 0 {
		return
	}
	copyCount := len(dst)
	if len(src) < copyCount {
		copyCount = len(src)
	}
	for i := 0; i < copyCount; i++ {
		dst[i] = int64(src[i])
	}
}

func embeddingsToFloat32Rows(embs []embeddings.Embedding) [][]float32 {
	rows := make([][]float32, len(embs))
	for i := range embs {
		content := embs[i].ContentAsFloat32()
		row := make([]float32, len(content))
		copy(row, content)
		rows[i] = row
	}
	return rows
}

func maxAbsDiff2D(a, b [][]float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}
	maxDiff := float32(0)
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return float32(math.Inf(1))
		}
		for j := range a[i] {
			diff := float32(math.Abs(float64(a[i][j] - b[i][j])))
			if diff > maxDiff {
				maxDiff = diff
			}
		}
	}
	return maxDiff
}

func embedWithChromaPython(documents []string) (_ [][]float32, skipReason string, err error) {
	inputJSON, err := json.Marshal(documents)
	if err != nil {
		return nil, "", err
	}

	const script = `
import json
import sys

docs = json.loads(sys.stdin.read())

try:
    from chromadb.utils.embedding_functions.onnx_mini_lm_l6_v2 import ONNXMiniLM_L6_V2
except Exception as exc:
    print(f"IMPORT_ERROR:{exc}", file=sys.stderr)
    raise

ef = ONNXMiniLM_L6_V2()
rows = ef(docs)
out = [[float(x) for x in row] for row in rows]
print(json.dumps(out))
`

	cmd := exec.Command("python3", "-c", script)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errText := stderr.String()
		if isPythonDependencyError(errText) {
			return nil, "skipping Chroma Python parity check: missing Python dependency (" + summarizePythonError(errText) + ")", nil
		}
		return nil, "", errors.New("python parity command failed: " + summarizePythonError(errText))
	}

	var rows [][]float32
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		return nil, "", err
	}
	return rows, "", nil
}

func isPythonDependencyError(stderr string) bool {
	lower := strings.ToLower(stderr)
	patterns := []string{
		"modulenotfounderror",
		"no module named",
		"import_error:",
		"the onnxruntime python package is not installed",
		"_array_api not found",
		"a module that was compiled using numpy 1.x cannot be run in",
	}
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func summarizePythonError(stderr string) string {
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return "unknown python error"
	}
	for _, line := range strings.Split(trimmed, "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" {
			continue
		}
		if len(candidate) > 220 {
			return candidate[:220] + "..."
		}
		return candidate
	}
	return "unknown python error"
}
