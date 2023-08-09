package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/sashabaranov/go-openai"

	"github.com/dleviminzi/chatrr/internal/chatrr"
	"github.com/dleviminzi/chatrr/internal/completions"
	"github.com/dleviminzi/chatrr/internal/db"
	"github.com/dleviminzi/chatrr/internal/embeddings"
)

const explanation = `Welcome to Chatrr. This is a normal chat bot conversation with one exception. 
You can use the reserved word memorize followed by a number n to tell the bot 
to create a memory of the last n (prompt, response) pairs.`

func main() {
	var (
		ctx          = context.Background()
		model        string
		openaiKey    = os.Getenv("OPENAI_API_KEY")
		openaiClient = openai.NewClient(openaiKey)
		db           = db.NewDatabaseConnection()
	)
	defer db.DB.Close()

	flag.StringVar(&model, "model", openai.GPT3Dot5Turbo16K, "which model to use (default gpt3.5 turbo 16k)")
	flag.Parse()
	completer := completions.NewOpenAiCompleter(openaiClient, model)
	embedder := embeddings.NewOpenAiEmbedder(openaiClient)
	chatrr := chatrr.NewChatrr(db, embedder, completer)
	chatrr.NewActiveConvo()

	// Handle SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		chatrr.StoreActiveConvo()
		fmt.Println("\nGoodbye!")
		os.Exit(0)
	}()

	fmt.Println(explanation)

	// Chat loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("User: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.HasPrefix(input, "memorize") {
			n := 2

			parts := strings.SplitN(input, " ", 2)
			if len(parts) > 1 {
				if parsedN, err := strconv.Atoi(parts[1]); err == nil {
					n = parsedN
				} else {
					fmt.Println("Invalid number for memorize. Using default value.")
				}
			}

			go chatrr.Memorize(ctx, n)
			continue
		}

		response, err := chatrr.GetResponse(ctx, input)
		if err != nil {
			fmt.Println("Chatrr: Uh oh, my brain doesn't seem to be working. You can try again, but I might be a goner.")
			continue
		}

		fmt.Println("Chatrr:", response)
	}
}
