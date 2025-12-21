package orenoagent

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type llmCaller struct {
	client     openai.Client
	responseID string
	tools      []Tool

	// Organizational authentication is required to generate inference summaries.
	// https://platform.openai.com/settings/organization/general
	useReasoningSummary bool
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
) (*responses.Response, error) {
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
	resp, err := a.client.Responses.New(ctx, params)
	if err != nil {
		return nil, err
	}
	a.setResponseID(resp.ID)

	return resp, nil
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

	resp, err := a.callAPI(ctx, inputs, responses.ToolChoiceOptionsAuto)
	if err != nil {
		return nil, err
	}
	results, err := a.handleResponse(ctx, yield, resp)
	if err != nil {
		return nil, err
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
	return results, nil
}

func (a *llmCaller) processFunctionCallInput(
	ctx context.Context,
	yield func(Result) bool,
	input *FunctionCallInput,
) ([]Result, error) {
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
	resp, err := a.callAPI(ctx, inputs, responses.ToolChoiceOptionsAuto)
	if err != nil {
		return nil, err
	}
	return a.handleResponse(ctx, yield, resp)
}
func (a *llmCaller) handleResponse(
	_ context.Context,
	yield func(Result) bool,
	resp *responses.Response,
) (Results, error) {
	var results []Result
	for _, output := range resp.Output {
		switch output.Type {
		case "function_call":
			functionCall := output.AsFunctionCall()
			r := NewFunctionCallResult(functionCall)
			if !yield(r) {
				return results, nil
			}
			results = append(results, r)

		case "message":
			message := output.AsMessage()
			var r Result
			r = NewMessageResult(message)
			if !yield(r) {
				return results, nil
			}
			results = append(results, r)

		case "reasoning":
			reasoning := output.AsReasoning()
			for _, summary := range reasoning.Summary {
				r := NewReasoningResult(summary)
				if !yield(r) {
					return results, nil
				}
				results = append(results, r)
			}

		default:
			fmt.Println("[DEBUG] output.Type:", output.Type)
			fmt.Println(output.RawJSON())

		}
	}

	return results, nil
}
