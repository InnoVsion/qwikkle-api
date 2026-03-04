package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	AppEnv         string
	JWTAccessSecret string
}

func Load() Config {
	// Load .env file if present; ignore error so production can rely on real env vars.
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	jwtSecret := os.Getenv("JWT_ACCESS_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET must be set")
	}

	return Config{
		Port:           port,
		AppEnv:         env,
		JWTAccessSecret: jwtSecret,
	}
}

