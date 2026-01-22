package provider

import (
	"github.com/demouth/orenoagent-go/util"
)

// MessageResult represents a complete message from the LLM.
type MessageResult struct {
	text string
}

// NewMessageResult creates a new MessageResult.
func NewMessageResult(text string) *MessageResult {
	return &MessageResult{
		text: text,
	}
}

func (r *MessageResult) Type() string {
	return "message"
}

// GetText returns the message text.
func (r *MessageResult) GetText() string {
	return r.text
}

// MessageDeltaResult represents a streaming message delta from the LLM.
type MessageDeltaResult struct {
	text       string
	subscriber *util.Subscriber[string]
}

// NewMessageDeltaResult creates a new MessageDeltaResult.
func NewMessageDeltaResult(text string) *MessageDeltaResult {
	subscriber := util.NewSubscriber[string](1000)
	r := &MessageDeltaResult{
		text:       text,
		subscriber: subscriber,
	}
	subscriber.Publish(text)
	return r
}

func (r *MessageDeltaResult) Type() string {
	return "message_delta"
}

// GetText returns the current message text.
func (r *MessageDeltaResult) GetText() string {
	return r.text
}

// AddDelta adds a delta to the message.
func (r *MessageDeltaResult) AddDelta(text string) {
	r.subscriber.Publish(text)
	r.text = r.text + text
}

// Subscribe returns a channel to receive message deltas.
func (r *MessageDeltaResult) Subscribe() <-chan string {
	return r.subscriber.Subscribe()
}

// Close closes the subscriber.
func (r *MessageDeltaResult) Close() {
	r.subscriber.Close()
}

// ReasoningResult represents a complete reasoning from the LLM.
type ReasoningResult struct {
	text string
}

// NewReasoningResult creates a new ReasoningResult.
func NewReasoningResult(text string) *ReasoningResult {
	return &ReasoningResult{
		text: text,
	}
}

func (r *ReasoningResult) Type() string {
	return "think"
}

// GetText returns the reasoning text.
func (r *ReasoningResult) GetText() string {
	return r.text
}

// ReasoningDeltaResult represents a streaming reasoning delta from the LLM.
type ReasoningDeltaResult struct {
	text       string
	subscriber *util.Subscriber[string]
}

// NewReasoningDeltaResult creates a new ReasoningDeltaResult.
func NewReasoningDeltaResult(text string) *ReasoningDeltaResult {
	subscriber := util.NewSubscriber[string](1000)
	r := &ReasoningDeltaResult{
		text:       text,
		subscriber: subscriber,
	}
	subscriber.Publish(text)
	return r
}

func (r *ReasoningDeltaResult) Type() string {
	return "reasoning_delta_result"
}

// GetText returns the current reasoning text.
func (r *ReasoningDeltaResult) GetText() string {
	return r.text
}

// AddDelta adds a delta to the reasoning.
func (r *ReasoningDeltaResult) AddDelta(text string) {
	r.subscriber.Publish(text)
	r.text = r.text + text
}

// Subscribe returns a channel to receive reasoning deltas.
func (r *ReasoningDeltaResult) Subscribe() <-chan string {
	return r.subscriber.Subscribe()
}

// Close closes the subscriber.
func (r *ReasoningDeltaResult) Close() {
	r.subscriber.Close()
}

// GetHistory returns all deltas.
func (r *ReasoningDeltaResult) GetHistory() []string {
	return r.subscriber.GetHistory()
}

// FunctionCallResult represents a function call request from the LLM.
type FunctionCallResult struct {
	callID    string
	name      string
	arguments string
}

// NewFunctionCallResult creates a new FunctionCallResult.
func NewFunctionCallResult(callID, name, arguments string) *FunctionCallResult {
	return &FunctionCallResult{
		callID:    callID,
		name:      name,
		arguments: arguments,
	}
}

func (r *FunctionCallResult) Type() string {
	return "function_call"
}

// GetCallID returns the call ID.
func (r *FunctionCallResult) GetCallID() string {
	return r.callID
}

// GetName returns the function name.
func (r *FunctionCallResult) GetName() string {
	return r.name
}

// GetArguments returns the function arguments as JSON string.
func (r *FunctionCallResult) GetArguments() string {
	return r.arguments
}
