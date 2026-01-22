package main

import (
	"context"
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

	provider := gemini.NewProvider(
		client,
		gemini.WithModel("gemini-2.5-flash-lite"),
		gemini.WithIncludeThoughts(true),
		gemini.WithThinkingBudget(1024),
	)
	agent := orenoagent.NewAgent(provider)

	questions := []string{
		"There are red, blue, and green boxes. Red is to the left of blue. Green is to the right of red. Red and green are next to each other. Which box is in the center?",
		"Please repeat the question I just asked.",
	}
	for _, question := range questions {
		println("[Question]")
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
			case *orenoagent.MessageResult:
				println("[Message]")
				println(r.String())
				println()
			case *orenoagent.ReasoningResult:
				println("[Reasoning]")
				println(r.String())
				println()
			case *orenoagent.FunctionCallResult:
				println("[FunctionCall]")
				println(r.String())
				println()
			}
		}
	}
}
