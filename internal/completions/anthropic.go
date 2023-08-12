package completions

import (
	"context"

	anthrogo "github.com/dleviminzi/go-anthropic"
)

type AnthropicCompleter struct {
	client anthrogo.Client
	model anthrogo.AnthropicModel
}

func (a AnthropicCompleter) Complete(ctx context.Context, msgs anthrogo.Conversation) (*anthrogo.CompleteResponse, error) {
	return a.client.Complete(&anthrogo.CompletePayload{Model: a.model, Prompt: msgs.GeneratePrompt()})
}