package xtremepkg

import (
	"context"
	"encoding/json"
	"fmt"
	log2 "github.com/globalxtreme/go-core/v2/grpc/pkg/log"
	"log"
	"os"
	"runtime/debug"
	"time"
)

type LogForm struct {
	Type            string      `json:"type"`
	DateTime        string      `json:"dateTime"`
	Content         any         `json:"content"`
	Stack           []byte      `json:"stack"`
	Bug             bool        `json:"bug"`
	Payload         interface{} `json:"payload"`
	PerformedBy     string      `json:"performedBy"`
	PerformedByName string      `json:"performedByName"`
	PerformedByType string      `json:"performedByType"`
}

func Log(form LogForm) {
	if LogRPCActive {
		message, _ := json.Marshal(form.Content)

		request := log2.LogRequest{
			Service:         os.Getenv("SERVICE"),
			DateTime:        form.DateTime,
			Message:         string(message),
			Type:            form.Type,
			Stack:           form.Stack,
			Bug:             form.Bug,
			PerformedBy:     form.PerformedBy,
			PerformedByName: form.PerformedByName,
			PerformedByType: form.PerformedByType,
		}

		if request.Type == "" {
			request.Type = "INFO"
		}

		if form.Payload != nil {
			payload, _ := json.Marshal(form.Payload)
			request.Payload = payload
		}

		sendLog(&request)
	} else {
		setLogOutput(form.Type, form.Content)
	}
}

func LogInfo(content any) {
	logType := "INFO"
	if LogRPCActive {
		message, _ := json.Marshal(content)

		sendLog(&log2.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: string(message),
		})
	} else {
		setLogOutput(logType, content)
	}
}

func LogError(content any, bug bool) {
	debug.PrintStack()

	logType := "ERROR"
	if LogRPCActive {
		sendLog(&log2.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: fmt.Sprintf("panic: %v", content),
			Stack:   debug.Stack(),
			Bug:     bug,
		})
	} else {
		setLogOutput(logType, fmt.Sprintf("panic: %v", content))
		setLogOutput(logType, string(debug.Stack()))
	}
}

func LogDebug(content any) {
	logType := "DEBUG"
	if LogRPCActive {
		message, _ := json.Marshal(content)

		sendLog(&log2.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: string(message),
		})
	} else {
		setLogOutput(logType, content)
	}
}

func sendLog(req *log2.LogRequest) (*log2.LGResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), LogRPCTimeout)
	defer cancel()

	return LogRPCClient.Log(ctx, req)
}

func setLogOutput(action string, error any) {
	storageDir := os.Getenv("STORAGE_DIR") + "/logs"
	CheckAndCreateDirectory(storageDir)

	filename := time.Now().Format("2006-01-02") + ".log"
	file, err := os.OpenFile(storageDir+"/"+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	log.SetOutput(file)
	log.Println(fmt.Sprintf("[%s]:", action), error)
}
