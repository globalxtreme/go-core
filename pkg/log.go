package xtremepkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/globalxtreme/go-core/v2/grpc/pkg/bug"
	"log"
	"os"
	"runtime/debug"
	"time"
)

func LogInfo(content any) {
	logType := "INFO"
	if BugRPCActive {
		message, _ := json.Marshal(content)

		SendBugLog(&bug.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: message,
		})
	} else {
		setLogOutput(logType, content)
	}
}

func LogError(content any) {
	debug.PrintStack()

	logType := "ERROR"
	if BugRPCActive {
		SendBugLog(&bug.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Title:   fmt.Sprintf("panic: %v", content),
			Message: debug.Stack(),
		})
	} else {
		setLogOutput("ERROR", fmt.Sprintf("panic: %v", content))
		setLogOutput("ERROR", string(debug.Stack()))
	}
}

func LogDebug(content any) {
	logType := "DEBUG"
	if BugRPCActive {
		message, _ := json.Marshal(content)

		SendBugLog(&bug.LogRequest{
			Service: os.Getenv("SERVICE"),
			Type:    logType,
			Message: message,
		})
	} else {
		setLogOutput("DEBUG", content)
	}
}

func SendBugLog(req *bug.LogRequest) (*bug.BGResponse, error) {
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
