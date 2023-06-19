package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
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

type WhatsappClient struct {
	client *whatsmeow.Client // Assuming you have a Client struct defined
}

var WhatsappCl = WhatsappClient{}

const (
	OpenAIAPIKeyEnvVar   = "OPENAI_API_KEY"
	HuggingfaceKeyEnvVar = "HUGGINGFACE_API_KEY"
)

type HuggingFaceResponse struct {
	GeneratedText string `json:"generated_text"`
	conversation  struct {
		generated_responses []string `json:"generated_responses"`
		past_user_inputs    []string `json:"past_user_inputs"`
	} `json:"conversation"`
	warnings []string `json:"warnings"`
}

func GetHuggingFaceResponse(prompt string) (string, error) {
	apiKey := os.Getenv(HuggingfaceKeyEnvVar)
	url := "https://api-inference.huggingface.co/models/microsoft/DialoGPT-medium"
	requestBody, err := json.Marshal(map[string]string{
		"inputs": prompt,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var response HuggingFaceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}
	println("response.generated_text:", response.GeneratedText)
	if len(strings.Fields(response.GeneratedText)) > 0 {
		return response.GeneratedText, nil
	}

	return "", fmt.Errorf("no response from Hugging Face")
}

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
				// } else if strings.HasPrefix(messageBody, "complete:") {
			} else {
				// Extract the command arguments
				// args := strings.Fields(messageBody)[1:]
				// Join the arguments to form the input message for GPT
				// input := strings.Join(args, " ")
				response, err := GenerateGPTResponse(messageBody+", respond in 90 chars only or less", gpt)
				// response, err := GetHuggingFaceResponse(messageBody)
				if err != nil {
					fmt.Printf("ChatCompletion error: %v\n", err)
					return
				}
				if len(response) > 0 {
					client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						Conversation: proto.String(response),
					})
				}
			}
		}
	}
}

func GenerateGPTResponse(input string, gpt *openai.Client) (string, error) {

	resp, err := gpt.CreateCompletion(
		context.Background(),
		openai.CompletionRequest{
			Model:     openai.GPT3TextDavinci003,
			MaxTokens: 150,
			Prompt:    input,
		},
	)
	if err != nil {
		return "nil", fmt.Errorf("chatCompletion error: %v", err)
	}
	if err != nil {
		return "nil", fmt.Errorf("failed to generate GPT response: %v", err)
	}
	return resp.Choices[0].Text, nil
}

// func getpdfchatbot() {
// 	// todo
// }
func main() {
	var wg sync.WaitGroup
	err := godotenv.Load()
	if err != nil {
		panic("Failed to load .env file")
	}
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
	WhatsappCl.client = whatsmeow.NewClient(deviceStore, clientLog)
	// Initialize OpenAI GPT
	openaiAPIKey := os.Getenv(OpenAIAPIKeyEnvVar)
	gpt := openai.NewClient(openaiAPIKey)
	if err != nil {
		panic(err)
	}

	WhatsappCl.client.AddEventHandler(GetEventHandler(WhatsappCl.client, gpt))

	if WhatsappCl.client.Store.ID == nil {
		qrChan, _ := WhatsappCl.client.GetQRChannel(context.Background())
		err = WhatsappCl.client.Connect()
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
		err = WhatsappCl.client.Connect()
		if err != nil {
			panic(err)
		}
	}
	run_api(&wg)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	WhatsappCl.client.Disconnect()
}
