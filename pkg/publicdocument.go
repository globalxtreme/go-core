package xtremepkg

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type PublicDocumentPayload struct {
	Blueprint string `json:"blueprint"`
	Reference string `json:"reference"`
	Payload   any    `json:"payload"`
}

type PublicDocument interface {
	GenerateLink(docType string, payload *PublicDocumentPayload) (string, error)
	Encrypt(payload any) (string, error)
	Decrypt(token string, payload *PublicDocumentPayload) error
}

func NewPublicDocument() PublicDocument {
	return &publicDocument{
		baseURL:    os.Getenv("PUBLIC_DOCUMENT_BASE_URL"),
		encryptKey: []byte(os.Getenv("PUBLIC_DOCUMENT_ENCRYPT_KEY")),
		signKey:    []byte(os.Getenv("PUBLIC_DOCUMENT_SIGN_KEY")),
	}
}

type publicDocument struct {
	baseURL    string
	encryptKey []byte
	signKey    []byte
}

func (ec *publicDocument) GenerateLink(docType string, payload *PublicDocumentPayload) (string, error) {
	token, err := ec.Encrypt(payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", ec.baseURL, docType, token), nil
}

func (ec *publicDocument) Encrypt(payload any) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload error: %w", err)
	}

	block, err := aes.NewCipher(ec.encryptKey)
	if err != nil {
		return "", fmt.Errorf("aes cipher error: %w", err)
	}

	plainBytes := ec.pkcs7Pad(jsonPayload, aes.BlockSize)

	ciphertext := make([]byte, aes.BlockSize+len(plainBytes))

	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("IV error: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plainBytes)

	encodedData := base64.RawURLEncoding.EncodeToString(ciphertext)

	signature := ec.calculateHMAC(encodedData)
	token := encodedData + "." + signature

	encryptedToken := base64.RawURLEncoding.EncodeToString([]byte(token))

	return encryptedToken, nil
}

func (ec *publicDocument) Decrypt(token string, payload *PublicDocumentPayload) error {
	tokenBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("base64 decode token error: %w", err)
	}

	decodedToken := string(tokenBytes)

	parts := strings.Split(decodedToken, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid token format")
	}

	encodedData := parts[0]
	providedSig := parts[1]

	expectedSig := ec.calculateHMAC(encodedData)

	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return fmt.Errorf("invalid signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(encodedData)
	if err != nil {
		return fmt.Errorf("base64 decode error: %w", err)
	}

	if len(data) < aes.BlockSize {
		return fmt.Errorf("invalid data length")
	}

	iv := data[:aes.BlockSize]
	actualCiphertext := data[aes.BlockSize:]

	if len(actualCiphertext)%aes.BlockSize != 0 {
		return fmt.Errorf("invalid ciphertext block size")
	}

	block, err := aes.NewCipher(ec.encryptKey)
	if err != nil {
		return fmt.Errorf("aes cipher error: %w", err)
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(actualCiphertext, actualCiphertext)

	plaintext, err := ec.pkcs7Unpad(actualCiphertext, aes.BlockSize)
	if err != nil {
		return fmt.Errorf("unpad error: %w", err)
	}

	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	return nil
}

/** --- UNEXPORTED FUNCTIONS --- */

func (ec *publicDocument) calculateHMAC(data string) string {
	h := hmac.New(sha256.New, ec.signKey)
	h.Write([]byte(data))

	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (ec *publicDocument) pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	pad := bytes.Repeat([]byte{byte(padding)}, padding)

	return append(data, pad...)
}

func (ec *publicDocument) pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 || length%blockSize != 0 {
		return nil, fmt.Errorf("invalid padded data")
	}

	padding := int(data[length-1])
	if padding == 0 || padding > blockSize || padding > length {
		return nil, fmt.Errorf("invalid padding")
	}

	for i := 0; i < padding; i++ {
		if data[length-1-i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding sequence")
		}
	}

	return data[:length-padding], nil
}
