// Package antigravity provides OAuth2 authentication functionality for the Antigravity provider.
package antigravity

import "os"

// OAuth client credentials and configuration.
// Override via the ANTIGRAVITY_CLIENT_ID and ANTIGRAVITY_CLIENT_SECRET environment variables.
var (
	ClientID     = envOrDefault("ANTIGRAVITY_CLIENT_ID", "ANTIGRAVITY_CLIENT_ID_PLACEHOLDER")
	ClientSecret = envOrDefault("ANTIGRAVITY_CLIENT_SECRET", "ANTIGRAVITY_CLIENT_SECRET_PLACEHOLDER")
	CallbackPort = 51121
)

// Scopes defines the OAuth scopes required for Antigravity authentication
var Scopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

// OAuth2 endpoints for Google authentication
const (
	TokenEndpoint    = "https://oauth2.googleapis.com/token"
	AuthEndpoint     = "https://accounts.google.com/o/oauth2/v2/auth"
	UserInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo?alt=json"
)

// Antigravity API configuration
const (
	APIEndpoint = "https://cloudcode-pa.googleapis.com"
	APIVersion  = "v1internal"
)

// envOrDefault returns the value of the environment variable named by key,
// or fallback if the variable is not set or empty.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
