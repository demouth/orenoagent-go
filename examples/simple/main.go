package main

import (
	"context"
	"fmt"

	"github.com/demouth/orenoagent-go"
	"github.com/openai/openai-go/v3"
)

func main() {
	// Set the `OPENAI_API_KEY` environment variable
	client := openai.NewClient()

	ctx := context.Background()
	agent := orenoagent.NewAgent(client, []orenoagent.Tool{}, false)

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
