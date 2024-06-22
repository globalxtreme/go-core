package xtremepkg

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"time"
)

var (
	// Host --> Host for run application without protocol
	Host string

	// HostFull --> Host with protocol
	HostFull string

	// DevMode --> Dev mode for use .env or kubernetes configmap
	DevMode bool

	// RPCDialTimeout --> gRPC dial timout to another services
	RPCDialTimeout time.Duration

	// RedisPool --> Redis pool for queue worker
	RedisPool *redis.Pool

	// XtremeValidate --> Validation configuration
	XtremeValidate *validator.Validate
)

func InitHost() {
	protocol := "http"
	ssl, _ := strconv.ParseBool(os.Getenv("USE_SSL"))
	if ssl == true {
		protocol = "https"
	}

	Host = os.Getenv("DOMAIN")
	port := os.Getenv("PORT")

	HostFull = protocol + "://" + Host
	if ssl == false {
		HostFull += ":" + port
	}

	Host += ":" + port
}

func InitDevMode() {
	if DevMode {
		fmt.Println("Running in development mode..")
		err := godotenv.Load()
		if err != nil {
			panic(err.Error())
		}
	}
}
