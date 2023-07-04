package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	reflect "reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"github.com/mzbaulhaque/gois/pkg/scraper/services"
	openai "github.com/sashabaranov/go-openai"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
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
	gmail_password       = "GMAIL_PASSWORD"
	gmail_email          = "GMAIL_EMAIL"
	manager_email        = "MANAGER_EMAIL"
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
	if len(strings.Fields(response.GeneratedText)) > 0 {
		return response.GeneratedText, nil
	}

	return "", fmt.Errorf("no response from Hugging Face")
}
func GetImageBytes(url string) ([]byte, string, error) {
	// Send a GET request to the URL
	response, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()

	// Check if the response status code is OK
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP request failed with status code %d", response.StatusCode)
	}

	// Read the response body
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}

	// Check if the URL points to an image
	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("URL does not point to an image")
	}

	return bodyBytes, contentType, nil
}

// GetMaxThreeURLs retrieves a maximum of three URLs from a list of interfaces
func GetMaxThreeURLs(items []interface{}) []string {
	var urls []string
	maxItems := 3

	for _, item := range items {
		if len(urls) >= maxItems {
			break
		}

		// Assuming each item is a map with a "URL" key
		if data, ok := item.(map[string]interface{}); ok {
			if url, ok := data["URL"].(string); ok {
				urls = append(urls, url)
			}
		}
	}

	return urls
}

// analyzeCSVData analyzes the CSV data using ChatGPT 3.5 and returns a summary
func analyzeCSVData(csvData string, gpt *openai.Client, command string) (string, error) {
	prompt := fmt.Sprintf("CSV Data: \n"+csvData+"\n %s", command)

	resp, err := gpt.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Name:    "Analyser",
				Content: "you are the best csv data analysis",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Name:    "Analyser",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		output := resp.Choices[0].Message.Content
		return output, nil
	}
	return "", nil
}
func ConvertToFlickrResult(data interface{}) (services.GoogleResult, bool) {
	result := services.GoogleResult{}

	// Use reflection to access the fields of the interface
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling interface:", err)
		return result, false
	}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return result, false
	}
	return result, true
}
func PrintInterface(i interface{}) {
	interfaceType := reflect.TypeOf(i)
	interfaceValue := reflect.ValueOf(i)

	fmt.Println("Interface Type:", interfaceType)

	if interfaceType.Kind() != reflect.Func {
		fmt.Println("Interface Value:", interfaceValue)
		return
	}

	fmt.Println("Arguments:")
	for i := 0; i < interfaceType.NumIn(); i++ {
		argType := interfaceType.In(i)
		fmt.Println(" -", argType)
	}
}
func GetImageBytecodeAndMIMEType(filePath string) ([]byte, string, error) {
	// Read the image file
	imageBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}

	// Determine the MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))

	return imageBytes, mimeType, nil
}

// for openai chatgpt

// var block_peoples = []string{"212709251456@s.whatsapp.net"}
var block_peoples = []string{"__"}
// var allowed_groups = []string{"120363159995578517@g.us"}
var allowed_groups = []string{"120363143651964565@g.us"}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

// GetTextFormatFromCSV converts CSV file bytes to text format and removes empty rows
func GetTextFormatFromCSV(csvData []byte) string {
	reader := csv.NewReader(bytes.NewReader(csvData))
	lines, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	var output strings.Builder
	for _, line := range lines {
		// Skip empty rows
		if len(line) == 0 {
			continue
		}

		// Remove empty cells in a row
		var nonEmptyCells []string
		for _, cell := range line {
			if cell != "" {
				nonEmptyCells = append(nonEmptyCells, cell)
			}
		}

		// Append non-empty cells in a row
		if len(nonEmptyCells) > 0 {
			output.WriteString(strings.Join(nonEmptyCells, ","))
			output.WriteString("\n")
		}
	}

	return output.String()
}

var _req = map[string]openai.ChatCompletionRequest{}

func GetEventHandler(client *whatsmeow.Client, gpt *openai.Client) func(interface{}) {
	questions := []string{
		"How can I assist you today?",
		"What specific information are you looking for?",
		"Is there a particular feature you need help with?",
		"Do you have any technical issues that need troubleshooting?",
	}

	// Create a button template with the predefined questions
	QuickReplybuttons := make([]*waProto.HydratedTemplateButton, len(questions))
	QuickReplybuttons_ := make([]*waProto.ButtonsMessage_Button, len(questions))
	for i, question := range questions {
		QuickReplybuttons_[i] = &waProto.ButtonsMessage_Button{
			ButtonId: proto.String(strconv.Itoa(i)),
			ButtonText: &waProto.ButtonsMessage_Button_ButtonText{
				DisplayText: proto.String(question),
			},
		}
	}
	for i, question := range questions {
		QuickReplybuttons[i] = &waProto.HydratedTemplateButton{
			Index: proto.Uint32(uint32(i)),
			HydratedButton: &waProto.HydratedTemplateButton_QuickReplyButton{
				QuickReplyButton: &waProto.HydratedTemplateButton_HydratedQuickReplyButton{
					DisplayText: proto.String(question),
					Id:          proto.String(question),
				},
			},
		}
	}
	hydratedCallButton := &waProto.HydratedTemplateButton{
		Index: proto.Uint32(uint32(10)),
		HydratedButton: &waProto.HydratedTemplateButton_CallButton{
			CallButton: &waProto.HydratedTemplateButton_HydratedCallButton{
				DisplayText: proto.String("Call US"),
				PhoneNumber: proto.String("+212709251456"),
			},
		},
	}
	QuickReplybuttons = append(QuickReplybuttons, hydratedCallButton)
	// buttons_title := "Please select one of the following questions:"
	// hydratedFourRowTemplate := waProto.TemplateMessage_HydratedFourRowTemplate{
	// 	HydratedContentText: proto.String("الآن العرض الجديد"),
	// 	HydratedFooterText:  proto.String("تطبّق الشروط والأحكام"),
	// 	HydratedButtons:     QuickReplybuttons,
	// 	TemplateId:          proto.String("id1"),
	// 	Title: &waProto.TemplateMessage_HydratedFourRowTemplate_HydratedTitleText{
	// 		HydratedTitleText: buttons_title,
	// 	},
	// }
	// templateMessage := waProto.TemplateMessage{
	// 	// ContextInfo:      &waProto.ContextInfo{},
	// 	HydratedTemplate: &hydratedFourRowTemplate,
	// 	TemplateId:       proto.String("kom"),
	// 	Format:           &waProto.TemplateMessage_FourRowTemplate_{},
	// }
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.LoggedOut:
			err := client.Connect()
			if err != nil {
				panic(err)
			}
		case *events.Message:
			var messageBody = v.Message.GetConversation()
			fmt.Println("Message event:", v.Message.GetConversation(), v.Info.Type)
			client.MarkRead([]string{v.Info.ID}, time.Now(), v.Info.Chat, v.Info.Sender)
			switch {
			case v.IsDocumentWithCaption:
				DocumentWithCaption := v.Message.DocumentMessage
				if bytes, _error := client.Download(DocumentWithCaption); _error == nil {
					switch DocumentWithCaption.GetMimetype() {
					case "text/csv":
						csvfile := GetTextFormatFromCSV(bytes)
						if res, err := analyzeCSVData(csvfile, gpt, DocumentWithCaption.GetCaption()); err == nil {
							client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
								Conversation: proto.String(res),
							})
						} else {
							fmt.Printf("analyzeCSVData: %v", err)
						}
					default:
						client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
							Conversation: proto.String("File format is not implimented yet!"),
						})
					}
				}
			case v.Info.Type == "media" && !v.IsDocumentWithCaption:
				client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
					Conversation: proto.String("File format is not implimented yet!"),
				})
			case strings.ToLower(messageBody) == "ping":
				client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
					Conversation: proto.String("pong"),
				})
			case strings.HasPrefix(strings.ToLower(messageBody), "/reset"):
				if _, ok := _req[v.Info.Sender.String()]; ok {
					var _allmessages []openai.ChatCompletionMessage = []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleSystem,
							Content: "you are a helpful personal assistant",
						},
					}
					_req[v.Info.Sender.String()] = openai.ChatCompletionRequest{
						Model:    openai.GPT3Dot5Turbo,
						Messages: _allmessages,
					}
				}
			case strings.HasPrefix(strings.ToLower(messageBody), "/new"):
				args := strings.Fields(messageBody)[1:]
				the_rest := strings.Join(args, " ")
				if the_rest == "" {
					var _allmessages []openai.ChatCompletionMessage = []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleSystem,
							Content: "you are a helpful personal assistant",
						},
					}
					_req[v.Info.Sender.String()] = openai.ChatCompletionRequest{
						Model:    openai.GPT3Dot5Turbo,
						Messages: _allmessages,
					}
				} else {
					var _allmessages []openai.ChatCompletionMessage = []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleSystem,
							Content: the_rest,
						},
					}
					_req[v.Info.Sender.String()] = openai.ChatCompletionRequest{
						Model:    openai.GPT3Dot5Turbo,
						Messages: _allmessages,
					}
				}
			case strings.HasPrefix(strings.ToLower(messageBody), "/set_group_name"):
				args := strings.Fields(messageBody)[1:]
				name := strings.Join(args, " ")
				if irr := client.SetGroupName(v.Info.Chat, name); irr != nil {
					_, err := client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						Conversation: proto.String(irr.Error()),
					})
					if err != nil {
						fmt.Printf("ImageMessage error: %v\n", err)
					}
				}
			case strings.HasPrefix(strings.ToLower(messageBody), "/image"):
				args := strings.Fields(messageBody)[1:]
				query := strings.Join(args, " ")
				fmt.Printf("query: %s", query)
				// if up, err := client.Upload(context.Background(), bytedata, whatsmeow.MediaImage); err != nil {
				// 	return nil, err
				//   } else {

				// 	var message = &waProto.ImageMessage{
				// 	  Url:           &up.URL,
				// 	  Mimetype:      proto.String(mimetype),
				// 	  Caption:       proto.String("Caption"),
				// 	  FileSha256:    up.FileSHA256,
				// 	  FileEncSha256: up.FileEncSHA256,
				// 	  FileLength:    &up.FileLength,
				// 	  MediaKey:      up.MediaKey,
				// 	  DirectPath:    &up.DirectPath,
				// 	}
				//   }
				config := &services.GoogleConfig{
					Query: query,
				}
				gs := &services.GoogleScraper{Config: config}
				items, _, _err := gs.Scrape()
				if _err != nil {
					fmt.Printf("ImageMessage error: %v\n", _err)
					client.SendPresence(types.PresenceAvailable)
					_, err := client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
						Conversation: proto.String("images not found!"),
					})
					if err != nil {
						fmt.Printf("ImageMessage error: %v\n", err)
					}
				}
				var convertedItems = make([]services.GoogleResult, 4)
				for i, item := range items {
					if i <= 3 {
						data, ok := ConvertToFlickrResult(item)
						if ok {
							convertedItems[i] = data
						}
					} else {
						break
					}
				}
				fmt.Print(convertedItems)
				if len(convertedItems) > 0 {
					for i := 0; i < len(convertedItems); i++ {
						if convertedItems[i].URL != "" {
							bytedata, mimeType, __err := GetImageBytes(convertedItems[i].URL)
							if __err != nil {
								fmt.Printf("ImageMessage error: %v\n", __err)
								return
							} else {
								if up, err := client.Upload(context.Background(), bytedata, whatsmeow.MediaImage); err == nil {
									var message = &waProto.ImageMessage{
										Url:           &up.URL,
										Mimetype:      proto.String(mimeType),
										Caption:       proto.String(convertedItems[i].Title),
										FileSha256:    up.FileSHA256,
										FileEncSha256: up.FileEncSHA256,
										FileLength:    &up.FileLength,
										MediaKey:      up.MediaKey,
										DirectPath:    &up.DirectPath,
									}
									_, err := client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
										ImageMessage: message,
									})
									if err != nil {
										fmt.Printf("ImageMessage error: %v\n", err)
										return
									}
								} else {
									fmt.Printf("Upload error: %v", err)
								}
							}
						}
					}

				}
			default:
					if !contains(block_peoples, v.Info.Sender.String()) && contains(allowed_groups, v.Info.Chat.String()) {
						response, err := GenerateGPTResponse(messageBody, v.Info.Sender.String(), gpt)
						// // response, err := GetHuggingFaceResponse(messageBody)
						if err != nil {
							fmt.Printf("ChatCompletion error: %v\n", err)
							return
						}
						if len(response) > 0 {
							// Create a buttons message.
							client.SendPresence(types.PresenceAvailable)
							_, err := client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
								Conversation: proto.String(response),
							})
							if err != nil {
								fmt.Printf("ERROR Message: %v", err)
							}
						}
					}
				}
			}
		}
	}
}

func GenerateGPTResponse(input string, user string, gpt *openai.Client) (string, error) {
	var _allmessages []openai.ChatCompletionMessage
	if _, ok := _req[user]; !ok {

		_allmessages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "you are a helpful personal assistant",
			},
		}
	}
	_allmessages = append(_allmessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input,
	})
	_req[user] = openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: _allmessages,
	}
	resp, err := gpt.CreateChatCompletion(
		context.Background(),
		_req[user],
	)
	if err != nil {
		return "fails!!!", fmt.Errorf("chatCompletion error: %v", err)
	}
	_allmessages = append(_allmessages, resp.Choices[0].Message)
	_req[user] = openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: _allmessages,
	}
	return resp.Choices[0].Message.Content, nil
}

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
				if err == nil {
					err_email := sendQREmail("", evt.Code)
					if err_email != nil {
						fmt.Printf("error: %v", err_email)
					}
				}
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
