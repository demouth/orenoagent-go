package openai

import (
	"github.com/demouth/orenoagent-go/provider"
)

// Results is a collection of Result values.
type Results []provider.Result

// HasToolCallResult returns true if any result is a function call.
func (r Results) HasToolCallResult() bool {
	for _, result := range r {
		if result.Type() == "function_call" {
			return true
		}
	}
	return false
}

// MakeToolCallInputs creates a FunctionCallInput from function call results.
func (r Results) MakeToolCallInputs() *provider.FunctionCallInput {
	fcInput := provider.NewFunctionCallInput()
	for _, result := range r {
		if result.Type() == "function_call" {
			fcResult := result.(*provider.FunctionCallResult)
			fcInput.Add(
				fcResult.GetCallID(),
				fcResult.GetName(),
				fcResult.GetArguments(),
			)
		}
	}
	return fcInput
}
