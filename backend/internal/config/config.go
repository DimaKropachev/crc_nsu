package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                  string
	HTTPAddr             string
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	DatabaseURL          string
	VKAppID              string
	VKAppSecret          string
	JWTSecret            string
	JWTAccessTTL         time.Duration
	AdminVKIDs           map[int64]struct{}
	StorageDir           string
	MaxUploadBytes       int64
	AIProvider           string
	AIAPIKey             string
	AIBaseURL            string
	AIModel              string
	AITimeout            time.Duration
	ReminderEnabled      bool
	NotificationProvider string
	NotificationAPIToken string
	VKAPIVersion         string
	AllowedOrigins       []string
}

func Load() (Config, error) {
	cfg := Config{
		Env:                  env("APP_ENV", "development"),
		HTTPAddr:             env("HTTP_ADDR", ":8080"),
		ReadTimeout:          duration("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:         duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:          duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		VKAppID:              os.Getenv("VK_APP_ID"),
		VKAppSecret:          os.Getenv("VK_APP_SECRET"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		JWTAccessTTL:         duration("JWT_ACCESS_TTL", time.Hour),
		AdminVKIDs:           parseIDs(os.Getenv("ADMIN_VK_IDS")),
		StorageDir:           env("FILE_STORAGE_DIR", "./storage"),
		MaxUploadBytes:       int64(integer("MAX_UPLOAD_SIZE_MB", 10)) * 1024 * 1024,
		AIProvider:           strings.ToLower(env("AI_PROVIDER", "")),
		AIAPIKey:             os.Getenv("AI_API_KEY"),
		AIBaseURL:            env("AI_BASE_URL", "https://api.openai.com/v1"),
		AIModel:              os.Getenv("AI_MODEL"),
		AITimeout:            duration("AI_TIMEOUT", 30*time.Second),
		ReminderEnabled:      os.Getenv("REMINDER_ENABLED") == "true",
		NotificationProvider: strings.ToLower(os.Getenv("NOTIFICATION_PROVIDER")),
		NotificationAPIToken: os.Getenv("NOTIFICATION_API_TOKEN"),
		VKAPIVersion:         env("VK_API_VERSION", "5.199"),
		AllowedOrigins:       split(os.Getenv("CORS_ALLOWED_ORIGINS")),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Env == "production" {
		if len(cfg.AllowedOrigins) == 0 {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain the production frontend origin")
		}
		if len(cfg.JWTSecret) < 32 || strings.HasPrefix(cfg.JWTSecret, "replace-with-") {
			return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters in production")
		}
		if cfg.VKAppID == "" || cfg.VKAppSecret == "" {
			return Config{}, fmt.Errorf("VK_APP_ID and VK_APP_SECRET are required in production")
		}
	} else if cfg.JWTSecret == "" {
		cfg.JWTSecret = "dev-only-change-me-before-production-32-bytes-min"
	}
	if cfg.JWTAccessTTL < time.Minute || cfg.JWTAccessTTL > 24*time.Hour {
		return Config{}, fmt.Errorf("JWT_ACCESS_TTL must be between 1m and 24h")
	}
	if cfg.MaxUploadBytes < 1024*1024 || cfg.MaxUploadBytes > 100*1024*1024 {
		return Config{}, fmt.Errorf("MAX_UPLOAD_SIZE_MB must be between 1 and 100")
	}
	if cfg.AIProvider != "" && (cfg.AIAPIKey == "" || cfg.AIModel == "") {
		return Config{}, fmt.Errorf("AI_API_KEY and AI_MODEL are required when AI_PROVIDER is set")
	}
	if cfg.ReminderEnabled && (cfg.NotificationProvider != "vk" || cfg.NotificationAPIToken == "") {
		return Config{}, fmt.Errorf("REMINDER_ENABLED=true requires NOTIFICATION_PROVIDER=vk and NOTIFICATION_API_TOKEN")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func duration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return fallback
}
func integer(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}
func split(v string) []string {
	var out []string
	for _, item := range strings.Split(v, ",") {
		if s := strings.TrimSpace(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}
func parseIDs(v string) map[int64]struct{} {
	result := make(map[int64]struct{})
	for _, p := range strings.Split(v, ",") {
		if n, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil && n > 0 {
			result[n] = struct{}{}
		}
	}
	return result
}
