package chatrr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/sashabaranov/go-openai"

	"github.com/dleviminzi/chatrr/internal/models"
)

type DatabaseConnector interface {
	GetConversationMemories(embedding []float32) ([]models.RecalledMemory, error)
	CreateConversationMemories(embedding [][]float32, conversationId int, input []openai.ChatCompletionMessage) error
	CreateConversation([]openai.ChatCompletionMessage) (int, error)
	UpdateConversatoin(convoId int, convo []openai.ChatCompletionMessage) error
}

type Embedder interface {
	Embed(ctx context.Context, input string) ([]float32, error)
}

type Completer interface {
	Complete(ctx context.Context, msgs []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error)
}

type Chatrr struct {
	db        DatabaseConnector
	embedder  Embedder
	completer Completer
	msgs      []openai.ChatCompletionMessage
	convoId   int
}

func NewChatrr(db DatabaseConnector, embedder Embedder, completer Completer) Chatrr {
	return Chatrr{
		db:        db,
		embedder:  embedder,
		completer: completer,
		msgs: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "Assistant, the following is a conversation between you and a user. If you know the user's name, address them with it. If you don't know the answer to a question, let the user know.",
			},
		},
	}
}

func (c *Chatrr) NewActiveConvo() error {
	convoId, err := c.db.CreateConversation(c.msgs)
	if err != nil {
		return err
	}

	c.convoId = convoId
	return nil
}

func (c *Chatrr) StoreActiveConvo() error {
	return c.db.UpdateConversatoin(c.convoId, c.msgs)
}

func (c *Chatrr) GetResponse(ctx context.Context, userInput string) (string, error) {
	memories, err := c.GetMemories(ctx, userInput)
	if err != nil {
		return "", err
	}

	convoFrags, err := evaluateCandidateMemories(memories)
	if err != nil {
		log.Println(err.Error())
	}

	memoryWithUserMsg := formatUserMemory(convoFrags)
	memoryWithUserMsg.Content += userInput
	c.msgs = append(c.msgs, memoryWithUserMsg)

	completion, err := c.completer.Complete(ctx, c.msgs)
	if err != nil {
		return "", err
	}

	response := completion.Choices[0].Message
	c.msgs = append(c.msgs, response)
	return response.Content, nil
}

func (c Chatrr) GetMemories(ctx context.Context, input string) ([]models.RecalledMemory, error) {
	embedding, err := c.embedder.Embed(ctx, input)
	if err != nil {
		return nil, err
	}

	return c.db.GetConversationMemories(embedding)
}

func (c Chatrr) Memorize(ctx context.Context, numBack int) {
	convoLength := len(c.msgs)

	// Determine start point
	start := 0
	if convoLength >= 2*numBack {
		start = convoLength - 2*numBack
	}

	embeddings := [][]float32{}
	for i := start; i < convoLength; i++ {
		e, err := c.embedder.Embed(ctx, c.msgs[i].Content)
		if err != nil {
			// Memorize ignores errors? idk
			log.Println(err.Error())
			return
		}

		embeddings = append(embeddings, e)
	}

	// TODO: come up with conversation storage plan
	err := c.db.CreateConversationMemories(embeddings, c.convoId, c.msgs[start:])
	if err != nil {
		log.Println(err.Error())
	}
}

func evaluateCandidateMemories(memories []models.RecalledMemory) ([]openai.ChatCompletionMessage, error) {
	var minSimilarityThreshold float32 = 0.75
	stddev, mean := calculateStddevAndMean(memories, minSimilarityThreshold)

	frags := []openai.ChatCompletionMessage{}
	for _, memory := range memories {
		// Throw out memories that seem to weak
		if memory.SimilarityScore < minSimilarityThreshold {
			continue
		}

		// Abnormally strong memory means we will now raise min similarity threshold so that we
		// only consider adding other abnormally strong memories.
		if memory.SimilarityScore >= mean+stddev {
			minSimilarityThreshold = mean + stddev
		}

		var frag []openai.ChatCompletionMessage
		err := json.Unmarshal([]byte(memory.ConversationFragment), &frag)
		if err != nil {
			return nil, err
		}

		frags = append(frags, frag...)
	}

	return frags, nil
}

func calculateStddevAndMean(memories []models.RecalledMemory, staticThreshold float32) (float32, float32) {
	var numMemories float32 = float32(len(memories))

	var mean float32 = 0
	for _, memory := range memories {
		if memory.SimilarityScore < staticThreshold {
			numMemories -= 1
			continue
		}
		mean += memory.SimilarityScore
	}

	mean /= float32(numMemories)

	var sumOfSqrDiff float64 = 0
	for _, memory := range memories {
		if memory.SimilarityScore < staticThreshold {
			continue
		}
		sumOfSqrDiff += math.Pow(float64(memory.SimilarityScore-mean), 2)
	}

	stddev := math.Sqrt(sumOfSqrDiff / float64(numMemories))

	return float32(stddev), mean
}

func formatUserMemory(convoFrags []openai.ChatCompletionMessage) openai.ChatCompletionMessage {
	msg := openai.ChatCompletionMessage{
		Role:    "user",
		Content: "Assistant, this is a previous conversation between us.\n",
	}

	injectedMemories := 0
	for _, frag := range convoFrags {
		if frag.Role == "system" {
			continue
		}

		msg.Content += fmt.Sprintf("%s said %s \n", frag.Role, frag.Content)
		injectedMemories += 1
	}

	// if no memories are injected the other content will confuse poor AI friend
	if injectedMemories < 1 {
		return openai.ChatCompletionMessage{
			Role: "user",
		}
	}

	msg.Content += "If this conversation seems relevant, you can use it to inform your response to my message. However, it is critical that you do not thank me for sharing it. It is also critical that you do not apologize.\n\n"
	return msg
}
