package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// ConnectRedis initializes the Redis client with optional TLS and Authentication
func ConnectRedis() {
	redisHost := getEnv("REDIS_HOST", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisSecure := getEnv("REDIS_SECURE", "false")

	var tlsConfig *tls.Config
	if redisSecure == "true" {
		tlsConfig = &tls.Config{} // Enable secure TLS
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		Password:     redisPassword,
		DB:           0, // Default DB
		TLSConfig:    tlsConfig,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(fmt.Errorf("failed to ping redis at %s: %w", redisHost, err))
	}

	RedisClient = rdb
	fmt.Printf("Connected to Redis successfully (%s, TLS: %s)!\n", redisHost, redisSecure)
}


