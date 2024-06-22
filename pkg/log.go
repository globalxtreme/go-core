package xtremepkg

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"
)

func LogInfo(content any) {
	setLogOutput("INFO", content)
}

func LogError(content any) {
	debug.PrintStack()

	setLogOutput("ERROR", fmt.Sprintf("panic: %v", content))
	setLogOutput("ERROR", string(debug.Stack()))
}

func LogDebug(content any) {
	setLogOutput("DEBUG", content)
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
