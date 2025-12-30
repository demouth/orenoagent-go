package orenoagent

import (
	"fmt"

	"github.com/demouth/orenoagent-go/util"
)

// Result is the interface for agent results.
type Result interface {
	isResult()
	Type() string
}

// MessageResult represents a complete message from the agent.
type MessageResult struct {
	text string
}

// NewMessageResult creates a new MessageResult.
func NewMessageResult(text string) *MessageResult {
	return &MessageResult{
		text: text,
	}
}

func (*MessageResult) isResult() {}

func (r *MessageResult) Type() string {
	return "message"
}

func (r *MessageResult) String() string {
	return r.text
}

// MessageDeltaResult represents a streaming message delta from the agent.
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

func (*MessageDeltaResult) isResult() {}

func (r *MessageDeltaResult) Type() string {
	return "message_delta"
}

func (r *MessageDeltaResult) String() string {
	return r.text
}

func (r *MessageDeltaResult) addDelta(text string) {
	r.subscriber.Publish(text)
	r.text = r.text + text
}

func (r *MessageDeltaResult) Subscribe() <-chan string {
	return r.subscriber.Subscribe()
}

func (r *MessageDeltaResult) Close() {
	r.subscriber.Close()
}

// ReasoningDeltaResult represents a streaming reasoning delta from the agent.
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

func (*ReasoningDeltaResult) isResult() {}

func (r *ReasoningDeltaResult) Type() string {
	return "reasoning_delta_result"
}

func (r *ReasoningDeltaResult) String() string {
	return r.text
}

func (r *ReasoningDeltaResult) addDelta(text string) {
	r.subscriber.Publish(text)
	r.text = r.text + text
}

func (r *ReasoningDeltaResult) Subscribe() <-chan string {
	return r.subscriber.Subscribe()
}

func (r *ReasoningDeltaResult) Close() {
	r.subscriber.Close()
}

func (r *ReasoningDeltaResult) GetHistory() []string {
	return r.subscriber.GetHistory()
}

// ReasoningResult represents a complete reasoning from the agent.
type ReasoningResult struct {
	text string
}

// NewReasoningResult creates a new ReasoningResult.
func NewReasoningResult(text string) *ReasoningResult {
	return &ReasoningResult{
		text: text,
	}
}

func (*ReasoningResult) isResult() {}

func (r *ReasoningResult) Type() string {
	return "think"
}

func (r *ReasoningResult) String() string {
	return r.text
}

// FunctionCallResult represents a function call request from the agent.
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

func (*FunctionCallResult) isResult() {}

func (r *FunctionCallResult) Type() string {
	return "function_call"
}

func (r *FunctionCallResult) String() string {
	return "FunctionToolCall: " + r.name + " args:" + r.arguments
}

// ErrorResult represents an error from the agent.
type ErrorResult struct {
	err error
}

// NewErrorResult creates a new ErrorResult.
func NewErrorResult(err error) *ErrorResult {
	return &ErrorResult{err: err}
}

func (*ErrorResult) isResult() {}

func (r *ErrorResult) Type() string {
	return "error"
}

func (r *ErrorResult) String() string {
	return fmt.Sprintf("Error: %v", r.err)
}

func (r *ErrorResult) Error() error {
	return r.err
}
