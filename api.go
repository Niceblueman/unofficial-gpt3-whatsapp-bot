package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	mathrand "math/rand"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/russross/blackfriday/v2"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Struct to represent the request body
type MessageRequest struct {
	Numbers []string `json:"numbers"`
	Message string   `json:"message"`
}

// Struct to represent the response
type MessageResponse struct {
	Message string `json:"message"`
}

// Struct to represent the response
type ErrorResponse struct {
	Reasons []string `json:"Reasons"`
}

// Map to store API keys and their corresponding messages
var apiKeyMap map[string]string

// Paths for public and private key files
const (
	privateKeyPath = "private_key.pem"
	publicKeyPath  = "public_key.pem"
)

// RSA key pair
var rsaKeyPair *rsa.PrivateKey

func run_api(wg *sync.WaitGroup) {
	apiKeyMap = make(map[string]string)

	// Check if the private key file exists
	if privateKeyExists() {
		// Load the existing private key
		privateKey, err := loadPrivateKey()
		if err != nil {
			log.Fatal(err)
		}
		rsaKeyPair = privateKey
	} else {
		// Generate a new private key
		privateKey, err := generatePrivateKey()
		if err != nil {
			log.Fatal(err)
		}
		rsaKeyPair = privateKey

		// Save the private key to a file
		err = savePrivateKey(privateKey)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Print three new API keys during the first run
	for i := 0; i < 100; i++ {
		apiKey, err := generateAPIKey()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("New API Key:", apiKey)
	}

	// Create a new router
	router := mux.NewRouter()

	// Define the API endpoint with API key authentication
	router.HandleFunc("/send-message", authenticate(sendMessage)).Methods("POST")
	router.HandleFunc("/", mainHandler).Methods("GET")

	// Redirect all unknown routes to /main
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})
	// Start the server on port 8385
	log.Fatal(http.ListenAndServe(":8385", router))
}

// Middleware function to authenticate API key
func authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")

		if apiKey == "" || !isValidAPIKey(apiKey) {
			println("apiKey %v", apiKey)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Invalid API key")
			return
		}

		// Call the next handler
		next(w, r)
	}
}

// Handler function for sending the message
func sendMessage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Numbers []string `json:"numbers"`
		Message string   `json:"message"`
	}

	// Parse the request body
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid request body")
		return
	}

	// Check if the message is within the allowed limit
	if len(request.Message) > 250 {
		fmt.Fprint(w, "Message exceeds the maximum length of 250 characters")
		return
	}

	// Validate phone numbers
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	for _, number := range request.Numbers {
		if !phoneRegex.MatchString(number) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid phone number: %s", number)
			return
		}
	}

	// Send the message to the phone numbers (you can implement your own logic here)
	// ...

	// Call the send_them function if all validations pass
	sendThem(request.Numbers, request.Message, w)
}
func sendThem(numbers []string, message string, w http.ResponseWriter) bool {
	if WhatsappCl.client != nil && WhatsappCl.client.IsConnected() {
		var sendedMsgErr []string
		for i, v := range numbers {
			randomprim := mathrand.Perm(8)[0]
			var Millisecond time.Duration = time.Duration(randomprim * 100000000)
			_, err := WhatsappCl.client.SendMessage(
				context.Background(),
				types.NewJID(v, "s.whatsapp.net"),
				&waProto.Message{
					Conversation: proto.String(message),
				},
			)
			if err != nil {
				sendedMsgErr = append(sendedMsgErr, fmt.Errorf("%v-%v: (%v)", err, i, v).Error())
			}
			time.Sleep(Millisecond)
		}
		if len(sendedMsgErr) > 0 {
			response := ErrorResponse{
				Reasons: sendedMsgErr,
			}
			w.WriteHeader(http.StatusBadRequest)
			// Send the response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		response := ErrorResponse{
			Reasons: []string{"whatsapp clinet not connected!"},
		}
		// Send the response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
	return true
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	// Read the content of the doc.md file
	content, err := ioutil.ReadFile("doc.md")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Failed to read doc.md file")
		return
	}

	// Convert Markdown content to HTML
	htmlContent := blackfriday.Run(content)

	// Write the HTML content as the response
	w.Header().Set("Content-Type", "text/html")
	w.Write(htmlContent)
}

// Function to generate a private key
func generatePrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// Function to save the private key to a file
func savePrivateKey(privateKey *rsa.PrivateKey) error {
	der := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	}
	pemData := pem.EncodeToMemory(block)
	return ioutil.WriteFile(privateKeyPath, pemData, 0644)
}

// Function to load the private key from a file
func loadPrivateKey() (*rsa.PrivateKey, error) {
	pemData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemData)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// Function to check if the private key file exists
func privateKeyExists() bool {
	_, err := ioutil.ReadFile(privateKeyPath)
	return err == nil
}

// Function to generate a new API key
func generateAPIKey() (string, error) {
	// Create a new token object with some claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(), // set expiration time
		"iat": time.Now().Unix(),                          // set issued at time
	})

	// Generate a symmetric key by hashing the base64-encoded modulus of the RSA public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&rsaKeyPair.PublicKey)
	if err != nil {
		return "", err
	}
	symmetricKey := sha256.Sum256(publicKeyBytes)

	// Sign and get the complete encoded token as a string using the secret
	apiKey, err := token.SignedString(symmetricKey[:])
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

// Function to validate the API key using public and private key logic
func isValidAPIKey(apiKey string) bool {
	// Generate a symmetric key by hashing the base64-encoded modulus of the RSA public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&rsaKeyPair.PublicKey)
	if err != nil {
		return false
	}
	symmetricKey := sha256.Sum256(publicKeyBytes)

	// Parse and validate the token using the secret
	token, err := jwt.Parse(apiKey, func(token *jwt.Token) (interface{}, error) {
		return symmetricKey[:], nil
	})
	if err != nil {
		return false
	}

	return token.Valid // return true if valid, false otherwise
}
