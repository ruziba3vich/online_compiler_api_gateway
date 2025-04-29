package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		PythonService       string
		JavaService         string
		CppService          string
		GatewayPort         string
		LangStorageFilePath string
		LogsFilePath        string
		RLCnfg              *RateLimiter
		RedisCfg            *RedisConfig
	}

	RedisConfig struct {
		Host, Port, Password string
		DB                   int
	}

	RateLimiter struct {
		MaxTokens  int
		RefillRate float64
		Window     time.Duration
	}
)

func NewConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		PythonService:       getEnv("PYTHON_SERVICE", "217.76.51.104:7771"),
		JavaService:         getEnv("JAVA_SERVICE", "217.76.51.104:7773"),
		CppService:          getEnv("CPP_SERVICE", "217.76.51.104:7774"),
		GatewayPort:         getEnv("GATEWAY_PORT", "7772"),
		LangStorageFilePath: getEnv("LANG_STORAGE_FPATH", "data/languages.db"),
		LogsFilePath:        getEnv("LOGS_FILE_PATH", "data/app.log"),
		RedisCfg: &RedisConfig{
			Host:     getEnv("REDIS_HOST", "redis"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		RLCnfg: &RateLimiter{
			RefillRate: getEnvFloat64("REFILL_RATE", 0.25),
			MaxTokens:  getEnvInt("MAX_TOKENS", 15),
			Window:     getEnvDuration("RL_WINDOW", 1),
		},
	}
}

// getEnv returns the fallback value if the given key is not provided in env
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		v, err := strconv.Atoi(value)
		if err == nil {
			return v
		}
	}
	return fallback
}

func getEnvFloat64(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		v, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return v
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback int) time.Duration {
	if value := os.Getenv(key); value != "" {
		v := getEnvInt(key, fallback)
		return time.Duration(v) * time.Minute
	}

	return time.Duration(fallback) * time.Minute
}
