package config

import (
	"testing"
	"time"
)

func TestDurationValidation(t *testing.T) {
	for _, value := range []string{"oops", "0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("HTTP_READ_TIMEOUT", value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid timeout accepted")
			}
		})
	}
	t.Setenv("HTTP_READ_TIMEOUT", "")
	value, err := getDuration("HTTP_READ_TIMEOUT", 5*time.Second)
	if err != nil || value != 5*time.Second {
		t.Fatalf("default = %v, error = %v", value, err)
	}
}

func TestWriteTimeoutBudget(t *testing.T) {
	cfg := Config{
		HTTP:   HTTPConfig{Port: "8080", ReadTimeout: time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: time.Second, ShutdownTimeout: time.Second},
		OpenAI: OpenAIConfig{APIKey: "test", Model: "test", Timeout: 30 * time.Second},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("no time left to send response")
	}
	cfg.HTTP.WriteTimeout = 35 * time.Second
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
