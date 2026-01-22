package gemini

import (
	"context"

	"github.com/demouth/orenoagent-go/provider"
	"google.golang.org/genai"
)

// Provider is the Gemini implementation of provider.Provider.
type Provider struct {
	client *client
}

// ProviderOption configures a Gemini Provider.
type ProviderOption func(*Provider)

// WithModel sets the model to use for the provider.
// Default: "gemini-2.5-flash-preview-04-17"
func WithModel(model string) ProviderOption {
	return func(p *Provider) {
		p.client.model = model
	}
}

// WithThinkingBudget sets the thinking budget for the provider.
// This controls how much "thinking" the model can do before responding.
func WithThinkingBudget(budget int32) ProviderOption {
	return func(p *Provider) {
		p.client.thinkingBudget = &budget
	}
}

// WithIncludeThoughts sets whether to include thoughts in the response.
// When enabled, the model's reasoning process will be visible.
func WithIncludeThoughts(include bool) ProviderOption {
	return func(p *Provider) {
		p.client.includeThoughts = include
	}
}

// NewProvider creates a new Gemini provider.
//
// Example usage:
//
//	provider := gemini.NewProvider(client)
//	provider := gemini.NewProvider(client, gemini.WithModel("gemini-2.5-flash-lite"), gemini.WithIncludeThoughts(true))
func NewProvider(genaiClient *genai.Client, opts ...ProviderOption) provider.Provider {
	p := &Provider{
		client: newClient(genaiClient),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ProcessMessage implements provider.Provider.
func (p *Provider) ProcessMessage(ctx context.Context, yield func(provider.Result) bool, question string) error {
	return p.client.processMessageInput(ctx, yield, question)
}

// SetTools implements provider.Provider.
func (p *Provider) SetTools(tools []provider.Tool) {
	p.client.tools = tools
}
