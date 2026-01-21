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
}
