package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/demouth/orenoagent-go"
	"github.com/demouth/orenoagent-go/provider/gemini"
	"google.golang.org/genai"
)

func main() {
	// Set the `GEMINI_API_KEY` or `GOOGLE_API_KEY` environment variable
	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		panic(err)
	}

	questions := []string{
		"Which book do I own has the most pages? Please tell me the title and page count of that book.",
		"Please list my books sorted by page count.",
	}

	provider := gemini.NewProvider(
		client,
		gemini.WithModel("gemini-2.5-flash-lite"),
		gemini.WithIncludeThoughts(true),
		gemini.WithThinkingBudget(512),
	)
	agent := orenoagent.NewAgent(
		provider,
		orenoagent.WithTools(Tools),
	)

	for i, question := range questions {
		if i > 0 {
			println("\n---\n")
		}
		println(decoration("Question"))
		println(question)
		println()
		subscriber, err := agent.Ask(ctx, question)
		if err != nil {
			panic(err)
		}
		for result := range subscriber.Subscribe() {
			switch r := result.(type) {
			case *orenoagent.ErrorResult:
				fmt.Printf("Error: %v\n", r.Error())
				return
			case *orenoagent.MessageDeltaResult:
				println(decoration("Message"))
				for delta := range r.Subscribe() {
					print(delta)
				}
				println()
				println()
			case *orenoagent.ReasoningDeltaResult:
				println(decoration("Reasoning"))
				print("\033[2m")
				for delta := range r.Subscribe() {
					print(delta)
				}
				println("\033[0m")
				println()
			case *orenoagent.FunctionCallResult:
				println(decoration("FunctionCall"))
				print("\033[2m")
				print(r.String())
				println("\033[0m")
				println()
			}
		}
	}
}

func decoration(s string) string {
	return fmt.Sprintf("\033[1;4m[%s]\033[0m", s)
}

var Tools = []orenoagent.Tool{
	{
		Name:        "get_current_weather",
		Description: "Get the current weather for a location.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "The city and state, e.g. San Francisco, CA",
				},
				"unit": map[string]any{
					"type":        "string",
					"enum":        []string{"celsius", "fahrenheit"},
					"description": "The unit of temperature",
				},
			},
			"required": []string{"location", "unit"},
		},
		Function: func(args string) string {
			var param struct {
				Location string `json:"location"`
				Unit     string `json:"unit"`
			}
			if err := json.Unmarshal([]byte(args), &param); err != nil {
				return fmt.Sprintf("error: %v", err)
			}
			result := map[string]any{
				"location":    param.Location,
				"unit":        param.Unit,
				"weather":     "sunny",
				"temperature": 20,
			}
			b, _ := json.Marshal(result)
			return string(b)
		},
	},
	{
		Name:        "get_book_list",
		Description: "Get the list of books owned by the user.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Function: func(_ string) string {
			books := []map[string]any{
				{"id": 1, "title": "Concurrency in Go"},
				{"id": 2, "title": "The Readable Code"},
				{"id": 3, "title": "Clean Architecture"},
			}
			result := map[string]any{"books": books}
			b, _ := json.Marshal(result)
			return string(b)
		},
	},
	{
		Name:        "get_book_detail",
		Description: "Get detailed information (price, page count) for a book by its ID.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"book_id": map[string]any{
					"type":        "integer",
					"description": "The ID of the book",
				},
			},
			"required": []string{"book_id"},
		},
		Function: func(args string) string {
			var param struct {
				BookID float64 `json:"book_id"`
			}
			if err := json.Unmarshal([]byte(args), &param); err != nil {
				return fmt.Sprintf("error: %v", err)
			}

			bookDetails := map[int]map[string]any{
				1: {"id": 1, "title": "Concurrency in Go", "price": 32, "pages": 304},
				2: {"id": 2, "title": "The Readable Code", "price": 26, "pages": 260},
				3: {"id": 3, "title": "Clean Architecture", "price": 35, "pages": 336},
			}

			detail, exists := bookDetails[int(param.BookID)]
			if !exists {
				return `{"error": "book not found"}`
			}

			b, _ := json.Marshal(detail)
			return string(b)
		},
	},
}
