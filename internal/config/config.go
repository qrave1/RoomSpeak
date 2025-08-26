package config

import (
	"os"
	"strconv"
)

// TODO: refactor to carlos env

// Config конфигурация приложения
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port      int
	Host      string
	StaticDir string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	serverPort, _ := strconv.Atoi(getEnv("SERVER_PORT", "3000"))

	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			Database: getEnv("DB_NAME", "roomspeak"),
			Username: getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Server: ServerConfig{
			Port:      serverPort,
			Host:      getEnv("SERVER_HOST", "localhost"),
			StaticDir: getEnv("STATIC_DIR", "./web/static"),
		},
	}
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
