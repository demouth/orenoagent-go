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
	agent := orenoagent.NewAgent(client, []orenoagent.Tool{}, true)

	questions := []string{
		"赤・青・緑の箱がある。赤は青の左。緑は赤の右。赤と緑は隣り合っている。中央にある箱は？",
		"小学生低学年が分かるように解説して",
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
			case *orenoagent.MessageDeltaResult:
				println("[Message Stream]")
				for delta := range r.Subscribe() {
					print(delta)
				}
				println()
				println()
			case *orenoagent.ReasoningDeltaResult:
				println("[Reasoning Stream]")
				for delta := range r.Subscribe() {
					print(delta)
				}
				println()
				println()
			case *orenoagent.FunctionCallResult:
				println("[FunctionCall]")
				println(r.String())
				println()
			}
		}
	}
}
