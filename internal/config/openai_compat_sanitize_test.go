package config

import (
	"strings"
	"testing"
)

func TestSanitizeOpenAICompatibility_FlatFields(t *testing.T) {
	cfg := &Config{
		OpenAICompatibility: []OpenAICompatibility{
			{
				Name:    "  FlatProvider  ",
				BaseURL: "  https://flat.example.com  ",
				APIKey:  "  flat-key-123  ",
				ProxyURL: "  http://proxy:8080  ",
			},
			{
				// Whitespace-only api-key must round-trip without panicking;
				// base-url is present so the entry survives.
				Name:    "WhitespaceKey",
				BaseURL: "https://ws.example.com",
				APIKey:  "   ",
			},
			{
				// No base-url: dropped.
				Name:    "NoBase",
				APIKey:  "key",
			},
		},
	}

	cfg.SanitizeOpenAICompatibility()

	got := cfg.OpenAICompatibility
	if len(got) != 2 {
		t.Fatalf("expected 2 entries after sanitize, got %d: %+v", len(got), got)
	}
	if got[0].Name != "FlatProvider" {
		t.Errorf("name not trimmed: %q", got[0].Name)
	}
	if got[0].BaseURL != "https://flat.example.com" {
		t.Errorf("base-url not trimmed: %q", got[0].BaseURL)
	}
	if got[0].APIKey != "flat-key-123" {
		t.Errorf("api-key not trimmed: %q", got[0].APIKey)
	}
	if got[0].ProxyURL != "http://proxy:8080" {
		t.Errorf("proxy-url not trimmed: %q", got[0].ProxyURL)
	}
	if strings.TrimSpace(got[1].APIKey) != "" {
		t.Errorf("whitespace-only api-key should be empty after trim, got %q", got[1].APIKey)
	}
}
