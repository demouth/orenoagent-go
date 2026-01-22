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

	provider := gemini.NewProvider(client)
	agent := orenoagent.NewAgent(provider)

	question := "Who was the first president of the United States?"
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
		case *orenoagent.FunctionCallResult:
			println("[FunctionCall]")
			println(r.String())
			println()
		}
	}
}
