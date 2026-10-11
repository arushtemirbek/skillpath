package config

import (
	"strings"
	"testing"
	"time"
)

func testEnv(overrides map[string]string) lookupEnv {
	values := map[string]string{"OPENAI_API_KEY": "test-key", "OPENAI_MODEL": "test-model"}
	for key, value := range overrides {
		values[key] = value
	}
	return func(key string) (string, bool) { value, exists := values[key]; return value, exists }
}

func TestConfigurationDefaults(t *testing.T) {
	cfg, err := load(testEnv(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Port != "8080" || cfg.OpenAI.Timeout != 30*time.Second || cfg.HTTP.WriteTimeout != 35*time.Second || cfg.Log.Level != "info" {
		t.Fatalf("incorrect defaults: HTTP=%+v", cfg.HTTP)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, value, message string }{
		{"OPENAI_API_KEY", "", "OPENAI_API_KEY is required"},
		{"OPENAI_MODEL", " ", "OPENAI_MODEL is required"},
		{"HTTP_PORT", "", "HTTP_PORT is required"},
		{"HTTP_PORT", "65536", "HTTP_PORT"},
		{"HTTP_PORT", "abc", "HTTP_PORT"},
		{"HTTP_READ_TIMEOUT", "oops", "HTTP_READ_TIMEOUT"},
		{"HTTP_IDLE_TIMEOUT", "0s", "must be positive"},
		{"HTTP_SHUTDOWN_TIMEOUT", "-1s", "must be positive"},
		{"REQUEST_TIMEOUT", "", "REQUEST_TIMEOUT"},
		{"HTTP_WRITE_TIMEOUT", "30s", "must exceed REQUEST_TIMEOUT"},
		{"OPENAI_BASE_URL", "api.openai.com", "OPENAI_BASE_URL"},
		{"OPENAI_BASE_URL", "https://user:secret@example.com", "OPENAI_BASE_URL"},
		{"OPENAI_BASE_URL", "https://example.com?secret=key", "OPENAI_BASE_URL"},
		{"LOG_LEVEL", "verbose", "LOG_LEVEL"},
	} {
		t.Run(tc.key+"/"+tc.value, func(t *testing.T) {
			_, err := load(testEnv(map[string]string{tc.key: tc.value}))
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error=%v, want %s", err, tc.message)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("secret leaked in configuration error")
			}
		})
	}
}

func TestTimeoutCompatibility(t *testing.T) {
	cfg, err := load(testEnv(map[string]string{"OPENAI_TIMEOUT": "20s"}))
	if err != nil || cfg.OpenAI.Timeout != 20*time.Second {
		t.Fatalf("legacy timeout: %v", err)
	}
	cfg, err = load(testEnv(map[string]string{"OPENAI_TIMEOUT": "invalid", "REQUEST_TIMEOUT": "10s"}))
	if err != nil || cfg.OpenAI.Timeout != 10*time.Second {
		t.Fatalf("REQUEST_TIMEOUT precedence: %v", err)
	}
}
