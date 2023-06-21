package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type APIKey struct {
	ID        uint `gorm:"primaryKey"`
	Key       string
	Deadline  time.Time
	Details   string
	signed    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type APIKeyManager struct {
	db         *gorm.DB
	privateKey *rsa.PrivateKey
}

func NewAPIKeyManager(db *gorm.DB, privateKeyPath string) (*APIKeyManager, error) {
	privateKeyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, err
	}

	return &APIKeyManager{
		db:         db,
		privateKey: privateKey,
	}, nil
}

func (m *APIKeyManager) RemoveAPIKey(key string) error {
	return m.db.Where("key = ?", key).Delete(&APIKey{}).Error
}

func (m *APIKeyManager) ValidateAPIKey(tokenString string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return &m.privateKey.PublicKey, nil
	})

	if err != nil {
		return false, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		key := claims["key"].(string)
		var apiKey APIKey
		if err := m.db.Where("key = ?", key).First(&apiKey).Error; err != nil {
			return false, err
		}
		err = token.Claims.Valid()
		if err != nil {
			// Handle the expiration error
			return false, err
		}
		return true, nil
	}

	return false, fmt.Errorf("invalid token")
}
func (m *APIKeyManager) EditAPIKey(tokenString string, newDeadline time.Time) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		var apiKey APIKey
		claims["exp"] = newDeadline.Unix()
		key := claims["key"].(string)

		if err := m.db.Where("key = ?", key).First(&apiKey).Error; err != nil {
			return "", err
		}

		apiKey.Deadline = newDeadline
		apiKey.UpdatedAt = time.Now()
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signedToken, err := token.SignedString(m.privateKey)
		if err != nil {
			return "", err
		}
		apiKey.signed = signedToken
		if err := m.db.Save(&apiKey).Error; err != nil {
			return "", err
		}
		return signedToken, nil
	}

	return "", fmt.Errorf("invalid token claims")
}
func (m *APIKeyManager) GenerateAPIKey(details string, deadline time.Time) (string, error) {
	var apiKey APIKey
	key := generateRandomKey()
	claims := jwt.MapClaims{
		"key":     key,
		"details": details,
		"exp":     deadline.Unix(),
	}
	apiKey.Deadline = deadline
	apiKey.UpdatedAt = time.Now()
	apiKey.Details = details
	apiKey.Key = key

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", err
	}
	apiKey.signed = signedToken
	if err := m.db.Save(&apiKey).Error; err != nil {
		return "", err
	}
	return signedToken, nil
}

func generateRandomKey() string {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		// Handle the error appropriately
		panic(err)
	}

	return fmt.Sprintf("%x", key)
}
func init_apikeymanager() (*APIKeyManager, error) {
	db, err := gorm.Open(sqlite.Open("api_keys.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Auto-migrate the APIKey model
	if err := db.AutoMigrate(&APIKey{}); err != nil {
		log.Fatal(err)
	}

	// Create a new APIKeyManager
	return NewAPIKeyManager(db, privateKeyPath)
}

// func testapimanager() {
// 	// Connect to the database
// 	db, err := gorm.Open(sqlite.Open("api_keys.db"), &gorm.Config{})
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Auto-migrate the APIKey model
// 	if err := db.AutoMigrate(&APIKey{}); err != nil {
// 		log.Fatal(err)
// 	}

// 	// Create a new APIKeyManager
// 	apiKeyManager := NewAPIKeyManager(db)

// 	// Generate a new API key
// 	apiKey, err := apiKeyManager.GenerateAPIKey("Key Details", time.Now().Add(24*time.Hour))
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Printf("Generated API Key: %s\n", apiKey.Key)

// 	// Edit the API key deadline
// 	if err := apiKeyManager.EditAPIKey(apiKey.Key, time.Now().Add(48*time.Hour)); err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println("API Key deadline edited successfully")

// 	// Remove the API key
// 	if err := apiKeyManager.RemoveAPIKey(apiKey.Key); err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println("API Key removed successfully")

// 	// Validate an API key token
// 	token := "your-token" // Replace with a valid token
// 	valid, err := apiKeyManager.ValidateAPIKey(token)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	if valid {
// 		fmt.Println("API Key token is valid")
// 	} else {
// 		fmt.Println("API Key token is invalid")
// 	}
// }
