package provider

// Tool represents a tool that can be used by the agent.
type Tool struct {
	Name        string
	Description string
	Function    func(string) string
	Parameters  map[string]any
}

// Input represents input to the provider.
type Input interface {
	isInput()
	Type() string
}

// MessageInput represents a user message input.
type MessageInput struct {
	question string
}

// NewMessageInput creates a new MessageInput.
func NewMessageInput(question string) *MessageInput {
	return &MessageInput{
		question: question,
	}
}

func (*MessageInput) isInput() {}

func (i *MessageInput) Type() string {
	return "message"
}

// GetQuestion returns the question text.
func (i *MessageInput) GetQuestion() string {
	return i.question
}

// FunctionCallInput represents function call results to be sent back.
type FunctionCallInput struct {
	param []FunctionCallInputParam
}

// FunctionCallInputParam represents a single function call parameter.
type FunctionCallInputParam struct {
	CallID       string
	FunctionName string
	Args         string
}

// NewFunctionCallInput creates a new FunctionCallInput.
func NewFunctionCallInput() *FunctionCallInput {
	return &FunctionCallInput{
		param: []FunctionCallInputParam{},
	}
}

// Add adds a function call parameter.
func (f *FunctionCallInput) Add(callID, functionName, args string) {
	f.param = append(f.param, FunctionCallInputParam{
		CallID:       callID,
		FunctionName: functionName,
		Args:         args,
	})
}

// Len returns the number of function call parameters.
func (f *FunctionCallInput) Len() int {
	return len(f.param)
}

// GetParams returns all function call parameters.
func (f *FunctionCallInput) GetParams() []FunctionCallInputParam {
	return f.param
}

func (FunctionCallInput) isInput() {}

func (i FunctionCallInput) Type() string {
	return "function_call"
}
