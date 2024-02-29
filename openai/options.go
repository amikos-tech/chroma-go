package openai

// Option is a function type that can be used to modify the client.
type Option func(c *OpenAIClient)

// WithOpenAIOrganizationID is an option for setting the OpenAI org id.
func WithOpenAIOrganizationID(openAiAPIKey string) Option {
	return func(c *OpenAIClient) {
		c.SetOrgID(openAiAPIKey)
	}
}

func applyClientOptions(c *OpenAIClient, opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
