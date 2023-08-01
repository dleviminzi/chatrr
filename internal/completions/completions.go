package completions

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type OpenAiCompleter struct {
	client *openai.Client
	model  string
}

func NewOpenAiCompleter(client *openai.Client, model string) OpenAiCompleter {
	return OpenAiCompleter{
		client: client,
		model:  model,
	}
}

func (o OpenAiCompleter) Complete(ctx context.Context, msgs []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	return o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{ // TODO: make configurable
		Model:    o.model,
		Messages: msgs,
	})
}
