package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/demouth/orenoagent-go/provider"
	"google.golang.org/genai"
)

// Results is a collection of Result values.
type Results []provider.Result

// HasToolCallResult returns true if any result is a function call.
func (r Results) HasToolCallResult() bool {
	for _, result := range r {
		if result.Type() == "function_call" {
			return true
		}
	}
	return false
}

type client struct {
	genaiClient *genai.Client
	chat        *genai.Chat
	tools       []provider.Tool

	// Model to use
	model string

	// Thinking configuration
	thinkingBudget  *int32
	includeThoughts bool

	latestMessageDeltaResult   *provider.MessageDeltaResult
	latestReasoningDeltaResult *provider.ReasoningDeltaResult
}

func newClient(genaiClient *genai.Client) *client {
	return &client{
		genaiClient:     genaiClient,
		tools:           []provider.Tool{},
		model:           "gemini-2.5-flash-lite",
		includeThoughts: false,
	}
}

func (c *client) buildConfig() *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr[float32](1.0),
	}

	// Set thinking config if enabled
	if c.includeThoughts || c.thinkingBudget != nil {
		config.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: c.includeThoughts,
		}
		if c.thinkingBudget != nil {
			config.ThinkingConfig.ThinkingBudget = c.thinkingBudget
		}
	}

	// Set tools if available
	if len(c.tools) > 0 {
		var funcDecls []*genai.FunctionDeclaration
		for _, t := range c.tools {
			funcDecl := &genai.FunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
			}

			// Convert parameters to genai.Schema if provided
			if t.Parameters != nil {
				funcDecl.Parameters = c.convertToSchema(t.Parameters)
			}

			funcDecls = append(funcDecls, funcDecl)
		}
		config.Tools = []*genai.Tool{
			{
				FunctionDeclarations: funcDecls,
			},
		}
	}

	// Set system instruction
	config.SystemInstruction = &genai.Content{
		Parts: []*genai.Part{
			{
				Text: `1. [MUST] Provide answers and reasoning in the language the user speaks to you in. Example: If asked in Japanese, respond in Japanese.
2. [MUST] Never answer with speculation.`,
			},
		},
	}

	return config
}

func (c *client) convertToSchema(params map[string]any) *genai.Schema {
	schema := &genai.Schema{
		Type: genai.TypeObject,
	}

	if props, ok := params["properties"].(map[string]any); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for name, prop := range props {
			if propMap, ok := prop.(map[string]any); ok {
				schema.Properties[name] = c.convertPropertyToSchema(propMap)
			}
		}
	}

	if required, ok := params["required"].([]string); ok {
		schema.Required = required
	} else if required, ok := params["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	return schema
}

func (c *client) convertPropertyToSchema(prop map[string]any) *genai.Schema {
	schema := &genai.Schema{}

	if t, ok := prop["type"].(string); ok {
		switch t {
		case "string":
			schema.Type = genai.TypeString
		case "integer":
			schema.Type = genai.TypeInteger
		case "number":
			schema.Type = genai.TypeNumber
		case "boolean":
			schema.Type = genai.TypeBoolean
		case "array":
			schema.Type = genai.TypeArray
		case "object":
			schema.Type = genai.TypeObject
		}
	}

	if desc, ok := prop["description"].(string); ok {
		schema.Description = desc
	}

	if enum, ok := prop["enum"].([]string); ok {
		schema.Enum = enum
	} else if enum, ok := prop["enum"].([]any); ok {
		for _, e := range enum {
			if s, ok := e.(string); ok {
				schema.Enum = append(schema.Enum, s)
			}
		}
	}

	return schema
}

func (c *client) ensureChat(ctx context.Context) error {
	if c.chat == nil {
		config := c.buildConfig()
		chat, err := c.genaiClient.Chats.Create(ctx, c.model, config, nil)
		if err != nil {
			return fmt.Errorf("failed to create chat: %w", err)
		}
		c.chat = chat
	}
	return nil
}

func (c *client) processMessageInput(
	ctx context.Context,
	yield func(provider.Result) bool,
	question string,
) error {
	if err := c.ensureChat(ctx); err != nil {
		return err
	}

	respIter := c.chat.SendMessageStream(
		ctx,
		genai.Part{Text: question},
	)

	results, err := c.processResponseStream(ctx, yield, respIter)
	if err != nil {
		return err
	}

	// Loop until no more function calls are needed
	for results.HasToolCallResult() {
		funcResults, err := c.executeFunctionCalls(results)
		if err != nil {
			return err
		}

		parts := make([]genai.Part, len(funcResults))
		for i, fr := range funcResults {
			parts[i] = genai.Part{
				FunctionResponse: fr,
			}
		}

		respIter = c.chat.SendMessageStream(ctx, parts...)
		results, err = c.processResponseStream(ctx, yield, respIter)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *client) executeFunctionCalls(results Results) ([]*genai.FunctionResponse, error) {
	var funcResponses []*genai.FunctionResponse

	for _, result := range results {
		if result.Type() != "function_call" {
			continue
		}

		fcResult := result.(*provider.FunctionCallResult)

		// Find and execute the tool
		var callResult string
		for _, t := range c.tools {
			if fcResult.GetName() == t.Name {
				callResult = t.Function(fcResult.GetArguments())
				break
			}
		}

		// Parse the result as JSON if possible, otherwise use as string
		var response map[string]any
		if err := json.Unmarshal([]byte(callResult), &response); err != nil {
			response = map[string]any{"result": callResult}
		}

		funcResponses = append(funcResponses, &genai.FunctionResponse{
			Name:     fcResult.GetName(),
			Response: response,
		})
	}

	return funcResponses, nil
}

func (c *client) processResponseStream(
	_ context.Context,
	yield func(provider.Result) bool,
	respIter func(func(*genai.GenerateContentResponse, error) bool),
) (Results, error) {
	var results Results
	var inThought bool
	var inMessage bool

	for resp, err := range respIter {
		if err != nil {
			return nil, err
		}
		if resp == nil {
			continue
		}

		for _, candidate := range resp.Candidates {
			if candidate.Content == nil {
				continue
			}

			for _, p := range candidate.Content.Parts {
				// Handle function calls
				if p.FunctionCall != nil {
					// Close any open delta results
					if c.latestMessageDeltaResult != nil {
						c.latestMessageDeltaResult.Close()
						c.latestMessageDeltaResult = nil
						inMessage = false
					}
					if c.latestReasoningDeltaResult != nil {
						c.latestReasoningDeltaResult.Close()
						c.latestReasoningDeltaResult = nil
						inThought = false
					}

					argsJSON, err := json.Marshal(p.FunctionCall.Args)
					if err != nil {
						argsJSON = []byte("{}")
					}

					// Gemini doesn't have callID, so we pass empty string
					result := provider.NewFunctionCallResult("", p.FunctionCall.Name, string(argsJSON))
					if !yield(result) {
						return nil, fmt.Errorf("cancelled")
					}
					results = append(results, result)
					continue
				}

				if p.Text == "" {
					continue
				}

				// Handle thoughts
				if p.Thought {
					// Close message delta if switching to thought
					if inMessage && c.latestMessageDeltaResult != nil {
						c.latestMessageDeltaResult.Close()
						c.latestMessageDeltaResult = nil
						inMessage = false
					}

					if !inThought {
						// Start new reasoning delta
						r := provider.NewReasoningDeltaResult(p.Text)
						c.latestReasoningDeltaResult = r
						if !yield(r) {
							return nil, fmt.Errorf("cancelled")
						}
						results = append(results, r)
						inThought = true
					} else {
						// Continue existing reasoning delta
						c.latestReasoningDeltaResult.AddDelta(p.Text)
					}
				} else {
					// Handle regular message
					// Close thought delta if switching to message
					if inThought && c.latestReasoningDeltaResult != nil {
						c.latestReasoningDeltaResult.Close()
						c.latestReasoningDeltaResult = nil
						inThought = false
					}

					if !inMessage {
						// Start new message delta
						r := provider.NewMessageDeltaResult(p.Text)
						c.latestMessageDeltaResult = r
						if !yield(r) {
							return nil, fmt.Errorf("cancelled")
						}
						results = append(results, r)
						inMessage = true
					} else {
						// Continue existing message delta
						c.latestMessageDeltaResult.AddDelta(p.Text)
					}
				}
			}
		}
	}

	// Close any remaining delta results and emit final results
	if c.latestMessageDeltaResult != nil {
		// Emit MessageResult with the complete text
		finalText := c.latestMessageDeltaResult.GetText()
		c.latestMessageDeltaResult.Close()
		c.latestMessageDeltaResult = nil

		messageResult := provider.NewMessageResult(finalText)
		if !yield(messageResult) {
			return nil, fmt.Errorf("cancelled")
		}
		results = append(results, messageResult)
	}
	if c.latestReasoningDeltaResult != nil {
		// Emit ReasoningResult with the complete text
		finalText := c.latestReasoningDeltaResult.GetText()
		c.latestReasoningDeltaResult.Close()
		c.latestReasoningDeltaResult = nil

		reasoningResult := provider.NewReasoningResult(finalText)
		if !yield(reasoningResult) {
			return nil, fmt.Errorf("cancelled")
		}
		results = append(results, reasoningResult)
	}

	return results, nil
}
