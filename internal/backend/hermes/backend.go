// Package hermes is a thin Lucinate backend for the Nous Research
// Hermes Agent (https://github.com/nousresearch/hermes-agent). Hermes
// exposes an OpenAI-compatible /v1 surface, so the heavy lifting —
// chat completions streaming, agent storage, models list — comes from
// embedding *openai.Backend. Only the defaults and registration are
// Hermes-specific.
//
// A Lucinate Hermes connection maps 1:1 to a Hermes profile: each
// profile runs its own API server on its own port with its own pinned
// model, so multiplexing across profiles is not possible over a single
// connection. Future Hermes-specific features (Runs API for proper
// cancellation, /v1/responses for server-side state, surfacing the
// custom hermes.tool.progress SSE event) would override the embedded
// methods on this type.
package hermes

import (
	"time"

	openaiBackend "github.com/lucinate-ai/lucinate/internal/backend/openai"
)

// DefaultBaseURL is the loopback URL Hermes binds when API_SERVER_HOST
// is unset, with the /v1 suffix the OpenAI-compatible endpoints live
// under.
const DefaultBaseURL = "http://127.0.0.1:8642/v1"

// Options bundles the per-connection configuration the Backend needs.
// Mirrors openai.Options (the inner backend) but applies Hermes
// defaults.
type Options struct {
	ConnectionID   string
	BaseURL        string
	APIKey         string
	DefaultModel   string
	ConnectTimeout time.Duration
}

// Backend embeds *openai.Backend so all backend.Backend methods —
// Connect, ChatSend, ListAgents, ModelsList, etc. — are inherited
// unchanged. The distinct type exists so the connection type dispatch
// stays explicit and so Hermes-specific behavior can be layered on
// later without touching the OpenAI backend.
type Backend struct {
	*openaiBackend.Backend
}

// New constructs a Backend with Hermes defaults applied. An empty
// BaseURL falls back to DefaultBaseURL; everything else is forwarded
// to openai.New verbatim.
func New(opts Options) (*Backend, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultBaseURL
	}
	inner, err := openaiBackend.New(openaiBackend.Options{
		ConnectionID:   opts.ConnectionID,
		BaseURL:        opts.BaseURL,
		APIKey:         opts.APIKey,
		DefaultModel:   opts.DefaultModel,
		ConnectTimeout: opts.ConnectTimeout,
	})
	if err != nil {
		return nil, err
	}
	return &Backend{Backend: inner}, nil
}
