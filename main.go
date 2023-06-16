package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	_ "strings"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	openai "github.com/sashabaranov/go-openai"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

const (
	OpenAIAPIKey = "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"
)

func GetEventHandler(client *whatsmeow.Client, gpt *openai.Client) func(interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			var messageBody = v.Message.GetConversation()
			fmt.Println("Message event:", messageBody)
			if messageBody == "ping" {
				client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
					Conversation: proto.String("pong"),
				})
			} else {
				response := GenerateGPTResponse(messageBody, gpt)
				if len(response) > 0 {
					client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						Conversation: proto.String(response),
					})
				}
			}
		}
	}
}

func GenerateGPTResponse(input string, gpt *openai.Client) string {

	resp, err := gpt.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: input,
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return ""
	}
	if err != nil {
		fmt.Println("Failed to generate GPT response:", err)
		return ""
	}
	return resp.Choices[0].Message.Content
}

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:store.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Initialize OpenAI GPT
	gpt := openai.NewClient(OpenAIAPIKey)
	if err != nil {
		panic(err)
	}

	client.AddEventHandler(GetEventHandler(client, gpt))

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
