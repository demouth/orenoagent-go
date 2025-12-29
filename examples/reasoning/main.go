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
	agent := orenoagent.NewAgent(
		client,
		orenoagent.WithReasoningSummary("detailed"),
		orenoagent.WithModel(openai.ChatModelGPT5Nano),
	)

	questions := []string{
		"赤・青・緑の箱がある。赤は青の左。緑は赤の右。赤と緑は隣り合っている。中央にある箱は？",
		"さっき私がした質問をもう一度繰り返して",
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
