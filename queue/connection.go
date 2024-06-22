package xtremequeue

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"os"
	"time"
)

var RedisPool *redis.Pool

func RegisterRedis() {
	RedisPool = &redis.Pool{
		MaxIdle:     3,
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
