package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		PythonService       string
		JavaService         string
		GatewayPort         string
		LangStorageFilePath string
		LogsFilePath        string
		RedisCfg            *RedisConfig
	}

	RedisConfig struct {
		Host, Port, Password string
		DB                   int
	}
)

func NewConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		PythonService:       getEnv("PYTHON_SERVICE", "217.76.51.104:7771"),
		JavaService:         getEnv("PYTHON_SERVICE", "217.76.51.104:7773"),
		GatewayPort:         getEnv("GATEWAY_PORT", "7772"),
		LangStorageFilePath: getEnv("LANG_STORAGE_FPATH", "languages.json"),
		LogsFilePath:        getEnv("LOGS_FILE_PATH", "app.log"),
		RedisCfg: &RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
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
