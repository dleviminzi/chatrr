package completions

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type OpenAiCompleter struct {
	client *openai.Client
}

func NewOpenAiCompleter(client *openai.Client) OpenAiCompleter {
	return OpenAiCompleter{
		client: client,
	}
}

func (o OpenAiCompleter) Complete(ctx context.Context, msgs []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	return o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{ // TODO: make configurable
		Model:    openai.GPT3Dot5Turbo16K,
		Messages: msgs,
	})
}
