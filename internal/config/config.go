package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environment-driven configuration for the application.
// It is loaded once at startup and passed through the app via dependency injection.
type Config struct {
	// Server
	AppEnv string
	Port   string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret      string
	JWTExpiryHours int

	// Cloudflare R2
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string

	// Email
	ResendAPIKey string
	EmailFrom    string

	// Payments
	StripeSecretKey     string
	StripeWebhookSecret string
	PaystackSecretKey   string

	// Frontend
	FrontendURL string
}

// Load reads from a .env file (in development) and then from real environment
// variables, with real env vars taking precedence. Safe to call in production
// where no .env file exists — godotenv simply skips missing files.
func Load() (*Config, error) {
	// Silently ignore missing .env — expected in production
	_ = godotenv.Load()

	jwtExpiry, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "72"))
	if err != nil {
		return nil, fmt.Errorf("config: JWT_EXPIRY_HOURS must be an integer: %w", err)
	}

	cfg := &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Port:   getEnv("PORT", "8080"),

		DatabaseURL: requireEnv("DATABASE_URL"),

		JWTSecret:      requireEnv("JWT_SECRET"),
		JWTExpiryHours: jwtExpiry,

		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", "zigakit-files"),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),

		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		EmailFrom:    getEnv("EMAIL_FROM", "noreply@zigakit.com"),

		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		PaystackSecretKey:   getEnv("PAYSTACK_SECRET_KEY", ""),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
	}

	return cfg, nil
}

// IsProd returns true when running in a production environment.
func (c *Config) IsProd() bool {
	return c.AppEnv == "production"
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// requireEnv panics at startup if a mandatory env var is missing.
// Failing fast here is intentional — a misconfigured server should not start.
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("config: required environment variable %q is not set", key))
	}
	return v
}
