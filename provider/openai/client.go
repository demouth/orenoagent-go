package openai

import (
	"context"
	"errors"

	"github.com/demouth/orenoagent-go/provider"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

type client struct {
	openaiClient openai.Client
	responseID   string
	tools        []provider.Tool

	// Organizational authentication is required to generate inference summaries.
	// https://platform.openai.com/settings/organization/general
	// Supported values: "auto", "concise", "detailed", "none"
	reasoningSummary string

	// Reasoning effort level
	// Supported values: "low", "medium", "high"
	reasoningEffort string

	// Model to use for the agent
	model string

	latestMessageDeltaResult   *MessageDeltaResult
	latestReasoningDeltaResult *ReasoningDeltaResult
}

func newClient(openaiClient openai.Client) *client {
	return &client{
		openaiClient:     openaiClient,
		tools:            []provider.Tool{},
		reasoningSummary: "", // 空文字列 = 未指定
		reasoningEffort:  "", // 空文字列 = 未指定
		model:            openai.ChatModelGPT5Nano,
	}
}

func (c *client) getResponseID() string {
	return c.responseID
}

func (c *client) setResponseID(id string) {
	c.responseID = id
}

func (c *client) getReasoningSummaryParam() openai.ReasoningSummary {
	switch c.reasoningSummary {
	case "auto":
		return openai.ReasoningSummaryAuto
	case "concise":
		return openai.ReasoningSummaryConcise
	case "detailed":
		return openai.ReasoningSummaryDetailed
	default:
		return openai.ReasoningSummaryAuto
	}
}

func (c *client) getReasoningEffortParam() openai.ReasoningEffort {
	switch c.reasoningEffort {
	case "none":
		return openai.ReasoningEffortNone
	case "minimal":
		return openai.ReasoningEffortMinimal
	case "low":
		return openai.ReasoningEffortLow
	case "medium":
		return openai.ReasoningEffortMedium
	case "high":
		return openai.ReasoningEffortHigh
	case "xhigh":
		return openai.ReasoningEffortXhigh
	default:
		return openai.ReasoningEffortHigh
	}
}

func (c *client) callAPI(
	ctx context.Context,
	input responses.ResponseNewParamsInputUnion,
	toolChoiceOption responses.ToolChoiceOptions,
) *ssestream.Stream[responses.ResponseStreamEventUnion] {
	tools := []responses.ToolUnionParam{}
	for _, t := range c.tools {
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

		Model: c.model,
	}

	if c.reasoningSummary != "" || c.reasoningEffort != "" {
		var reasoning openai.ReasoningParam

		// Set summary if specified
		if c.reasoningSummary != "" {
			reasoning.Summary = c.getReasoningSummaryParam()
		}

		// Set effort if specified
		if c.reasoningEffort != "" {
			reasoning.Effort = c.getReasoningEffortParam()
		}

		params.Reasoning = reasoning
	}

	if c.getResponseID() == "" {
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
		params.PreviousResponseID = openai.String(c.getResponseID())
	}
	resp := c.openaiClient.Responses.NewStreaming(ctx, params)

	return resp
}

func (c *client) processMessageInput(
	ctx context.Context,
	yield func(provider.Result) bool,
	question string,
) (Results, error) {

	inputs := responses.ResponseNewParamsInputUnion{
		OfInputItemList: []responses.ResponseInputItemUnionParam{},
	}
	if question != "" {
		inputs.OfInputItemList = append(
			inputs.OfInputItemList,
			responses.ResponseInputItemUnionParam{
				OfInputMessage: &responses.ResponseInputItemMessageParam{
					Role: "user",
					Content: responses.ResponseInputMessageContentListParam{
						responses.ResponseInputContentUnionParam{
							OfInputText: &responses.ResponseInputTextParam{
								Text: question,
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

	stream := c.callAPI(ctx, inputs, responses.ToolChoiceOptionsAuto)
	if err := stream.Err(); err != nil {
		return nil, err
	}
	var results Results
	for stream.Next() {
		if err := stream.Err(); err != nil {
			return nil, err
		}
		event := stream.Current()
		result, err := c.handleResponse(ctx, yield, event)
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
			moreResults, err := c.processFunctionCallInput(ctx, yield, results.MakeToolCallInputs())
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

func (c *client) processFunctionCallInput(
	ctx context.Context,
	yield func(provider.Result) bool,
	input *provider.FunctionCallInput,
) (Results, error) {
	var itemList []responses.ResponseInputItemUnionParam
	for _, param := range input.GetParams() {
		callResult := ""
		for _, t := range c.tools {
			if param.FunctionName == t.Name {
				callResult = t.Function(param.Args)
			}
		}
		itemList = append(itemList, responses.ResponseInputItemUnionParam{
			OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
				CallID: param.CallID,
				Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
					OfString: openai.String(callResult),
				},
			},
		})
	}
	inputs := responses.ResponseNewParamsInputUnion{
		OfInputItemList: itemList,
	}
	stream := c.callAPI(ctx, inputs, responses.ToolChoiceOptionsAuto)

	var results Results
	for stream.Next() {
		if err := stream.Err(); err != nil {
			return nil, err
		}
		event := stream.Current()
		result, err := c.handleResponse(ctx, yield, event)
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

func (c *client) handleResponse(
	_ context.Context,
	yield func(provider.Result) bool,
	event responses.ResponseStreamEventUnion,
) (provider.Result, error) {

	switch event.Type {

	case "response.function_call_arguments.done":
	case "response.output_item.done":
		r := event.AsResponseOutputItemDone()
		switch r.Item.Type {
		case "message":
		case "function_call":
			item := r.Item.AsFunctionCall()
			result := NewFunctionCallResult(item.CallID, item.Name, item.Arguments)
			if !yield(result) {
				return nil, errors.New("cancel iter")
			}
			return result, nil
		}

	case "response.content_part.added":
		t := event.AsResponseContentPartAdded()
		r := NewMessageDeltaResult(t.Part.Text)
		c.latestMessageDeltaResult = r
		if !yield(r) {
			return nil, errors.New("cancel iter")
		}
		return r, nil

	case "response.output_text.delta":
		t := event.AsResponseOutputTextDelta()
		r := c.latestMessageDeltaResult
		r.addDelta(t.Delta)

	case "response.content_part.done":
		if c.latestMessageDeltaResult != nil {
			c.latestMessageDeltaResult.Close()
		}

	case "response.output_text.done":
		t := event.AsResponseOutputTextDone()
		var r provider.Result
		r = NewMessageResult(t.Text)
		if !yield(r) {
			return nil, errors.New("cancel iter")
		}
		return r, nil

	case "response.reasoning_summary_part.added":
		t := event.AsResponseReasoningSummaryPartAdded()
		r := NewReasoningDeltaResult(t.Part.Text)
		c.latestReasoningDeltaResult = r
		if !yield(r) {
			return nil, errors.New("cancel iter")
		}
		return r, nil

	case "response.reasoning_summary_text.delta":
		t := event.AsResponseReasoningSummaryTextDelta()
		r := c.latestReasoningDeltaResult
		r.addDelta(t.Delta)

	case "response.reasoning_summary_part.done":
		if c.latestReasoningDeltaResult != nil {
			c.latestReasoningDeltaResult.Close()
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
		c.setResponseID(t.Response.ID)

	default:

	}

	return nil, nil
}
