package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	mathrand "math/rand"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
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

// Paths for public and private key files
const (
	privateKeyPath = "private_key.pem"
	publicKeyPath  = "public_key.pem"
)

// RSA key pair
var (
	rsaKeyPair  *rsa.PrivateKey
	_keymanager *APIKeyManager
)

func run_api(wg *sync.WaitGroup) {
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
	// init the api keys manager
	_keymanager, _ = init_apikeymanager()
	println(_keymanager.GenerateAPIKey("kimo", time.Now().AddDate(0, 12, 0)))
	println(_keymanager.GenerateAPIKey("baddi", time.Now().AddDate(0, 12, 0)))
	// Create a new Gin router
	router := gin.Default()

	// Define the API endpoint with API key authentication
	router.POST("/send-message", authenticate, sendMessage)

	// Define the root route
	router.GET("/", mainHandler)
	useAdmin(router)
	// Start the server on port 8385
	log.Fatal(router.Run(":8385"))
}

// Middleware function to authenticate API key
func authenticate(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	valid, __err := _keymanager.ValidateAPIKey(apiKey)
	if __err != nil || !valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid API key: %v", __err)})
	}

	// Call the next handler
	c.Next()
}

// Handler function for sending the message
func sendMessage(c *gin.Context) {
	var request MessageRequest

	// Parse the request body
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if the message is within the allowed limit
	if len(request.Message) > 250 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message exceeds the maximum length of 250 characters"})
		return
	}

	// Validate phone numbers
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	for _, number := range request.Numbers {
		if !phoneRegex.MatchString(number) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid phone number: %s", number)})
			return
		}
	}

	// Send the message to the phone numbers (you can implement your own logic here)
	// ...

	// Call the sendThem function if all validations pass
	sendThem(request.Numbers, request.Message, c)
}
func sendThem(numbers []string, message string, c *gin.Context) bool {
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
			c.JSON(http.StatusBadRequest, response)
		}
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{Reasons: []string{"WhatsApp client not connected!"}})
	}
	return true
}

func mainHandler(c *gin.Context) {
	// Read the content of the doc.md file
	content, err := ioutil.ReadFile("doc.md")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read doc.md file")
		return
	}

	// Convert Markdown content to HTML
	htmlContent := blackfriday.Run(content)

	// Write the HTML content as the response
	c.Data(http.StatusOK, "text/html", htmlContent)
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
