package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

type GeminiModelsSummary struct {
	hash  string
	count int
}

type ClaudeModelsSummary struct {
	hash  string
	count int
}

type CodexModelsSummary struct {
	hash  string
	count int
}

type VertexModelsSummary struct {
	hash  string
	count int
}

// SummarizeGeminiModels hashes Gemini model aliases for change detection.
func SummarizeGeminiModels(models []config.GeminiModel) GeminiModelsSummary {
	if len(models) == 0 {
		return GeminiModelsSummary{}
	}
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			name := strings.TrimSpace(model.Name)
			alias := strings.TrimSpace(model.Alias)
			if name == "" && alias == "" {
				continue
			}
			out(strings.ToLower(name) + "|" + strings.ToLower(alias))
		}
	})
	return GeminiModelsSummary{
		hash:  hashJoined(keys),
		count: len(keys),
	}
}

// SummarizeClaudeModels hashes Claude model aliases for change detection.
func SummarizeClaudeModels(models []config.ClaudeModel) ClaudeModelsSummary {
	if len(models) == 0 {
		return ClaudeModelsSummary{}
	}
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			name := strings.TrimSpace(model.Name)
			alias := strings.TrimSpace(model.Alias)
			if name == "" && alias == "" {
				continue
			}
			out(strings.ToLower(name) + "|" + strings.ToLower(alias))
		}
	})
	return ClaudeModelsSummary{
		hash:  hashJoined(keys),
		count: len(keys),
	}
}

// SummarizeCodexModels hashes Codex model aliases for change detection.
func SummarizeCodexModels(models []config.CodexModel) CodexModelsSummary {
	if len(models) == 0 {
		return CodexModelsSummary{}
	}
	keys := normalizeModelPairs(func(out func(key string)) {
		for _, model := range models {
			name := strings.TrimSpace(model.Name)
			alias := strings.TrimSpace(model.Alias)
			if name == "" && alias == "" {
				continue
			}
			out(strings.ToLower(name) + "|" + strings.ToLower(alias))
		}
	})
	return CodexModelsSummary{
		hash:  hashJoined(keys),
		count: len(keys),
	}
}

// SummarizeVertexModels hashes Vertex-compatible model aliases for change detection.
func SummarizeVertexModels(models []config.VertexCompatModel) VertexModelsSummary {
	if len(models) == 0 {
		return VertexModelsSummary{}
	}
	names := make([]string, 0, len(models))
	for _, model := range models {
		name := strings.TrimSpace(model.Name)
		alias := strings.TrimSpace(model.Alias)
		if name == "" && alias == "" {
			continue
		}
		if alias != "" {
			name = alias
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return VertexModelsSummary{}
	}
	sort.Strings(names)
	sum := sha256.Sum256([]byte(strings.Join(names, "|")))
	return VertexModelsSummary{
		hash:  hex.EncodeToString(sum[:]),
		count: len(names),
	}
}

// APIKeyEntriesSummary captures a stable hash and count of api-key-entries for change detection.
type APIKeyEntriesSummary struct {
	hash  string
	count int
}

// SummarizeAPIKeyEntries hashes a config-backed provider's APIKeyEntries (api-key, proxy-url, priority).
// Keys are normalized and sorted so reordering alone is not reported as a change.
func SummarizeAPIKeyEntries(entries []config.APIKeyEntry) APIKeyEntriesSummary {
	if len(entries) == 0 {
		return APIKeyEntriesSummary{}
	}
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		apiKey := strings.TrimSpace(e.APIKey)
		if apiKey == "" {
			continue
		}
		keys = append(keys, strings.ToLower(apiKey)+"|"+strings.TrimSpace(e.ProxyURL)+"|"+strconv.Itoa(e.Priority))
	}
	if len(keys) == 0 {
		return APIKeyEntriesSummary{}
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return APIKeyEntriesSummary{
		hash:  hex.EncodeToString(sum[:]),
		count: len(keys),
	}
}
