package xtremepkg

import (
	xtremeres "github.com/globalxtreme/go-core/response"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func RandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randomBytes := make([]byte, length)
	for i := 0; i < length; i++ {
		randomBytes[i] = chars[rand.Intn(len(chars))]
	}

	return string(randomBytes) + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func CheckAndCreateDirectory(path string) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(path, os.ModePerm)
		}
	}
}

func SetStorageDir(path ...string) string {
	storagePath := os.Getenv("STORAGE_DIR")
	if len(storagePath) == 0 {
		storagePath = "storages"
	}

	if len(path) > 0 {
		storagePath += "/" + path[0]
	}

	return storagePath
}

func SetStorageAppDir(path ...string) string {
	appDir := "app"
	if len(path) > 0 {
		appDir += "/" + path[0]
	}

	return SetStorageDir(appDir)
}

func SetStorageAppPublicDir(path ...string) string {
	publicDir := "app/public"
	if len(path) > 0 {
		publicDir += "/" + path[0]
	}

	return SetStorageDir(publicDir)
}

func StringToArrayInt(text string) []int {
	var array []int
	texts := strings.Split(text, ",")
	for _, value := range texts {
		item, _ := strconv.Atoi(value)
		array = append(array, item)
	}

	return array
}

func StringToArrayString(text string) []string {
	var array []string
	texts := strings.Split(text, ",")
	for _, value := range texts {
		array = append(array, value)
	}

	return array
}

func GetMimeType(file multipart.File, handler *multipart.FileHeader, mimeType *string) string {
	if mimeType == nil || *mimeType == "" {
		buf := make([]byte, 512)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			xtremeres.ErrXtremeUploadFile("Unable to reading file!!")
		}

		mimeTypeSystem := http.DetectContentType(buf[:n])
		if mimeTypeSystem == "application/zip" {
			ext := strings.ToLower(filepath.Ext(handler.Filename))
			switch ext {
			case ".xlsx":
				return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
			case ".docx":
				return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			}
		}

		return mimeTypeSystem
	}

	return *mimeType
}

func CountFunc[S ~[]E, E any](data S, count *int, cb func(E) bool) {
	for _, value := range data {
		if cb(value) {
			*count++
		}
	}
}

func ToInt(text string) int {
	value, _ := strconv.Atoi(text)
	return value
}

func ToBool(text string) bool {
	value, _ := strconv.ParseBool(text)
	return value
}

func ToFloat64(text string) float64 {
	value, _ := strconv.ParseFloat(text, 64)
	return value
}
