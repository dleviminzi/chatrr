package embeddings

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type OpenAiEmbedder struct {
	client *openai.Client
}

func NewOpenAiEmbedder(client *openai.Client) OpenAiEmbedder {
	return OpenAiEmbedder{
		client: client,
	}
}

func (o OpenAiEmbedder) Embed(ctx context.Context, input string) ([]float32, error) {
	embedding, err := o.client.CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
		Input: []string{input},
		Model: openai.AdaEmbeddingV2, // TODO: make configurable
	})
	if err != nil {
		return nil, err
	}

	if len(embedding.Data) < 1 {
		return nil, fmt.Errorf("failed to embed: %s", input)
	}

	return embedding.Data[0].Embedding, nil
}
