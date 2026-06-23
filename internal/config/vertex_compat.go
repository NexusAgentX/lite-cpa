package config

import "strings"

// VertexCompatKey represents the configuration for Vertex AI-compatible API keys.
// This supports third-party services that use Vertex AI-style endpoint paths
// (/publishers/google/models/{model}:streamGenerateContent) but authenticate
// with simple API keys instead of Google Cloud service account credentials.
//
// Example services: zenmux.ai and similar Vertex-compatible providers.
type VertexCompatKey struct {
	// APIKey is the authentication key for accessing the Vertex-compatible API.
	// Maps to the x-goog-api-key header.
	APIKey string `yaml:"api-key" json:"api-key"`

	// APIKeyEntries allows multiple keys to share this entry's config.
	// Preferred over the flat APIKey/ProxyURL/Priority fields; when non-empty the synthesizer iterates these.
	APIKeyEntries []APIKeyEntry `yaml:"api-key-entries,omitempty" json:"api-key-entries,omitempty"`

	// Priority controls selection preference when multiple credentials match.
	// Higher values are preferred; defaults to 0.
	Priority int `yaml:"priority,omitempty" json:"priority,omitempty"`

	// Prefix optionally namespaces model aliases for this credential (e.g., "teamA/vertex-pro").
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`

	// BaseURL optionally overrides the Vertex-compatible API endpoint.
	// The executor will append "/v1/publishers/google/models/{model}:action" to this.
	// When empty, requests fall back to the default Vertex API base URL.
	BaseURL string `yaml:"base-url,omitempty" json:"base-url,omitempty"`

	// ProxyURL optionally overrides the global proxy for this API key.
	ProxyURL string `yaml:"proxy-url,omitempty" json:"proxy-url,omitempty"`

	// Headers optionally adds extra HTTP headers for requests sent with this key.
	// Commonly used for cookies, user-agent, and other authentication headers.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// Models defines the model configurations including aliases for routing.
	Models []VertexCompatModel `yaml:"models,omitempty" json:"models,omitempty"`

	// ExcludedModels lists model IDs that should be excluded for this provider.
	ExcludedModels []string `yaml:"excluded-models,omitempty" json:"excluded-models,omitempty"`
}

func (k VertexCompatKey) GetAPIKey() string  { return k.APIKey }
func (k VertexCompatKey) GetBaseURL() string { return k.BaseURL }

// VertexCompatModel represents a model configuration for Vertex compatibility,
// including the actual model name and its alias for API routing.
type VertexCompatModel struct {
	// Name is the actual model name used by the external provider.
	Name string `yaml:"name" json:"name"`

	// Alias is the model name alias that clients will use to reference this model.
	Alias string `yaml:"alias" json:"alias"`
}

func (m VertexCompatModel) GetName() string  { return m.Name }
func (m VertexCompatModel) GetAlias() string { return m.Alias }

// sanitizeAPIKeyEntries trims, drops empty, and dedupes api-key-entries by api-key.
// Returns nil when the result is empty so omitempty yaml/json tags kick in.
func sanitizeAPIKeyEntries(entries []APIKeyEntry) []APIKeyEntry {
	if len(entries) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(entries))
	out := entries[:0]
	for i := range entries {
		e := entries[i]
		e.APIKey = strings.TrimSpace(e.APIKey)
		if e.APIKey == "" {
			continue
		}
		e.ProxyURL = strings.TrimSpace(e.ProxyURL)
		if _, exists := seen[e.APIKey]; exists {
			continue
		}
		seen[e.APIKey] = struct{}{}
		out = append(out, e)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SanitizeVertexCompatKeys deduplicates and normalizes Vertex-compatible API key credentials.
func (cfg *Config) SanitizeVertexCompatKeys() {
	if cfg == nil {
		return
	}

	seen := make(map[string]struct{}, len(cfg.VertexCompatAPIKey))
	out := cfg.VertexCompatAPIKey[:0]
	for i := range cfg.VertexCompatAPIKey {
		entry := cfg.VertexCompatAPIKey[i]
		entry.APIKey = strings.TrimSpace(entry.APIKey)
		entry.APIKeyEntries = sanitizeAPIKeyEntries(entry.APIKeyEntries)
		// Drop only when both legacy flat key and entries are empty.
		if entry.APIKey == "" && len(entry.APIKeyEntries) == 0 {
			continue
		}
		entry.Prefix = normalizeModelPrefix(entry.Prefix)
		entry.BaseURL = strings.TrimSpace(entry.BaseURL)
		entry.ProxyURL = strings.TrimSpace(entry.ProxyURL)
		entry.Headers = NormalizeHeaders(entry.Headers)
		entry.ExcludedModels = NormalizeExcludedModels(entry.ExcludedModels)

		// Sanitize models: remove entries without valid alias
		sanitizedModels := make([]VertexCompatModel, 0, len(entry.Models))
		for _, model := range entry.Models {
			model.Alias = strings.TrimSpace(model.Alias)
			model.Name = strings.TrimSpace(model.Name)
			if model.Alias != "" && model.Name != "" {
				sanitizedModels = append(sanitizedModels, model)
			}
		}
		entry.Models = sanitizedModels

		// Dedupe by flat api-key + base-url. Items using api-key-entries are preserved
		// as-is to honor their per-key configuration.
		if entry.APIKey != "" {
			uniqueKey := entry.APIKey + "|" + entry.BaseURL
			if _, exists := seen[uniqueKey]; exists {
				continue
			}
			seen[uniqueKey] = struct{}{}
		}
		out = append(out, entry)
	}
	cfg.VertexCompatAPIKey = out
}
