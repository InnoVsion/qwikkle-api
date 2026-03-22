package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	AppEnv             string
	JWTAccessSecret    string
	JWTRefreshSecret   string
	CookieDomain       string
	CORSAllowedOrigins string
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

	jwtRefreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if jwtRefreshSecret == "" {
		log.Fatal("JWT_REFRESH_SECRET must be set")
	}

	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	corsAllowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")

	return Config{
		Port:               port,
		AppEnv:             env,
		JWTAccessSecret:    jwtSecret,
		JWTRefreshSecret:   jwtRefreshSecret,
		CookieDomain:       cookieDomain,
		CORSAllowedOrigins: corsAllowedOrigins,
	}
}
