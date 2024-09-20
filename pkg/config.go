package xtremepkg

import (
	"context"
	"fmt"
	"github.com/globalxtreme/go-core/v2/grpc/pkg/bug"
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

	// DevMode --> Dev mode for use .env or kubernetes configmap
	DevMode bool

	// RPCDialTimeout --> gRPC dial timout to another services
	RPCDialTimeout time.Duration

	// XtremeValidate --> Validation configuration
	XtremeValidate *validator.Validate

	// BugRPCClient --> Bug service gRPC client
	BugRPCClient bug.BugServiceClient

	// BugRPCTimeout --> Bug service gRPC timeout while send log
	BugRPCTimeout time.Duration

	// BugRPCActive --> Bug service gRPC status active or inactive
	BugRPCActive bool

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

func InitDevMode() {
	if DevMode {
		fmt.Println("Running in development mode..")
		err := godotenv.Load()
		if err != nil {
			panic(err.Error())
		}
	}
}

func InitBugRPC() func() {
	addr := os.Getenv("GRPC_BUG_HOST")
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

		BugRPCClient = bug.NewBugServiceClient(conn)
		BugRPCActive = true

		BugRPCTimeout = 5 * time.Second
		if bugTimeoutENV := os.Getenv("GRPC_BUG_TIMEOUT"); bugTimeoutENV != "" {
			BugRPCTimeout = time.Duration(ToInt(bugTimeoutENV)) * time.Second
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
			return c, err
		},
	}
}
