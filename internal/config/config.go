package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	TelegramToken  string
	BaseURL        string
}

func Load() *Config {
	_ = godotenv.Load()
	cfg := &Config{
		Port:          get("PORT", "8080"),
		DatabaseURL:   must("DATABASE_URL"),
		JWTSecret:     must("JWT_SECRET"),
		TelegramToken: must("TELEGRAM_BOT_TOKEN"),
		BaseURL:       must("BASE_URL"),
	}
	return cfg
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func must(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing env %s", k)
	}
	return v
}
