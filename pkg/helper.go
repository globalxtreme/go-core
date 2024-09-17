package xtremepkg

import (
	xtremeres "github.com/globalxtreme/go-core/v2/response"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	re := regexp.MustCompile(`[^0-9\-.]`)
	text = re.ReplaceAllString(text, " ")

	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	words := strings.Split(text, " ")

	var result []int
	for _, word := range words {
		if word != "" {
			number, _ := strconv.Atoi(word)
			result = append(result, number)
		}
	}

	return result
}

func StringToArrayString(text string) []string {
	re := regexp.MustCompile(`[^A-Za-z0-9\-.]`)
	text = re.ReplaceAllString(text, " ")

	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	words := strings.Split(text, " ")

	var result []string
	for _, word := range words {
		if word != "" {
			result = append(result, word)
		}
	}

	return result
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

func ArrayDiff[TD comparable](arr1, arr2 []TD) []TD {
	elMap := make(map[TD]bool)
	for _, val := range arr2 {
		elMap[val] = true
	}

	var res []TD
	for _, val := range arr1 {
		if !elMap[val] {
			res = append(res, val)
		}
	}

	return res
}

func ArrayUdiff[TD comparable](arr1, arr2 []TD) []TD {
	elMap := make(map[TD]bool)
	for _, val := range arr1 {
		elMap[val] = true
	}

	var res []TD
	for _, val := range arr2 {
		if elMap[val] {
			res = append(res, val)
		}
	}

	return res
}

func ArrayUnique[TD comparable](strings []TD) []TD {
	seen := make(map[TD]bool)
	var result []TD

	for _, str := range strings {
		if !seen[str] {
			result = append(result, str)
			seen[str] = true
		}
	}

	return result
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
