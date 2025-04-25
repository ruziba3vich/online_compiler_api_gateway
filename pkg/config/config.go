package config

import (
	"os"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		PythonService       string
		JavaService         string
		GatewayPort         string
		LangStorageFilePath string
	}
)

func NewConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		PythonService:       getEnv("PYTHON_SERVICE", "217.76.51.104:7771"),
		JavaService:         getEnv("PYTHON_SERVICE", "217.76.51.104:7773"),
		GatewayPort:         getEnv("GATEWAY_PORT", "7772"),
		LangStorageFilePath: getEnv("LANG_STORAGE_FPATH", "languages.json"),
	}
}

// getEnv returns the fallback value if the given key is not provided in env
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
