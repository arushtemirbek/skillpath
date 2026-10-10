package config

import (
	"errors"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP   HTTPConfig
	OpenAI OpenAIConfig
	Log    LogConfig
}

type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	// Загружаем .env, если файл существует.
	// Если его нет — это не ошибка, например в production
	// переменные могут передаваться напрямую через environment.
	_ = godotenv.Load()

	cfg := &Config{
		HTTP: HTTPConfig{
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},

		OpenAI: OpenAIConfig{
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			Model:   os.Getenv("OPENAI_MODEL"),
			BaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
			Timeout: getDuration("OPENAI_TIMEOUT", 30*time.Second),
		},

		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.HTTP.Port == "" {
		return errors.New("HTTP_PORT is required")
	}

	if c.OpenAI.APIKey == "" {
		return errors.New("OPENAI_API_KEY is required")
	}

	if c.OpenAI.Model == "" {
		return errors.New("OPENAI_MODEL is required")
	}

	return nil
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return duration
}
