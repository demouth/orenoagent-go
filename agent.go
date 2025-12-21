package orenoagent

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type Agent struct {
	responseID string
	llmCaller  *llmCaller
}

// NewAgent
//
// useReasoningSummary:
//   - To obtain a summary using the inference model, set this to true.
//   - Organizational authentication is required to generate inference summaries.
//   - https://platform.openai.com/settings/organization/general
func NewAgent(client openai.Client, tools []Tool, useReasoningSummary bool) *Agent {
	return &Agent{
		llmCaller: newLLMCaller(client, tools, useReasoningSummary),
	}
}

type ResultItr func(func(Result) bool)

func (a *Agent) Ask(ctx context.Context, question string) (ResultItr, error) {
	input := NewMessageInput(question)
	f := func(yield func(Result) bool) {
		_, err := a.llmCaller.processMessageInput(ctx, yield, input)
		if err != nil {
			panic(err)
		}
	}
	return f, nil
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
				fcResult.functionToolCall.CallID,
				fcResult.functionToolCall.Name,
				fcResult.functionToolCall.Arguments,
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
	message responses.ResponseOutputMessage
}

func NewMessageResult(message responses.ResponseOutputMessage) *MessageResult {
	return &MessageResult{
		message: message,
	}
}
func (*MessageResult) isResult() {}
func (r *MessageResult) Type() string {
	return "message"
}
func (r *MessageResult) String() string {
	var outputText strings.Builder
	for _, content := range r.message.Content {
		outputText.WriteString(content.Text)
	}
	return outputText.String()
}

type ReasoningResult struct {
	reasoningSummary responses.ResponseReasoningItemSummary
}

func NewReasoningResult(reasoningSummary responses.ResponseReasoningItemSummary) *ReasoningResult {
	return &ReasoningResult{
		reasoningSummary: reasoningSummary,
	}
}
func (*ReasoningResult) isResult() {}
func (r *ReasoningResult) Type() string {
	return "think"
}
func (r *ReasoningResult) String() string {
	return r.reasoningSummary.Text
}

type FunctionCallResult struct {
	functionToolCall responses.ResponseFunctionToolCall
}

func NewFunctionCallResult(functionToolCall responses.ResponseFunctionToolCall) *FunctionCallResult {
	return &FunctionCallResult{
		functionToolCall: functionToolCall,
	}
}
func (*FunctionCallResult) isResult() {}
func (r *FunctionCallResult) Type() string {
	return "function_call"
}
func (r *FunctionCallResult) String() string {
	return "FunctionToolCall: " + r.functionToolCall.Name + " args:" + r.functionToolCall.Arguments
}
