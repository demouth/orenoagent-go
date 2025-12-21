package main

import (
	"context"

	"github.com/demouth/orenoagent"
	"github.com/openai/openai-go/v3"
)

func main() {
	client := openai.NewClient()
	ctx := context.Background()
	agent := orenoagent.NewAgent(client, []orenoagent.Tool{}, true)

	question := "Who was the first president of the United States?"
	println("[Question]")
	println(question)
	println()
	results, err := agent.Ask(ctx, question)
	if err != nil {
		panic(err)
	}
	for result := range results {
		switch r := result.(type) {
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
		default:
			panic("unkown result type")
		}
	}
}
