package roboflow

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

const (
	DefaultBaseURL     = "https://infer.roboflow.com"
	APIKeyEnvVar       = "ROBOFLOW_API_KEY"
	DefaultHTTPTimeout = 60 * time.Second
)

type textEmbeddingRequest struct {
	Text string `json:"text"`
}

type imageEmbeddingRequest struct {
	Image imageData `json:"image"`
}

type imageData struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type embeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

var (
	_ embeddings.EmbeddingFunction           = (*RoboflowEmbeddingFunction)(nil)
	_ embeddings.MultimodalEmbeddingFunction = (*RoboflowEmbeddingFunction)(nil)
)

func getDefaults() *RoboflowEmbeddingFunction {
	return &RoboflowEmbeddingFunction{
		httpClient: &http.Client{Timeout: DefaultHTTPTimeout},
		baseURL:    DefaultBaseURL,
	}
}

type RoboflowEmbeddingFunction struct {
	httpClient   *http.Client
	APIKey       embeddings.Secret `json:"-" validate:"required"`
	apiKeyEnvVar string
	baseURL      string
	insecure     bool
}

func validate(ef *RoboflowEmbeddingFunction) error {
	if err := embeddings.NewValidator().Struct(ef); err != nil {
		return err
	}
	parsed, err := url.Parse(ef.baseURL)
	if err != nil {
		return errors.Wrap(err, "invalid base URL")
	}
	if !ef.insecure && !strings.EqualFold(parsed.Scheme, "https") {
		return errors.New("base URL must use HTTPS scheme for secure API key transmission; use WithInsecure() to override")
	}
	return nil
}

func NewRoboflowEmbeddingFunction(opts ...Option) (*RoboflowEmbeddingFunction, error) {
	ef := getDefaults()
	for _, opt := range opts {
		if err := opt(ef); err != nil {
			return nil, err
		}
	}
	if err := validate(ef); err != nil {
		return nil, errors.Wrap(err, "failed to validate Roboflow embedding function options")
	}
	return ef, nil
}

func (e *RoboflowEmbeddingFunction) sendTextRequest(ctx context.Context, text string) (*embeddingResponse, error) {
	endpoint := e.baseURL + "/clip/embed_text?api_key=" + e.APIKey.Value()

	payload, err := json.Marshal(textEmbeddingRequest{Text: text})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal text embedding request")
	}

	return e.doRequest(ctx, endpoint, payload)
}

func (e *RoboflowEmbeddingFunction) sendImageRequest(ctx context.Context, base64Image string) (*embeddingResponse, error) {
	endpoint := e.baseURL + "/clip/embed_image?api_key=" + e.APIKey.Value()

	payload, err := json.Marshal(imageEmbeddingRequest{
		Image: imageData{
			Type:  "base64",
			Value: base64Image,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal image embedding request")
	}

	return e.doRequest(ctx, endpoint, payload)
}

func (e *RoboflowEmbeddingFunction) doRequest(ctx context.Context, endpoint string, payload []byte) (*embeddingResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create embedding request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send embedding request")
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected response %v: %s", resp.Status, string(respData))
	}

	var response embeddingResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal embedding response")
	}

	return &response, nil
}

func (e *RoboflowEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	if len(documents) == 0 {
		return nil, nil
	}

	result := make([]embeddings.Embedding, len(documents))
	for i, doc := range documents {
		emb, err := e.EmbedQuery(ctx, doc)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to embed document %d", i)
		}
		result[i] = emb
	}
	return result, nil
}

func (e *RoboflowEmbeddingFunction) EmbedQuery(ctx context.Context, text string) (embeddings.Embedding, error) {
	response, err := e.sendTextRequest(ctx, text)
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed text")
	}
	if len(response.Embeddings) == 0 {
		return nil, errors.New("empty embedding response from Roboflow API")
	}
	return embeddings.NewEmbeddingFromFloat32(response.Embeddings[0]), nil
}

func (e *RoboflowEmbeddingFunction) EmbedImages(ctx context.Context, images []embeddings.ImageInput) ([]embeddings.Embedding, error) {
	if len(images) == 0 {
		return nil, nil
	}

	result := make([]embeddings.Embedding, len(images))
	for i, img := range images {
		emb, err := e.EmbedImage(ctx, img)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to embed image %d", i)
		}
		result[i] = emb
	}
	return result, nil
}

func (e *RoboflowEmbeddingFunction) EmbedImage(ctx context.Context, image embeddings.ImageInput) (embeddings.Embedding, error) {
	base64Data, err := image.ToBase64(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert image to base64")
	}

	response, err := e.sendImageRequest(ctx, base64Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to embed image")
	}
	if len(response.Embeddings) == 0 {
		return nil, errors.New("empty embedding response from Roboflow API")
	}
	return embeddings.NewEmbeddingFromFloat32(response.Embeddings[0]), nil
}

func (e *RoboflowEmbeddingFunction) Name() string {
	return "roboflow"
}

func (e *RoboflowEmbeddingFunction) GetConfig() embeddings.EmbeddingFunctionConfig {
	envVar := e.apiKeyEnvVar
	if envVar == "" {
		envVar = APIKeyEnvVar
	}
	apiURL := e.baseURL
	if apiURL == "" {
		apiURL = DefaultBaseURL
	}
	cfg := embeddings.EmbeddingFunctionConfig{
		"api_key_env_var": envVar,
		"api_url":         apiURL,
	}
	if e.insecure {
		cfg["insecure"] = true
	}
	return cfg
}

func (e *RoboflowEmbeddingFunction) DefaultSpace() embeddings.DistanceMetric {
	return embeddings.COSINE
}

func (e *RoboflowEmbeddingFunction) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{embeddings.COSINE, embeddings.L2, embeddings.IP}
}

// NewRoboflowEmbeddingFunctionFromConfig creates a Roboflow embedding function from a config map.
// Uses schema-compliant field names: api_key_env_var, api_url, insecure.
func NewRoboflowEmbeddingFunctionFromConfig(cfg embeddings.EmbeddingFunctionConfig) (*RoboflowEmbeddingFunction, error) {
	envVar, ok := cfg["api_key_env_var"].(string)
	if !ok || envVar == "" {
		return nil, errors.New("api_key_env_var is required in config")
	}
	apiURL, ok := cfg["api_url"].(string)
	if !ok || apiURL == "" {
		return nil, errors.New("api_url is required in config")
	}
	opts := []Option{WithAPIKeyFromEnvVar(envVar), WithBaseURL(apiURL)}
	if insecure, ok := cfg["insecure"].(bool); ok && insecure {
		opts = append(opts, WithInsecure())
	} else if embeddings.AllowInsecureFromEnv() {
		embeddings.LogInsecureEnvVarWarning("Roboflow")
		opts = append(opts, WithInsecure())
	}
	return NewRoboflowEmbeddingFunction(opts...)
}

func init() {
	if err := embeddings.RegisterDense("roboflow", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.EmbeddingFunction, error) {
		return NewRoboflowEmbeddingFunctionFromConfig(cfg)
	}); err != nil {
		panic(err)
	}

	if err := embeddings.RegisterMultimodal("roboflow", func(cfg embeddings.EmbeddingFunctionConfig) (embeddings.MultimodalEmbeddingFunction, error) {
		return NewRoboflowEmbeddingFunctionFromConfig(cfg)
	}); err != nil {
		panic(err)
	}
}
