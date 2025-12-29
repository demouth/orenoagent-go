package orenoagent

import (
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

type llmCaller struct {
	client     openai.Client
	responseID string
	tools      []Tool

	// Organizational authentication is required to generate inference summaries.
	// https://platform.openai.com/settings/organization/general
	useReasoningSummary bool

	latestMessageDeltaResult   *MessageDeltaResult
	latestReasoningDeltaResult *ReasoningDeltaResult
}

func newLLMCaller(client openai.Client, tools []Tool, useReasoningSummary bool) *llmCaller {
	return &llmCaller{
		client:              client,
		tools:               tools,
		useReasoningSummary: useReasoningSummary,
	}
}
func (a *llmCaller) getResponseID() string {
	return a.responseID
}

func (a *llmCaller) setResponseID(id string) {
	a.responseID = id
}

func (a *llmCaller) callAPI(
	ctx context.Context,
	input responses.ResponseNewParamsInputUnion,
	toolChoiceOption responses.ToolChoiceOptions,
) *ssestream.Stream[responses.ResponseStreamEventUnion] {
	tools := []responses.ToolUnionParam{}
	for _, t := range a.tools {
		tools = append(tools, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        t.Name,
				Description: openai.String(t.Description),
				Parameters:  t.Parameters,
			},
		})
	}

	params := responses.ResponseNewParams{
		Input: input,
		Tools: tools,
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(toolChoiceOption),
		},

		Model: openai.ChatModelGPT5Nano,
		// Model: openai.ChatModelGPT4_1Nano,

	}

	if a.useReasoningSummary {
		params.Reasoning = openai.ReasoningParam{
			Summary: openai.ReasoningSummaryDetailed,
		}
	}

	if a.getResponseID() == "" {
		params.Input.OfInputItemList = append(
			[]responses.ResponseInputItemUnionParam{
				{
					OfInputMessage: &responses.ResponseInputItemMessageParam{
						Role: "system",
						Content: responses.ResponseInputMessageContentListParam{
							responses.ResponseInputContentUnionParam{
								OfInputText: &responses.ResponseInputTextParam{
									Text: `1. [MUST] Provide answers and reasoning in the language the user speaks to you in. Example: If asked in Japanese, respond in Japanese.
2. [MUST] Never answer with speculation.`,
								},
							},
						},
					},
				},
			},
			params.Input.OfInputItemList...,
		)
	} else {
		params.PreviousResponseID = openai.String(a.getResponseID())
	}
	resp := a.client.Responses.NewStreaming(ctx, params)

	return resp
}

func (a *llmCaller) processMessageInput(
	ctx context.Context,
	yield func(Result) bool,
	input *MessageInput,
) ([]Result, error) {

	inputs := responses.ResponseNewParamsInputUnion{
		OfInputItemList: []responses.ResponseInputItemUnionParam{},
	}
	if input != nil {
		inputs.OfInputItemList = append(
			inputs.OfInputItemList,
			responses.ResponseInputItemUnionParam{
				OfInputMessage: &responses.ResponseInputItemMessageParam{
					Role: "user",
					Content: responses.ResponseInputMessageContentListParam{
						responses.ResponseInputContentUnionParam{
							OfInputText: &responses.ResponseInputTextParam{
								Text: input.question,
							},
						},
					},
				},
			},
		)
	}
	inputs.OfInputItemList = append(
		inputs.OfInputItemList,
		responses.ResponseInputItemUnionParam{
			OfInputMessage: &responses.ResponseInputItemMessageParam{
				Role: "developer",
				Content: responses.ResponseInputMessageContentListParam{
					responses.ResponseInputContentUnionParam{
						OfInputText: &responses.ResponseInputTextParam{
							Text: "If tools are available, use them to investigate. If there are no tools or tool calls are not needed, answer directly.",
						},
					},
				},
			},
		},
	)

	stream := a.callAPI(ctx, inputs, responses.ToolChoiceOptionsAuto)
	var results Results
	for stream.Next() {
		if err := stream.Err(); err != nil {
			return nil, err
		}
		event := stream.Current()
		result, err := a.handleResponse(ctx, yield, event)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		results = append(results, result)
	}

	for {
		if results.HasToolCallResult() {
			moreResults, err := a.processFunctionCallInput(ctx, yield, results.MakeToolCallInputs())
			if err != nil {
				return nil, err
			}
			results = moreResults
		} else {
			break
		}
	}
	return nil, nil
}

func (a *llmCaller) processFunctionCallInput(
	ctx context.Context,
	yield func(Result) bool,
	input *FunctionCallInput,
) (Results, error) {
	var itemList []responses.ResponseInputItemUnionParam
	for _, param := range input.param {
		callResult := ""
		for _, t := range a.tools {
			if param.functionName == t.Name {
				callResult = t.Function(param.args)
			}
		}
		itemList = append(itemList, responses.ResponseInputItemUnionParam{
			OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
				CallID: param.callID,
				Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
					OfString: openai.String(callResult),
				},
			},
		})
	}
	inputs := responses.ResponseNewParamsInputUnion{
		OfInputItemList: itemList,
	}
	stream := a.callAPI(ctx, inputs, responses.ToolChoiceOptionsAuto)

	var results Results
	for stream.Next() {
		if err := stream.Err(); err != nil {
			return nil, err
		}
		event := stream.Current()
		result, err := a.handleResponse(ctx, yield, event)
		if err != nil {
			return nil, err
		}
		if result == nil {
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func (a *llmCaller) handleResponse(
	_ context.Context,
	yield func(Result) bool,
	event responses.ResponseStreamEventUnion,
) (Result, error) {

	// fmt.Println("[DEBUG] output.Type:", event.Type)
	// fmt.Println(event.RawJSON())

	switch event.Type {

	case "response.function_call_arguments.done":
	case "response.output_item.done":
		r := event.AsResponseOutputItemDone()
		switch r.Item.Type {
		case "message":
		case "function_call":
			item := r.Item.AsFunctionCall()
			r := NewFunctionCallResult(item.CallID, item.Name, item.Arguments)
			if !yield(r) {
				return nil, errors.New("cancel iter")
			}
			return r, nil
		}

	case "response.content_part.added":
		t := event.AsResponseContentPartAdded()
		r := NewMessageDeltaResult(t.Part.Text)
		a.latestMessageDeltaResult = r
		if !yield(r) {
			return nil, errors.New("cancel iter")
		}
		return r, nil

	case "response.output_text.delta":
		t := event.AsResponseOutputTextDelta()
		r := a.latestMessageDeltaResult
		r.addDelta(t.Delta)

	case "response.content_part.done":
		if a.latestMessageDeltaResult != nil {
			a.latestMessageDeltaResult.Close()
		}

	case "response.output_text.done":
		t := event.AsResponseOutputTextDone()
		var r Result
		r = NewMessageResult(t.Text)
		if !yield(r) {
			return nil, errors.New("cancel iter")
		}
		return r, nil

	case "response.reasoning_summary_part.added":
		t := event.AsResponseReasoningSummaryPartAdded()
		r := NewReasoningDeltaResult(t.Part.Text)
		a.latestReasoningDeltaResult = r
		if !yield(r) {
			return nil, errors.New("cancel iter")
		}
		return r, nil

	case "response.reasoning_summary_text.delta":
		t := event.AsResponseReasoningSummaryTextDelta()
		r := a.latestReasoningDeltaResult
		r.addDelta(t.Delta)

	case "response.reasoning_summary_part.done":
		if a.latestReasoningDeltaResult != nil {
			a.latestReasoningDeltaResult.Close()
		}

	case "response.reasoning_summary_text.done":
		t := event.AsResponseReasoningSummaryTextDone()
		r := NewReasoningResult(t.Text)
		if !yield(r) {
			return nil, errors.New("cancel iter")
		}
		return r, nil

	case "response.completed":
		t := event.AsResponseCompleted()
		a.setResponseID(t.Response.ID)

	default:

	}

	return nil, nil
}
