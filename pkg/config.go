package xtremepkg

import (
	"context"
	"fmt"
	log2 "github.com/globalxtreme/go-core/v2/grpc/pkg/log"
	"github.com/go-playground/validator/v10"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	// Host --> Host for run application without protocol
	Host string

	// HostFull --> Host with protocol
	HostFull string

	// PrivateHost --> Private Host for microservice communication
	PrivateHost string

	// DevMode --> Dev mode for use .env or kubernetes configmap
	DevMode bool

	// RPCDialTimeout --> gRPC dial timout to another services
	RPCDialTimeout time.Duration

	// XtremeValidate --> Validation configuration
	XtremeValidate *validator.Validate

	// LogRPCClient --> Log service gRPC client
	LogRPCClient log2.LogServiceClient

	// LogRPCTimeout --> Log service gRPC timeout while send log
	LogRPCTimeout time.Duration

	// LogRPCActive --> Log service gRPC status active or inactive
	LogRPCActive bool

	// RedisPool --> Redis pool for open connection
	RedisPool *redis.Pool
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

func InitPrivateHost() {
	PrivateHost = os.Getenv("PRIVATE_HOST")
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

func InitLogRPC() func() {
	addr := os.Getenv("GRPC_LOG_HOST")
	if !DevMode && addr != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		keepaliveParam := keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}

		conn, err := grpc.DialContext(ctx, addr,
			grpc.WithInsecure(),
			grpc.WithBlock(),
			grpc.WithKeepaliveParams(keepaliveParam),
		)
		if err != nil {
			log.Panicf("Did not connect to %s: %v", addr, err)
		}

		LogRPCClient = log2.NewLogServiceClient(conn)
		LogRPCActive = true

		LogRPCTimeout = 5 * time.Second
		if bugTimeoutENV := os.Getenv("GRPC_LOG_TIMEOUT"); bugTimeoutENV != "" {
			LogRPCTimeout = time.Duration(ToInt(bugTimeoutENV)) * time.Second
		}

		cleanup := func() {
			cancel()
			conn.Close()
		}

		return cleanup
	}

	return func() {}
}

func InitRedisPool() {
	RedisPool = &redis.Pool{
		MaxIdle:     100,
		MaxActive:   500,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")))
			if err != nil {
				return nil, err
			}

			if os.Getenv("REDIS_PASSWORD") != "" {
				if _, err = c.Do("AUTH", os.Getenv("REDIS_PASSWORD")); err != nil {
					c.Close()
					return nil, err
				}
			}

			return c, nil
		},
	}
}
