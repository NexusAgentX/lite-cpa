package management

import (
	"fmt"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/watcher/synthesizer"
)

type apiKeyEntryWithAuthIndex struct {
	config.APIKeyEntry
	AuthIndex string `json:"auth-index,omitempty"`
}

type geminiKeyWithAuthIndex struct {
	config.GeminiKey
	// Shadows the embedded APIKeyEntries to attach a per-entry AuthIndex.
	APIKeyEntries []apiKeyEntryWithAuthIndex `json:"api-key-entries,omitempty"`
	AuthIndex     string                     `json:"auth-index,omitempty"`
}

type claudeKeyWithAuthIndex struct {
	config.ClaudeKey
	APIKeyEntries []apiKeyEntryWithAuthIndex `json:"api-key-entries,omitempty"`
	AuthIndex     string                     `json:"auth-index,omitempty"`
}

type codexKeyWithAuthIndex struct {
	config.CodexKey
	APIKeyEntries []apiKeyEntryWithAuthIndex `json:"api-key-entries,omitempty"`
	AuthIndex     string                     `json:"auth-index,omitempty"`
}

type vertexCompatKeyWithAuthIndex struct {
	config.VertexCompatKey
	APIKeyEntries []apiKeyEntryWithAuthIndex `json:"api-key-entries,omitempty"`
	AuthIndex     string                     `json:"auth-index,omitempty"`
}

type openAICompatibilityAPIKeyWithAuthIndex struct {
	config.OpenAICompatibilityAPIKey
	AuthIndex string `json:"auth-index,omitempty"`
}

type openAICompatibilityWithAuthIndex struct {
	Name          string                                   `json:"name"`
	Priority      int                                      `json:"priority,omitempty"`
	Disabled      bool                                     `json:"disabled"`
	Prefix        string                                   `json:"prefix,omitempty"`
	BaseURL       string                                   `json:"base-url"`
	APIKey        string                                   `json:"api-key,omitempty"`
	ProxyURL      string                                   `json:"proxy-url,omitempty"`
	APIKeyEntries []openAICompatibilityAPIKeyWithAuthIndex `json:"api-key-entries,omitempty"`
	Models        []config.OpenAICompatibilityModel        `json:"models,omitempty"`
	Headers       map[string]string                        `json:"headers,omitempty"`
	AuthIndex     string                                   `json:"auth-index,omitempty"`
}

func (h *Handler) liveAuthIndexByID() map[string]string {
	out := map[string]string{}
	if h == nil {
		return out
	}
	h.mu.Lock()
	manager := h.authManager
	h.mu.Unlock()
	if manager == nil {
		return out
	}
	// authManager.List() returns clones, so EnsureIndex only affects these copies.
	for _, auth := range manager.List() {
		if auth == nil {
			continue
		}
		id := strings.TrimSpace(auth.ID)
		if id == "" {
			continue
		}
		idx := strings.TrimSpace(auth.Index)
		if idx == "" {
			idx = auth.EnsureIndex()
		}
		if idx == "" {
			continue
		}
		out[id] = idx
	}
	return out
}

// expandEntriesForAuthIndex mirrors synthesizer.expandAPIKeyEntries: returns the
// configured entries as-is, or expands flat fields into a single entry for the
// legacy single-key form.
func expandEntriesForAuthIndex(entries []config.APIKeyEntry, flatKey, flatProxyURL string, flatPriority int) []config.APIKeyEntry {
	if len(entries) > 0 {
		return entries
	}
	if strings.TrimSpace(flatKey) == "" {
		return nil
	}
	return []config.APIKeyEntry{{APIKey: flatKey, ProxyURL: flatProxyURL, Priority: flatPriority}}
}

func (h *Handler) geminiKeysWithAuthIndex() []geminiKeyWithAuthIndex {
	if h == nil {
		return nil
	}
	liveIndexByID := h.liveAuthIndexByID()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg == nil {
		return nil
	}

	idGen := synthesizer.NewStableIDGenerator()
	out := make([]geminiKeyWithAuthIndex, len(h.cfg.GeminiKey))
	for i := range h.cfg.GeminiKey {
		entry := h.cfg.GeminiKey[i]
		entries := expandEntriesForAuthIndex(entry.APIKeyEntries, entry.APIKey, entry.ProxyURL, entry.Priority)
		wrapped := make([]apiKeyEntryWithAuthIndex, 0, len(entries))
		var flatIndex string
		for _, e := range entries {
			key := strings.TrimSpace(e.APIKey)
			var idx string
			if key != "" {
				id, _ := idGen.Next("gemini:apikey", key, entry.BaseURL)
				idx = liveIndexByID[id]
			}
			if flatIndex == "" {
				flatIndex = idx
			}
			wrapped = append(wrapped, apiKeyEntryWithAuthIndex{APIKeyEntry: e, AuthIndex: idx})
		}
		out[i] = geminiKeyWithAuthIndex{
			GeminiKey:     entry,
			APIKeyEntries: wrapped,
			AuthIndex:     flatIndex,
		}
	}
	return out
}

func (h *Handler) claudeKeysWithAuthIndex() []claudeKeyWithAuthIndex {
	if h == nil {
		return nil
	}
	liveIndexByID := h.liveAuthIndexByID()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg == nil {
		return nil
	}

	idGen := synthesizer.NewStableIDGenerator()
	out := make([]claudeKeyWithAuthIndex, len(h.cfg.ClaudeKey))
	for i := range h.cfg.ClaudeKey {
		entry := h.cfg.ClaudeKey[i]
		entries := expandEntriesForAuthIndex(entry.APIKeyEntries, entry.APIKey, entry.ProxyURL, entry.Priority)
		wrapped := make([]apiKeyEntryWithAuthIndex, 0, len(entries))
		var flatIndex string
		for _, e := range entries {
			key := strings.TrimSpace(e.APIKey)
			var idx string
			if key != "" {
				id, _ := idGen.Next("claude:apikey", key, entry.BaseURL)
				idx = liveIndexByID[id]
			}
			if flatIndex == "" {
				flatIndex = idx
			}
			wrapped = append(wrapped, apiKeyEntryWithAuthIndex{APIKeyEntry: e, AuthIndex: idx})
		}
		out[i] = claudeKeyWithAuthIndex{
			ClaudeKey:     entry,
			APIKeyEntries: wrapped,
			AuthIndex:     flatIndex,
		}
	}
	return out
}

func (h *Handler) codexKeysWithAuthIndex() []codexKeyWithAuthIndex {
	if h == nil {
		return nil
	}
	liveIndexByID := h.liveAuthIndexByID()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg == nil {
		return nil
	}

	idGen := synthesizer.NewStableIDGenerator()
	out := make([]codexKeyWithAuthIndex, len(h.cfg.CodexKey))
	for i := range h.cfg.CodexKey {
		entry := h.cfg.CodexKey[i]
		entries := expandEntriesForAuthIndex(entry.APIKeyEntries, entry.APIKey, entry.ProxyURL, entry.Priority)
		wrapped := make([]apiKeyEntryWithAuthIndex, 0, len(entries))
		var flatIndex string
		for _, e := range entries {
			key := strings.TrimSpace(e.APIKey)
			var idx string
			if key != "" {
				id, _ := idGen.Next("codex:apikey", key, entry.BaseURL)
				idx = liveIndexByID[id]
			}
			if flatIndex == "" {
				flatIndex = idx
			}
			wrapped = append(wrapped, apiKeyEntryWithAuthIndex{APIKeyEntry: e, AuthIndex: idx})
		}
		out[i] = codexKeyWithAuthIndex{
			CodexKey:      entry,
			APIKeyEntries: wrapped,
			AuthIndex:     flatIndex,
		}
	}
	return out
}

func (h *Handler) vertexCompatKeysWithAuthIndex() []vertexCompatKeyWithAuthIndex {
	if h == nil {
		return nil
	}
	liveIndexByID := h.liveAuthIndexByID()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg == nil {
		return nil
	}

	idGen := synthesizer.NewStableIDGenerator()
	out := make([]vertexCompatKeyWithAuthIndex, len(h.cfg.VertexCompatAPIKey))
	for i := range h.cfg.VertexCompatAPIKey {
		entry := h.cfg.VertexCompatAPIKey[i]
		// Vertex-compat preserves items with empty api-key (auth via headers).
		entries := entry.APIKeyEntries
		if len(entries) == 0 {
			entries = []config.APIKeyEntry{{APIKey: entry.APIKey, ProxyURL: entry.ProxyURL, Priority: entry.Priority}}
		}
		wrapped := make([]apiKeyEntryWithAuthIndex, 0, len(entries))
		var flatIndex string
		for _, e := range entries {
			id, _ := idGen.Next("vertex:apikey", e.APIKey, entry.BaseURL, e.ProxyURL)
			idx := liveIndexByID[id]
			if flatIndex == "" {
				flatIndex = idx
			}
			wrapped = append(wrapped, apiKeyEntryWithAuthIndex{APIKeyEntry: e, AuthIndex: idx})
		}
		out[i] = vertexCompatKeyWithAuthIndex{
			VertexCompatKey: entry,
			APIKeyEntries:   wrapped,
			AuthIndex:       flatIndex,
		}
	}
	return out
}

func (h *Handler) openAICompatibilityWithAuthIndex() []openAICompatibilityWithAuthIndex {
	if h == nil {
		return nil
	}
	liveIndexByID := h.liveAuthIndexByID()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg == nil {
		return nil
	}

	normalized := normalizedOpenAICompatibilityEntries(h.cfg.OpenAICompatibility)
	out := make([]openAICompatibilityWithAuthIndex, len(normalized))
	idGen := synthesizer.NewStableIDGenerator()
	for i := range normalized {
		entry := normalized[i]
		providerName := strings.ToLower(strings.TrimSpace(entry.Name))
		if providerName == "" {
			providerName = "openai-compatible"
		}
		idKind := fmt.Sprintf("openai-compatible:%s", providerName)

		response := openAICompatibilityWithAuthIndex{
			Name:      entry.Name,
			Priority:  entry.Priority,
			Disabled:  entry.Disabled,
			Prefix:    entry.Prefix,
			BaseURL:   entry.BaseURL,
			Models:    entry.Models,
			Headers:   entry.Headers,
			AuthIndex: "",
		}
		response.APIKey = entry.APIKey
		response.ProxyURL = entry.ProxyURL
		// Expand flat form so single-key configs surface the same AuthIndex
		// as multi-key entries. The synthesizer performs the same expansion.
		entries := synthesizer.ExpandOpenAICompatAPIKeyEntries(entry.APIKeyEntries, entry.APIKey, entry.ProxyURL)
		if len(entries) == 0 {
			id, _ := idGen.Next(idKind, entry.BaseURL)
			response.AuthIndex = liveIndexByID[id]
		} else {
			response.APIKeyEntries = make([]openAICompatibilityAPIKeyWithAuthIndex, len(entries))
			for j := range entries {
				apiKeyEntry := entries[j]
				id, _ := idGen.Next(idKind, apiKeyEntry.APIKey, entry.BaseURL, apiKeyEntry.ProxyURL)
				response.APIKeyEntries[j] = openAICompatibilityAPIKeyWithAuthIndex{
					OpenAICompatibilityAPIKey: apiKeyEntry,
					AuthIndex:                 liveIndexByID[id],
				}
			}
		}
		out[i] = response
	}
	return out
}
