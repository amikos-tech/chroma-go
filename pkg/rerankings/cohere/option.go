package cohere

import ccommons "github.com/amikos-tech/chroma-go/pkg/commons/cohere"

type Option func(p *CohereRerankingFunction) ccommons.Option

func WithBaseURL(baseURL string) Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		return ccommons.WithBaseURL(baseURL)
	}
}

func WithDefaultModel(model ccommons.CohereModel) Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		return ccommons.WithDefaultModel(model)
	}
}

func WithAPIKey(apiKey string) Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		return ccommons.WithAPIKey(apiKey)
	}
}

// WithEnvAPIKey configures the client to use the COHERE_API_KEY environment variable as the API key
func WithEnvAPIKey() Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		return ccommons.WithEnvAPIKey()
	}
}

func WithTopN(topN int) Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		p.TopN = topN
		return ccommons.NoOp()
	}
}

// WithRerankFields configures the client to use the specified fields for reranking if the documents are in JSON format
func WithRerankFields(fields []string) Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		p.RerankFields = fields
		return ccommons.NoOp()
	}
}

// WithReturnDocuments configures the client to return the original documents in the response
func WithReturnDocuments() Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		p.ReturnDocuments = true
		return ccommons.NoOp()
	}
}

func WithMaxChunksPerDoc(maxChunks int) Option {
	return func(p *CohereRerankingFunction) ccommons.Option {
		p.MaxChunksPerDoc = maxChunks
		return ccommons.NoOp()
	}
}
