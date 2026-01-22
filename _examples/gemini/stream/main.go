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
	)
	agent := orenoagent.NewAgent(
		provider,
		orenoagent.WithTools(Tools),
	)

	questions := []string{
		"First, use the getWeather tool to check today's weather. Then write a detailed 500-word plan for an outdoor picnic including: 1) What food and drinks to bring, 2) What clothing and accessories to wear based on the weather, 3) Fun activities and games to do, 4) Safety tips. Be thorough and creative!",
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
