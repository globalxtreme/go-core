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

func LogInfo(content any) {
	logType := "INFO"
	if BugRPCActive {
		message, _ := json.Marshal(content)

		SendBugLog(&log2.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: string(message),
		})
	} else {
		setLogOutput(logType, content)
	}
}

func LogError(content any) {
	debug.PrintStack()

	logType := "ERROR"
	if BugRPCActive {
		SendBugLog(&log2.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: fmt.Sprintf("panic: %v", content),
			Detail:  debug.Stack(),
		})
	} else {
		setLogOutput(logType, fmt.Sprintf("panic: %v", content))
		setLogOutput(logType, string(debug.Stack()))
	}
}

func LogDebug(content any) {
	logType := "DEBUG"
	if BugRPCActive {
		message, _ := json.Marshal(content)

		SendBugLog(&log2.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: string(message),
		})
	} else {
		setLogOutput(logType, content)
	}
}

func SendBugLog(req *log2.LogRequest) (*log2.LGResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), BugRPCTimeout)
	defer cancel()

	return BugRPCClient.Log(ctx, req)
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
