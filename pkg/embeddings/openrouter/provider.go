package openrouter

import "encoding/json"

// ProviderPreferences controls OpenRouter provider routing behavior.
type ProviderPreferences struct {
	AllowFallbacks         *bool          `json:"allow_fallbacks,omitempty"`
	RequireParameters      *bool          `json:"require_parameters,omitempty"`
	DataCollection         string         `json:"data_collection,omitempty"`
	ZDR                    *bool          `json:"zdr,omitempty"`
	EnforceDistillableText *bool          `json:"enforce_distillable_text,omitempty"`
	Order                  []string       `json:"order,omitempty"`
	Only                   []string       `json:"only,omitempty"`
	Ignore                 []string       `json:"ignore,omitempty"`
	Quantizations          []string       `json:"quantizations,omitempty"`
	Sort                   map[string]any `json:"sort,omitempty"`
	MaxPrice               map[string]any `json:"max_price,omitempty"`
	PreferredMinThroughput any            `json:"preferred_min_throughput,omitempty"`
	PreferredMaxLatency    any            `json:"preferred_max_latency,omitempty"`
	Extras                 map[string]any `json:"-"`
}

func (p ProviderPreferences) MarshalJSON() ([]byte, error) {
	type Alias ProviderPreferences
	data, err := json.Marshal(Alias(p))
	if err != nil {
		return nil, err
	}
	if len(p.Extras) == 0 {
		return data, nil
	}
	var merged map[string]any
	if err := json.Unmarshal(data, &merged); err != nil {
		return nil, err
	}
	for k, v := range p.Extras {
		if _, exists := merged[k]; !exists {
			merged[k] = v
		}
	}
	return json.Marshal(merged)
}
