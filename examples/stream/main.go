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
	agent := orenoagent.NewAgent(client,
		Tools,
		orenoagent.WithReasoningSummary("detailed"),
		orenoagent.WithModel(openai.ChatModelGPT5Nano),
	)

	questions := []string{
		"Do I need an umbrella when I go out today?",
		"What clothing do you recommend for today?",
	}
	for _, question := range questions {
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
		Name:        "getWeather",
		Description: "Get the weather information for today, tomorrow, and the day after tomorrow.",
		Function: func(_ string) string {
			return "Today's weather: Light rain, 15°C. Tomorrow's weather: Sunny, 20°C. Day after tomorrow's weather: Cloudy, 18°C."
		},
	},
}
