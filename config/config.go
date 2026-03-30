package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	OpenRouterAPIKey string
	InternalSecret   string
	JWTSecret        string
	AuthUsername     string
	AuthPassword     string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		InternalSecret:   os.Getenv("INTERNAL_SERVICE_SECRET"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AuthUsername:     os.Getenv("AUTH_USERNAME"),
		AuthPassword:     os.Getenv("AUTH_PASSWORD"),
	}

	if cfg.OpenRouterAPIKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable not set")
	}
	if cfg.InternalSecret == "" {
		log.Fatal("INTERNAL_SERVICE_SECRET environment variable not set")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}
	if cfg.AuthUsername == "" || cfg.AuthPassword == "" {
		log.Fatal("AUTH_USERNAME and AUTH_PASSWORD environment variables not set")
	}

	return cfg
}
