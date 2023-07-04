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
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/russross/blackfriday/v2"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Struct to represent the request body
type MessageRequest struct {
	Numbers []string         `form:"numbers"`
	Message string           `form:"message"`
	File    []multipart.File `form:"file"`
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
	// gin.SetMode(gin.ReleaseMode)
	// gin.DefaultWriter = ioutil.Discard
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
	// println(_keymanager.GenerateAPIKey("kimo", time.Now().AddDate(0, 12, 0)))
	// println(_keymanager.GenerateAPIKey("baddi", time.Now().AddDate(0, 12, 0)))
	// Create a new Gin router
	router := gin.Default()

	// Define the API endpoint with API key authentication
	router.POST("/send-message", authenticate, sendMessage)
	router.POST("/keygen", genkey)

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
func genkey(c *gin.Context) {
	// Read the Protobuf message from the request body or any other source
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Create an instance of the KeyApiProto message
	keyApi := &KeyApiProto{}

	// Unmarshal the Protobuf data into the KeyApiProto message
	err = proto.Unmarshal(data, keyApi)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to unmarshal Protobuf data"})
		return
	}
	if len(keyApi.GetSignedKey()) > 0 {
		valid, __err := _keymanager.ValidateNewAPIKey(keyApi.GetSignedKey())
		if __err != nil || !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid API key: %v", __err)})
		}
		c.JSON(http.StatusOK, gin.H{"message": "Key API generated successfully"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to unmarshal Protobuf data"})
	}
	// Perform the necessary operations to generate the key API
	// ...

	// Return the response or perform any other actions

}

// Handler function for sending the message
func sendMessage(c *gin.Context) {

	// Parse the form data
	err := c.Request.ParseMultipartForm(32 << 20) // 32MB max size
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	// Extract the message field
	message := c.Request.FormValue("message")
	if len(message) > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message exceeds the maximum length of 600 characters"})
		return
	}

	// Extract the numbers field
	var numbers []string = []string{}
	if len(c.Request.Form["numbers"]) > 0 {
		numbers = strings.Fields(c.Request.Form["numbers"][0])
	}
	if len(numbers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Numbers field is required"})
		return
	}

	// Handle the file field
	files := c.Request.MultipartForm.File["file"]

	var fileBytes []byte = []byte{}
	var mimeType string = ""
	var filename string = ""
	var isfile bool = false
	if len(files) > 0 {
		filename = files[0].Filename
		// Only process the first file in the slice
		file, err := files[0].Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read the file"})
			return
		}
		defer file.Close()

		// Read the file bytes
		fileBytes, err = ioutil.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read the file"})
			return
		}

		// Get the file's MIME type
		mimeType = files[0].Header.Get("Content-Type")

		// Check if the file type is allowed
		if !isAllowedFileType(mimeType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed file types are: csv, pdf, docx, doc, xlsx, xls, png, jpeg, jpg, gif"})
			return
		}
		isfile = true
	}
	// Send the message to the phone numbers (you can implement your own logic here)
	// ...

	// Call the sendThem function if all validations pass
	done := sendThem(numbers, message, isfile, fileBytes, filename, mimeType, c)
	if done {
		c.JSON(http.StatusOK, gin.H{
			"message": "Success!",
		})
	}
}
func isAllowedFileType(mimeType string) bool {
	allowedTypes := []string{
		"application/pdf",
		"text/csv",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-excel",
		"image/png",
		"image/jpeg",
		"image/jpg",
		"image/gif",
	}

	for _, allowedType := range allowedTypes {
		if strings.EqualFold(mimeType, allowedType) {
			return true
		}
	}

	return false
}
func isImage(mimeType string) bool {
	allowedTypes := []string{
		"image/png",
		"image/jpeg",
		"image/jpg",
		"image/gif",
	}

	for _, allowedType := range allowedTypes {
		if strings.EqualFold(mimeType, allowedType) {
			return true
		}
	}

	return false
}
func sendThem(numbers []string, message string, isfile bool, fileBytes []byte, filename string, mimitype string, c *gin.Context) bool {
	if WhatsappCl.client != nil && WhatsappCl.client.IsConnected() {
		var sendedMsgErr []string
		for i, v := range numbers {
			fmt.Printf("Number: %s, isfile: %v, filesize: %d, message: %s\n", v, isfile, len(fileBytes), message)
			randomprim := mathrand.Perm(8)[0]
			var Millisecond time.Duration = time.Duration(randomprim * 100000000)
			var _message waProto.Message
			switch {
			case isfile && len(fileBytes) > 0:
				if isImage(mimitype) {
					if up, err := WhatsappCl.client.Upload(context.Background(), fileBytes, whatsmeow.MediaImage); err == nil {
						_imagemessage := &waProto.ImageMessage{
							Url:           &up.URL,
							Mimetype:      proto.String(mimitype),
							Caption:       proto.String(message),
							FileSha256:    up.FileSHA256,
							FileEncSha256: up.FileEncSHA256,
							FileLength:    &up.FileLength,
							MediaKey:      up.MediaKey,
							DirectPath:    &up.DirectPath,
						}
						_message = waProto.Message{
							ImageMessage: _imagemessage,
						}
					}
				} else {
					if up, err := WhatsappCl.client.Upload(context.Background(), fileBytes, whatsmeow.MediaDocument); err == nil {
						_documentmessage := &waProto.DocumentMessage{
							Url:           &up.URL,
							Mimetype:      proto.String(mimitype),
							Caption:       proto.String(message),
							FileSha256:    up.FileSHA256,
							FileName:      proto.String(filename),
							FileEncSha256: up.FileEncSHA256,
							FileLength:    &up.FileLength,
							MediaKey:      up.MediaKey,
							DirectPath:    &up.DirectPath,
						}
						_message = waProto.Message{
							DocumentMessage: _documentmessage,
						}
					}

				}
			default:
				_message = waProto.Message{
					Conversation: proto.String(message),
				}
			}
			_, err := WhatsappCl.client.SendMessage(
				context.Background(),
				types.NewJID(v, "s.whatsapp.net"),
				&_message,
			)
			if err != nil {
				sendedMsgErr = append(sendedMsgErr, fmt.Errorf("%v-%v: (%v)", err, i, v).Error())
			}
			fmt.Println("Number: %s, isfile: %v, filesize: %d\n", v, isfile, len(fileBytes))
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
