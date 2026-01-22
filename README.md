# orenoagent-go

A lightweight AI agent framework supporting multiple LLM providers.

## Demo

![Demo](screencapture.gif)

## Features

- Multi-provider support (OpenAI, Gemini)
- Streaming support for real-time output
- Tool calling and function execution
- Easy provider switching

## Requirements

Set the appropriate environment variable for your provider:

| Provider | Environment Variable |
|----------|---------------------|
| OpenAI   | `OPENAI_API_KEY`    |
| Gemini   | `GEMINI_API_KEY` or `GOOGLE_API_KEY` |

## Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/demouth/orenoagent-go"
    "github.com/demouth/orenoagent-go/provider/openai"  // or "provider/gemini"
    openaiSDK "github.com/openai/openai-go/v3"
)

func main() {
    client := openaiSDK.NewClient()
    ctx := context.Background()

    // Switch providers by changing this line
    provider := openai.NewProvider(client)
    agent := orenoagent.NewAgent(provider)

    subscriber, _ := agent.Ask(ctx, "Hello!")
    for result := range subscriber.Subscribe() {
        switch r := result.(type) {
        case *orenoagent.ErrorResult:
            fmt.Printf("Error: %v\n", r.Error())
        case *orenoagent.MessageResult:
            println(r.String())
        }
    }
}
```

### Using Gemini

```go
import (
    "github.com/demouth/orenoagent-go/provider/gemini"
    "google.golang.org/genai"
)

client, _ := genai.NewClient(ctx, nil)
provider := gemini.NewProvider(client)
```

See `_examples/` for more usage examples.
