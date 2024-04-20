package ollama

type Option func(p *OllamaClient) error

func WithBaseURL(baseURL string) Option {
	return func(p *OllamaClient) error {
		p.BaseURL = baseURL
		return nil
	}
}
func WithModel(model string) Option {
	return func(p *OllamaClient) error {
		p.Model = model
		return nil
	}
}
