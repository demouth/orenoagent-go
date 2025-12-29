package orenoagent

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
)

type Agent struct {
	responseID string
	llmCaller  *llmCaller
}

// AgentOption configures an Agent.
type AgentOption func(*Agent)

// WithTools sets the tools available to the agent.
func WithTools(tools []Tool) AgentOption {
	return func(a *Agent) {
		a.llmCaller.tools = tools
	}
}

// WithReasoningSummary sets the reasoning summary level.
// Available values: "auto", "concise", "detailed", "none" (default: "none")
//
// Note:
//   - Organizational authentication is required to use reasoning summaries.
//   - https://platform.openai.com/settings/organization/general
func WithReasoningSummary(summary string) AgentOption {
	return func(a *Agent) {
		a.llmCaller.reasoningSummary = summary
	}
}

// WithModel sets the model to use for the agent.
// Default: openai.ChatModelGPT5Nano
func WithModel(model string) AgentOption {
	return func(a *Agent) {
		a.llmCaller.model = model
	}
}

// NewAgent creates a new Agent with the given client.
//
// Example usage:
//
//	agent := orenoagent.NewAgent(client)
//	agent := orenoagent.NewAgent(client, orenoagent.WithTools(tools), orenoagent.WithReasoningSummary("detailed"))
func NewAgent(client openai.Client, opts ...AgentOption) *Agent {
	agent := &Agent{
		llmCaller: newLLMCaller(client),
	}

	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

func (a *Agent) Ask(ctx context.Context, question string) (*Subscriber[Result], error) {
	input := NewMessageInput(question)
	subscriber := NewSubscriber[Result](100)

	go func() {
		defer subscriber.Close()

		yield := func(r Result) bool {
			return subscriber.Publish(r)
		}

		_, err := a.llmCaller.processMessageInput(ctx, yield, input)
		if err != nil {
			subscriber.Publish(NewErrorResult(err))
			return
		}
	}()

	return subscriber, nil
}

// Tool

type Tool struct {
	Name        string
	Description string
	Function    func(string) string
	Parameters  map[string]any
}

// Input

type Input interface {
	isInput()
	Type() string
}

type MessageInput struct {
	question string
}

func NewMessageInput(question string) *MessageInput {
	return &MessageInput{
		question: question,
	}
}
func (*MessageInput) isInput() {}
func (i *MessageInput) Type() string {
	return "message"
}

type FunctionCallInput struct {
	param []FunctionCallInputParam
}
type FunctionCallInputParam struct {
	callID       string
	functionName string
	args         string
}

func NewFunctionCallInput() *FunctionCallInput {
	return &FunctionCallInput{
		param: []FunctionCallInputParam{},
	}
}
func (f *FunctionCallInput) add(callID, functionName, args string) {
	f.param = append(f.param, FunctionCallInputParam{
		callID:       callID,
		functionName: functionName,
		args:         args,
	})
}
func (f *FunctionCallInput) Len() int {
	return len(f.param)
}

func (FunctionCallInput) isInput() {}
func (i FunctionCallInput) Type() string {
	return "function_call"
}

// Result

type Results []Result

func (r Results) HasToolCallResult() bool {
	for _, result := range r {
		if result.Type() == "function_call" {
			return true
		}
	}
	return false
}
func (r Results) MakeToolCallInputs() *FunctionCallInput {
	fcInput := NewFunctionCallInput()
	for _, result := range r {
		if result.Type() == "function_call" {
			fcResult := result.(*FunctionCallResult)
			fcInput.add(
				fcResult.callID,
				fcResult.name,
				fcResult.arguments,
			)
		}
	}
	return fcInput
}

type Result interface {
	isResult()
	Type() string
}

type MessageResult struct {
	text string
}

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

type MessageDeltaResult struct {
	text       string
	subscriber *Subscriber[string]
}

func NewMessageDeltaResult(text string) *MessageDeltaResult {
	subscriber := NewSubscriber[string](1000)
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

type ReasoningDeltaResult struct {
	text       string
	subscriber *Subscriber[string]
}

func NewReasoningDeltaResult(text string) *ReasoningDeltaResult {
	subscriber := NewSubscriber[string](1000)
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

type ReasoningResult struct {
	text string
}

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

type FunctionCallResult struct {
	callID    string
	name      string
	arguments string
}

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

type ErrorResult struct {
	err error
}

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
