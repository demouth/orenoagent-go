# orenoagent-go

A simple AI agent implementation using OpenAI's Reasoning models.

## Overview

- A learning-focused AI agent implementation
- Built with OpenAI's Reasoning models (GPT-5)

## Requirements

- Set the `OPENAI_API_KEY` environment variable
- To enable `useReasoningSummary`, you need to verify your organization in the OpenAI settings

## Example

```go
package main

import (
    "context"
    "github.com/demouth/orenoagent"
    "github.com/openai/openai-go/v3"
)

func main() {
    client := openai.NewClient()
    ctx := context.Background()

    agent := orenoagent.NewAgent(client, tools, true)
    results, _ := agent.Ask(ctx, "What is the current date and time?")

    for result := range results {
        switch r := result.(type) {
        case *orenoagent.MessageResult:
            println(r.String())
        case *orenoagent.ReasoningResult:
            println(r.String())
        case *orenoagent.FunctionCallResult:
            println(r.String())
        }
    }
}
```

---

## 概要

「俺のエージェント」は OpenAI の Reasoning models を使用したシンプルな AI エージェントの実装です。

- 学習用に作成した AI エージェント
- OpenAI の Reasoning models（GPT-5）を使用

## 必要な設定

- 環境変数 `OPENAI_API_KEY` の設定が必要です
- `useReasoningSummary` を有効にする場合は、OpenAI の設定画面で organization の検証が必要です

## 使用例

```go
package main

import (
    "context"
    "github.com/demouth/orenoagent"
    "github.com/openai/openai-go/v3"
)

func main() {
    client := openai.NewClient()
    ctx := context.Background()

    agent := orenoagent.NewAgent(client, tools, true)
    results, _ := agent.Ask(ctx, "What is the current date and time?")

    for result := range results {
        switch r := result.(type) {
        case *orenoagent.MessageResult:
            println(r.String())
        case *orenoagent.ReasoningResult:
            println(r.String())
        case *orenoagent.FunctionCallResult:
            println(r.String())
        }
    }
}
```
