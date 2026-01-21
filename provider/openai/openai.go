package openai

import (
	"context"

	"github.com/demouth/orenoagent-go/provider"
	"github.com/openai/openai-go/v3"
)

// Provider is the OpenAI implementation of provider.Provider.
type Provider struct {
	client *client
}

// ProviderOption configures an OpenAI Provider.
type ProviderOption func(*Provider)

// WithModel sets the model to use for the provider.
// Default: openai.ChatModelGPT5Nano
func WithModel(model string) ProviderOption {
	return func(p *Provider) {
		p.client.model = model
	}
}

// WithReasoningSummary sets the reasoning summary level.
// Available values: "auto", "concise", "detailed"
// If not specified, the OpenAI default will be used.
//
// Note:
//   - Organizational authentication is required to use reasoning summaries.
//   - https://platform.openai.com/settings/organization/general
func WithReasoningSummary(summary string) ProviderOption {
	return func(p *Provider) {
		p.client.reasoningSummary = summary
	}
}

// WithReasoningEffort sets the reasoning effort level.
// Available values: "none", "minimal", "low", "medium", "high", "xhigh"
// If not specified, the OpenAI default will be used.
func WithReasoningEffort(effort string) ProviderOption {
	return func(p *Provider) {
		p.client.reasoningEffort = effort
	}
}

// NewProvider creates a new OpenAI provider.
//
// Example usage:
//
//	provider := openai.NewProvider(client)
//	provider := openai.NewProvider(client, openai.WithModel("o3"), openai.WithReasoningEffort("high"))
func NewProvider(openaiClient openai.Client, opts ...ProviderOption) provider.Provider {
	p := &Provider{
		client: newClient(openaiClient),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ProcessMessage implements provider.Provider.
func (p *Provider) ProcessMessage(ctx context.Context, yield func(provider.Result) bool, question string) error {
	_, err := p.client.processMessageInput(ctx, yield, question)
	return err
}

// SetTools implements provider.Provider.
func (p *Provider) SetTools(tools []provider.Tool) {
	p.client.tools = tools
}
