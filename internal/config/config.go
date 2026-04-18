package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the OpenClaw gateway connection settings.
type Config struct {
	GatewayURL string
	WSURL      string
	Token      string
}

// Load reads configuration from environment variables (and .env file if present).
func Load() (*Config, error) {
	_ = godotenv.Load() // silently ignore missing .env

	gatewayURL := os.Getenv("OPENCLAW_GATEWAY_URL")
	if gatewayURL == "" {
		return nil, fmt.Errorf("OPENCLAW_GATEWAY_URL is required")
	}

	token := os.Getenv("OPENCLAW_GATEWAY_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("OPENCLAW_GATEWAY_TOKEN is required")
	}

	wsURL, err := deriveWSURL(gatewayURL)
	if err != nil {
		return nil, fmt.Errorf("invalid OPENCLAW_GATEWAY_URL: %w", err)
	}

	return &Config{
		GatewayURL: gatewayURL,
		WSURL:      wsURL,
		Token:      token,
	}, nil
}

func deriveWSURL(gatewayURL string) (string, error) {
	u, err := url.Parse(gatewayURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "wss", "ws":
		// already correct
	default:
		return "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	u.Path = "/ws"
	return u.String(), nil
}
