package synthesizer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/watcher/diff"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// ConfigSynthesizer generates Auth entries from configuration API keys.
// It handles Gemini, Claude, Codex, OpenAI-compatible, and Vertex-compat providers.
type ConfigSynthesizer struct{}

// NewConfigSynthesizer creates a new ConfigSynthesizer instance.
func NewConfigSynthesizer() *ConfigSynthesizer {
	return &ConfigSynthesizer{}
}

// Synthesize generates Auth entries from config API keys.
func (s *ConfigSynthesizer) Synthesize(ctx *SynthesisContext) ([]*coreauth.Auth, error) {
	out := make([]*coreauth.Auth, 0, 32)
	if ctx == nil || ctx.Config == nil {
		return out, nil
	}

	// Gemini API Keys
	out = append(out, s.synthesizeGeminiKeys(ctx)...)
	// Claude API Keys
	out = append(out, s.synthesizeClaudeKeys(ctx)...)
	// Codex API Keys
	out = append(out, s.synthesizeCodexKeys(ctx)...)
	// OpenAI-compat
	out = append(out, s.synthesizeOpenAICompat(ctx)...)
	// Vertex-compat
	out = append(out, s.synthesizeVertexCompat(ctx)...)

	return out, nil
}

// expandAPIKeyEntries returns the effective per-key entries to synthesize for a
// provider item. Returns the configured APIKeyEntries as-is when present;
// otherwise expands the legacy flat APIKey/ProxyURL/Priority fields into a
// single synthetic entry so legacy configs keep working.
func expandAPIKeyEntries(entries []config.APIKeyEntry, flatKey, flatProxyURL string, flatPriority int) []config.APIKeyEntry {
	if len(entries) > 0 {
		return entries
	}
	if strings.TrimSpace(flatKey) == "" {
		return nil
	}
	return []config.APIKeyEntry{{APIKey: flatKey, ProxyURL: flatProxyURL, Priority: flatPriority}}
}

// ExpandOpenAICompatAPIKeyEntries mirrors ExpandAPIKeyEntries for the
// OpenAI-compatible provider: returns the configured entries as-is when
// present, otherwise expands the legacy flat APIKey/ProxyURL fields into a
// single synthetic entry so single-key configs keep working without the
// entries wrapper.
func ExpandOpenAICompatAPIKeyEntries(entries []config.OpenAICompatibilityAPIKey, flatKey, flatProxyURL string) []config.OpenAICompatibilityAPIKey {
	if len(entries) > 0 {
		return entries
	}
	if strings.TrimSpace(flatKey) == "" {
		return nil
	}
	return []config.OpenAICompatibilityAPIKey{{APIKey: flatKey, ProxyURL: flatProxyURL}}
}

// synthesizeGeminiKeys creates Auth entries for Gemini API keys.
func (s *ConfigSynthesizer) synthesizeGeminiKeys(ctx *SynthesisContext) []*coreauth.Auth {
	cfg := ctx.Config
	now := ctx.Now
	idGen := ctx.IDGenerator

	out := make([]*coreauth.Auth, 0, len(cfg.GeminiKey))
	for i := range cfg.GeminiKey {
		entry := cfg.GeminiKey[i]
		entries := expandAPIKeyEntries(entry.APIKeyEntries, entry.APIKey, entry.ProxyURL, entry.Priority)
		if len(entries) == 0 {
			continue
		}
		prefix := strings.TrimSpace(entry.Prefix)
		base := strings.TrimSpace(entry.BaseURL)
		modelsHash := diff.ComputeGeminiModelsHash(entry.Models)
		metadata := map[string]any{}
		if entry.DisableCooling {
			metadata["disable_cooling"] = true
		}
		for j := range entries {
			e := entries[j]
			key := strings.TrimSpace(e.APIKey)
			if key == "" {
				continue
			}
			proxyURL := strings.TrimSpace(e.ProxyURL)
			if proxyURL == "" {
				proxyURL = strings.TrimSpace(entry.ProxyURL)
			}
			priority := e.Priority
			if priority == 0 {
				priority = entry.Priority
			}
			id, token := idGen.Next("gemini:apikey", key, base)
			attrs := map[string]string{
				"source":  fmt.Sprintf("config:gemini[%s]", token),
				"api_key": key,
			}
			if priority != 0 {
				attrs["priority"] = strconv.Itoa(priority)
			}
			if base != "" {
				attrs["base_url"] = base
			}
			if modelsHash != "" {
				attrs["models_hash"] = modelsHash
			}
			addConfigHeadersToAttrs(entry.Headers, attrs)
			a := &coreauth.Auth{
				ID:         id,
				Provider:   "gemini",
				Label:      "gemini-apikey",
				Prefix:     prefix,
				Status:     coreauth.StatusActive,
				ProxyURL:   proxyURL,
				Attributes: attrs,
				Metadata:   metadata,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			ApplyAuthExcludedModelsMeta(a, cfg, entry.ExcludedModels, "apikey")
			if len(a.Metadata) == 0 {
				a.Metadata = nil
			}
			out = append(out, a)
		}
	}
	return out
}

// synthesizeClaudeKeys creates Auth entries for Claude API keys.
func (s *ConfigSynthesizer) synthesizeClaudeKeys(ctx *SynthesisContext) []*coreauth.Auth {
	cfg := ctx.Config
	now := ctx.Now
	idGen := ctx.IDGenerator

	out := make([]*coreauth.Auth, 0, len(cfg.ClaudeKey))
	for i := range cfg.ClaudeKey {
		ck := cfg.ClaudeKey[i]
		entries := expandAPIKeyEntries(ck.APIKeyEntries, ck.APIKey, ck.ProxyURL, ck.Priority)
		if len(entries) == 0 {
			continue
		}
		prefix := strings.TrimSpace(ck.Prefix)
		base := strings.TrimSpace(ck.BaseURL)
		modelsHash := diff.ComputeClaudeModelsHash(ck.Models)
		metadata := map[string]any{}
		if ck.DisableCooling {
			metadata["disable_cooling"] = true
		}
		for j := range entries {
			e := entries[j]
			key := strings.TrimSpace(e.APIKey)
			if key == "" {
				continue
			}
			proxyURL := strings.TrimSpace(e.ProxyURL)
			if proxyURL == "" {
				proxyURL = strings.TrimSpace(ck.ProxyURL)
			}
			priority := e.Priority
			if priority == 0 {
				priority = ck.Priority
			}
			id, token := idGen.Next("claude:apikey", key, base)
			attrs := map[string]string{
				"source":  fmt.Sprintf("config:claude[%s]", token),
				"api_key": key,
			}
			if priority != 0 {
				attrs["priority"] = strconv.Itoa(priority)
			}
			if base != "" {
				attrs["base_url"] = base
			}
			if modelsHash != "" {
				attrs["models_hash"] = modelsHash
			}
			addConfigHeadersToAttrs(ck.Headers, attrs)
			a := &coreauth.Auth{
				ID:         id,
				Provider:   "claude",
				Label:      "anthropic-apikey",
				Prefix:     prefix,
				Status:     coreauth.StatusActive,
				ProxyURL:   proxyURL,
				Attributes: attrs,
				Metadata:   metadata,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			ApplyAuthExcludedModelsMeta(a, cfg, ck.ExcludedModels, "apikey")
			if len(a.Metadata) == 0 {
				a.Metadata = nil
			}
			out = append(out, a)
		}
	}
	return out
}

// synthesizeCodexKeys creates Auth entries for Codex API keys.
func (s *ConfigSynthesizer) synthesizeCodexKeys(ctx *SynthesisContext) []*coreauth.Auth {
	cfg := ctx.Config
	now := ctx.Now
	idGen := ctx.IDGenerator

	out := make([]*coreauth.Auth, 0, len(cfg.CodexKey))
	for i := range cfg.CodexKey {
		ck := cfg.CodexKey[i]
		entries := expandAPIKeyEntries(ck.APIKeyEntries, ck.APIKey, ck.ProxyURL, ck.Priority)
		if len(entries) == 0 {
			continue
		}
		prefix := strings.TrimSpace(ck.Prefix)
		modelsHash := diff.ComputeCodexModelsHash(ck.Models)
		metadata := map[string]any{}
		if ck.DisableCooling {
			metadata["disable_cooling"] = true
		}
		for j := range entries {
			e := entries[j]
			key := strings.TrimSpace(e.APIKey)
			if key == "" {
				continue
			}
			proxyURL := strings.TrimSpace(e.ProxyURL)
			if proxyURL == "" {
				proxyURL = strings.TrimSpace(ck.ProxyURL)
			}
			priority := e.Priority
			if priority == 0 {
				priority = ck.Priority
			}
			id, token := idGen.Next("codex:apikey", key, ck.BaseURL)
			attrs := map[string]string{
				"source":  fmt.Sprintf("config:codex[%s]", token),
				"api_key": key,
			}
			if priority != 0 {
				attrs["priority"] = strconv.Itoa(priority)
			}
			if ck.BaseURL != "" {
				attrs["base_url"] = ck.BaseURL
			}
			if ck.Websockets {
				attrs["websockets"] = "true"
			}
			if modelsHash != "" {
				attrs["models_hash"] = modelsHash
			}
			addConfigHeadersToAttrs(ck.Headers, attrs)
			a := &coreauth.Auth{
				ID:         id,
				Provider:   "codex",
				Label:      "openai-responses-apikey",
				Prefix:     prefix,
				Status:     coreauth.StatusActive,
				ProxyURL:   proxyURL,
				Attributes: attrs,
				Metadata:   metadata,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			ApplyAuthExcludedModelsMeta(a, cfg, ck.ExcludedModels, "apikey")
			if len(a.Metadata) == 0 {
				a.Metadata = nil
			}
			out = append(out, a)
		}
	}
	return out
}

// synthesizeOpenAICompat creates Auth entries for OpenAI-compatible providers.
func (s *ConfigSynthesizer) synthesizeOpenAICompat(ctx *SynthesisContext) []*coreauth.Auth {
	cfg := ctx.Config
	now := ctx.Now
	idGen := ctx.IDGenerator

	out := make([]*coreauth.Auth, 0)
	for i := range cfg.OpenAICompatibility {
		compat := &cfg.OpenAICompatibility[i]
		if compat.Disabled {
			continue
		}
		prefix := strings.TrimSpace(compat.Prefix)
		providerName := strings.ToLower(strings.TrimSpace(compat.Name))
		if providerName == "" {
			providerName = "openai-compatible"
		}
		base := strings.TrimSpace(compat.BaseURL)
		disableCooling := compat.DisableCooling

		// Expand flat APIKey/ProxyURL into a single synthetic entry for the
		// single-key case; APIKeyEntries wins when both are set.
		entries := ExpandOpenAICompatAPIKeyEntries(compat.APIKeyEntries, compat.APIKey, compat.ProxyURL)
		createdEntries := 0
		for j := range entries {
			entry := &entries[j]
			key := strings.TrimSpace(entry.APIKey)
			proxyURL := strings.TrimSpace(entry.ProxyURL)
			idKind := fmt.Sprintf("openai-compatible:%s", providerName)
			id, token := idGen.Next(idKind, key, base, proxyURL)
			attrs := map[string]string{
				"source":       fmt.Sprintf("config:%s[%s]", providerName, token),
				"base_url":     base,
				"compat_name":  compat.Name,
				"provider_key": providerName,
			}
			metadata := map[string]any{}
			if disableCooling {
				metadata["disable_cooling"] = true
			}
			if compat.Priority != 0 {
				attrs["priority"] = strconv.Itoa(compat.Priority)
			}
			if key != "" {
				attrs["api_key"] = key
			}
			if hash := diff.ComputeOpenAICompatModelsHash(compat.Models); hash != "" {
				attrs["models_hash"] = hash
			}
			addConfigHeadersToAttrs(compat.Headers, attrs)
			a := &coreauth.Auth{
				ID:         id,
				Provider:   providerName,
				Label:      compat.Name,
				Prefix:     prefix,
				Status:     coreauth.StatusActive,
				ProxyURL:   proxyURL,
				Attributes: attrs,
				Metadata:   metadata,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if len(a.Metadata) == 0 {
				a.Metadata = nil
			}
			out = append(out, a)
			createdEntries++
		}
		// Fallback: create a keyless Auth when neither flat api-key nor entries
		// are configured (e.g. providers that authenticate via custom headers).
		if createdEntries == 0 {
			idKind := fmt.Sprintf("openai-compatible:%s", providerName)
			id, token := idGen.Next(idKind, base)
			attrs := map[string]string{
				"source":       fmt.Sprintf("config:%s[%s]", providerName, token),
				"base_url":     base,
				"compat_name":  compat.Name,
				"provider_key": providerName,
			}
			metadata := map[string]any{}
			if disableCooling {
				metadata["disable_cooling"] = true
			}
			if compat.Priority != 0 {
				attrs["priority"] = strconv.Itoa(compat.Priority)
			}
			if hash := diff.ComputeOpenAICompatModelsHash(compat.Models); hash != "" {
				attrs["models_hash"] = hash
			}
			addConfigHeadersToAttrs(compat.Headers, attrs)
			a := &coreauth.Auth{
				ID:         id,
				Provider:   providerName,
				Label:      compat.Name,
				Prefix:     prefix,
				Status:     coreauth.StatusActive,
				Attributes: attrs,
				Metadata:   metadata,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if len(a.Metadata) == 0 {
				a.Metadata = nil
			}
			out = append(out, a)
		}
	}
	return out
}

// synthesizeVertexCompat creates Auth entries for Vertex-compatible providers.
func (s *ConfigSynthesizer) synthesizeVertexCompat(ctx *SynthesisContext) []*coreauth.Auth {
	cfg := ctx.Config
	now := ctx.Now
	idGen := ctx.IDGenerator

	out := make([]*coreauth.Auth, 0, len(cfg.VertexCompatAPIKey))
	for i := range cfg.VertexCompatAPIKey {
		compat := cfg.VertexCompatAPIKey[i]
		// Vertex-compat historically creates an auth even when api-key is empty
		// (auth may be supplied via headers/cookies), so do not use the shared
		// expandAPIKeyEntries helper which filters empty keys.
		entries := compat.APIKeyEntries
		if len(entries) == 0 {
			entries = []config.APIKeyEntry{{APIKey: compat.APIKey, ProxyURL: compat.ProxyURL, Priority: compat.Priority}}
		}
		if len(entries) == 0 {
			continue
		}
		base := strings.TrimSpace(compat.BaseURL)
		prefix := strings.TrimSpace(compat.Prefix)
		modelsHash := diff.ComputeVertexCompatModelsHash(compat.Models)
		for j := range entries {
			e := entries[j]
			key := strings.TrimSpace(e.APIKey)
			proxyURL := strings.TrimSpace(e.ProxyURL)
			if proxyURL == "" {
				proxyURL = strings.TrimSpace(compat.ProxyURL)
			}
			priority := e.Priority
			if priority == 0 {
				priority = compat.Priority
			}
			id, token := idGen.Next("vertex:apikey", key, base, proxyURL)
			attrs := map[string]string{
				"source":       fmt.Sprintf("config:vertex-apikey[%s]", token),
				"base_url":     base,
				"provider_key": "vertex",
			}
			if priority != 0 {
				attrs["priority"] = strconv.Itoa(priority)
			}
			if key != "" {
				attrs["api_key"] = key
			}
			if modelsHash != "" {
				attrs["models_hash"] = modelsHash
			}
			addConfigHeadersToAttrs(compat.Headers, attrs)
			a := &coreauth.Auth{
				ID:         id,
				Provider:   "vertex",
				Label:      "vertex-apikey",
				Prefix:     prefix,
				Status:     coreauth.StatusActive,
				ProxyURL:   proxyURL,
				Attributes: attrs,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			ApplyAuthExcludedModelsMeta(a, cfg, compat.ExcludedModels, "apikey")
			out = append(out, a)
		}
	}
	return out
}
