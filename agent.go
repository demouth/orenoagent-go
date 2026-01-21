package orenoagent

import (
	"context"
	"fmt"

	"github.com/demouth/orenoagent-go/provider"
	"github.com/demouth/orenoagent-go/provider/openai"
	"github.com/demouth/orenoagent-go/util"
)

type Agent struct {
	prov provider.Provider
}

// AgentOption configures an Agent.
type AgentOption func(*Agent)

// WithTools sets the tools available to the agent.
func WithTools(tools []provider.Tool) AgentOption {
	return func(a *Agent) {
		a.prov.SetTools(tools)
	}
}

// NewAgent creates a new Agent with the given provider.
//
// Example usage:
//
//	provider := openai.NewProvider(client)
//	agent := orenoagent.NewAgent(provider)
//	agent := orenoagent.NewAgent(provider, orenoagent.WithTools(tools))
func NewAgent(prov provider.Provider, opts ...AgentOption) *Agent {
	agent := &Agent{
		prov: prov,
	}

	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

func (a *Agent) Ask(ctx context.Context, question string) (*util.Subscriber[Result], error) {
	subscriber := util.NewSubscriber[Result](100)

	go func() {
		defer subscriber.Close()

		yield := func(providerResult provider.Result) bool {
			agentResult, err := convertProviderResult(providerResult)
			if err != nil {
				subscriber.Publish(NewErrorResult(err))
				return false
			}
			return subscriber.Publish(agentResult)
		}

		err := a.prov.ProcessMessage(ctx, yield, question)
		if err != nil {
			subscriber.Publish(NewErrorResult(err))
			return
		}
	}()

	return subscriber, nil
}

// convertProviderResult converts a provider.Result to an agent Result.
func convertProviderResult(providerResult provider.Result) (Result, error) {
	switch pr := providerResult.(type) {
	case *openai.MessageResult:
		return NewMessageResult(pr.GetText()), nil
	case *openai.MessageDeltaResult:
		return convertMessageDeltaResult(pr), nil
	case *openai.ReasoningResult:
		return NewReasoningResult(pr.GetText()), nil
	case *openai.ReasoningDeltaResult:
		return convertReasoningDeltaResult(pr), nil
	case *openai.FunctionCallResult:
		return NewFunctionCallResult(pr.GetCallID(), pr.GetName(), pr.GetArguments()), nil
	default:
		return nil, fmt.Errorf("unknown provider result type: %T", providerResult)
	}
}

// convertMessageDeltaResult converts a provider MessageDeltaResult to an agent MessageDeltaResult.
func convertMessageDeltaResult(pr *openai.MessageDeltaResult) *MessageDeltaResult {
	// Create a new MessageDeltaResult with empty text initially
	// We'll receive the full stream from the provider
	subscriber := util.NewSubscriber[string](1000)
	agentResult := &MessageDeltaResult{
		text:       "",
		subscriber: subscriber,
	}

	// Subscribe to provider deltas and forward them to agent result
	go func() {
		defer subscriber.Close()
		for delta := range pr.Subscribe() {
			agentResult.text += delta
			subscriber.Publish(delta)
		}
	}()

	return agentResult
}

// convertReasoningDeltaResult converts a provider ReasoningDeltaResult to an agent ReasoningDeltaResult.
func convertReasoningDeltaResult(pr *openai.ReasoningDeltaResult) *ReasoningDeltaResult {
	// Create a new ReasoningDeltaResult with empty text initially
	// We'll receive the full stream from the provider
	subscriber := util.NewSubscriber[string](1000)
	agentResult := &ReasoningDeltaResult{
		text:       "",
		subscriber: subscriber,
	}

	// Subscribe to provider deltas and forward them to agent result
	go func() {
		defer subscriber.Close()
		for delta := range pr.Subscribe() {
			agentResult.text += delta
			subscriber.Publish(delta)
		}
	}()

	return agentResult
}

// Tool is re-exported from provider for convenience.
type Tool = provider.Tool
