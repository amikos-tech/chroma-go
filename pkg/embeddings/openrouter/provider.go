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

func (p *ProviderPreferences) UnmarshalJSON(data []byte) error {
	type Alias ProviderPreferences
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	extras := make(map[string]any)
	for key, value := range raw {
		if isProviderPreferenceField(key) {
			continue
		}
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return err
		}
		extras[key] = decoded
	}

	*p = ProviderPreferences(alias)
	if len(extras) > 0 {
		p.Extras = extras
	}
	return nil
}

// Keep this list in sync with typed ProviderPreferences fields so new keys do not leak into Extras.
func isProviderPreferenceField(key string) bool {
	switch key {
	case "allow_fallbacks",
		"require_parameters",
		"data_collection",
		"zdr",
		"enforce_distillable_text",
		"order",
		"only",
		"ignore",
		"quantizations",
		"sort",
		"max_price",
		"preferred_min_throughput",
		"preferred_max_latency":
		return true
	default:
		return false
	}
}

func (p *ProviderPreferences) ConfigMap() map[string]any {
	if p == nil {
		return nil
	}

	cfg := make(map[string]any)
	if p.AllowFallbacks != nil {
		cfg["allow_fallbacks"] = *p.AllowFallbacks
	}
	if p.RequireParameters != nil {
		cfg["require_parameters"] = *p.RequireParameters
	}
	if p.DataCollection != "" {
		cfg["data_collection"] = p.DataCollection
	}
	if p.ZDR != nil {
		cfg["zdr"] = *p.ZDR
	}
	if p.EnforceDistillableText != nil {
		cfg["enforce_distillable_text"] = *p.EnforceDistillableText
	}
	if len(p.Order) > 0 {
		cfg["order"] = p.Order
	}
	if len(p.Only) > 0 {
		cfg["only"] = p.Only
	}
	if len(p.Ignore) > 0 {
		cfg["ignore"] = p.Ignore
	}
	if len(p.Quantizations) > 0 {
		cfg["quantizations"] = p.Quantizations
	}
	if len(p.Sort) > 0 {
		cfg["sort"] = p.Sort
	}
	if len(p.MaxPrice) > 0 {
		cfg["max_price"] = p.MaxPrice
	}
	if p.PreferredMinThroughput != nil {
		cfg["preferred_min_throughput"] = p.PreferredMinThroughput
	}
	if p.PreferredMaxLatency != nil {
		cfg["preferred_max_latency"] = p.PreferredMaxLatency
	}
	for key, value := range p.Extras {
		if _, exists := cfg[key]; !exists {
			cfg[key] = value
		}
	}
	if len(cfg) == 0 {
		return nil
	}
	return cfg
}
