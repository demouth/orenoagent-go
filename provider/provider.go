package provider

import "context"

// Result is the interface for provider results.
type Result interface {
	Type() string
}

// Provider is the interface for LLM providers.
type Provider interface {
	// ProcessMessage processes a user message and yields results through the yield function.
	// The yield function returns false to cancel processing.
	ProcessMessage(ctx context.Context, yield func(Result) bool, question string) error

	// SetTools sets the tools available to the provider.
	SetTools(tools []Tool)

	// SetReasoningSummary sets the reasoning summary level.
	// Available values: "auto", "concise", "detailed"
	SetReasoningSummary(summary string)

	// SetReasoningEffort sets the reasoning effort level.
	// Available values: "none", "minimal", "low", "medium", "high", "xhigh"
	SetReasoningEffort(effort string)

	// SetModel sets the model to use.
	SetModel(model string)
}
