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

// NewProvider creates a new OpenAI provider.
func NewProvider(openaiClient openai.Client) provider.Provider {
	return &Provider{
		client: newClient(openaiClient),
	}
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

// SetReasoningSummary implements provider.Provider.
func (p *Provider) SetReasoningSummary(summary string) {
	p.client.reasoningSummary = summary
}

// SetReasoningEffort implements provider.Provider.
func (p *Provider) SetReasoningEffort(effort string) {
	p.client.reasoningEffort = effort
}

// SetModel implements provider.Provider.
func (p *Provider) SetModel(model string) {
	p.client.model = model
}
