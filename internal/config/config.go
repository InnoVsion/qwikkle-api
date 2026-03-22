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
	StorageProvider    string
	S3Region           string
	S3Bucket           string
	S3Endpoint         string
	S3AccessKeyID      string
	S3SecretAccessKey  string
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
	storageProvider := os.Getenv("STORAGE_PROVIDER")
	if storageProvider == "" {
		storageProvider = "s3"
	}

	return Config{
		Port:               port,
		AppEnv:             env,
		JWTAccessSecret:    jwtSecret,
		JWTRefreshSecret:   jwtRefreshSecret,
		CookieDomain:       cookieDomain,
		CORSAllowedOrigins: corsAllowedOrigins,
		StorageProvider:    storageProvider,
		S3Region:           os.Getenv("S3_REGION"),
		S3Bucket:           os.Getenv("S3_BUCKET"),
		S3Endpoint:         os.Getenv("S3_ENDPOINT"),
		S3AccessKeyID:      os.Getenv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:  os.Getenv("S3_SECRET_ACCESS_KEY"),
	}
}
