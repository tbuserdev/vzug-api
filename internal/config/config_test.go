package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BaseURL != "http://device.local" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.Port != 3000 {
		t.Fatalf("Port = %d", cfg.Port)
	}
	if cfg.HTTPTimeout != 10*time.Second {
		t.Fatalf("HTTPTimeout = %s", cfg.HTTPTimeout)
	}
	if cfg.MQTTEnabled() {
		t.Fatal("MQTT should be disabled without MQTT_BROKER")
	}
}

func TestLoadMQTTConfigSanitizesIDs(t *testing.T) {
	clearEnv(t)
	setEnv(t, "MQTT_BROKER", "tcp://mqtt.local:1883")
	setEnv(t, "MQTT_CLIENT_ID", "V-ZUG bridge")
	setEnv(t, "DEVICE_ID", "Kitchen V-ZUG")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.MQTT.ClientID != "V-ZUG_bridge" {
		t.Fatalf("ClientID = %q", cfg.MQTT.ClientID)
	}
	if cfg.MQTT.DeviceID != "Kitchen_V-ZUG" {
		t.Fatalf("DeviceID = %q", cfg.MQTT.DeviceID)
	}
	if !cfg.MQTTEnabled() {
		t.Fatal("MQTT should be enabled")
	}
}

func TestLoadReadsDotEnv(t *testing.T) {
	clearEnv(t)
	tempDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(tempDir) error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	env := strings.Join([]string{
		"BASE_URL=http://oven.local",
		"PORT=8123",
		"ALLOW_INSECURE_TLS=true",
		"HTTP_TIMEOUT=7s",
		"RETRY_COUNT=4",
		"RETRY_DELAY=2s",
		"MQTT_BROKER=tcp://mqtt.local:1883",
	}, "\n")
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatalf("WriteFile(.env) error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BaseURL != "http://oven.local" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.Port != 8123 {
		t.Fatalf("Port = %d", cfg.Port)
	}
	if !cfg.AllowInsecureTLS {
		t.Fatal("AllowInsecureTLS = false")
	}
	if cfg.HTTPTimeout != 7*time.Second {
		t.Fatalf("HTTPTimeout = %s", cfg.HTTPTimeout)
	}
	if cfg.RetryCount != 4 {
		t.Fatalf("RetryCount = %d", cfg.RetryCount)
	}
	if cfg.RetryDelay != 2*time.Second {
		t.Fatalf("RetryDelay = %s", cfg.RetryDelay)
	}
	if !cfg.MQTTEnabled() {
		t.Fatal("MQTT should be enabled from .env")
	}
}

func TestLoadRejectsInvalidEnvValues(t *testing.T) {
	cases := map[string]string{
		"PORT":               "abc",
		"ALLOW_INSECURE_TLS": "treu",
		"HTTP_TIMEOUT":       "wat",
		"RETRY_COUNT":        "many",
		"RETRY_DELAY":        "later",
	}

	for key, value := range cases {
		t.Run(key, func(t *testing.T) {
			clearEnv(t)
			setEnv(t, key, value)
			_, err := Load()
			if err == nil {
				t.Fatalf("Load() expected error for %s=%q", key, value)
			}
			if !strings.Contains(err.Error(), key) {
				t.Fatalf("Load() error %q does not mention %s", err, key)
			}
		})
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range configKeys() {
		unsetEnv(t, key)
	}
}

func setEnv(t *testing.T, key string, value string) {
	t.Helper()
	previous, ok := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Setenv(%s) error = %v", key, err)
	}
	t.Cleanup(func() {
		restoreEnv(key, previous, ok)
	})
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	previous, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%s) error = %v", key, err)
	}
	t.Cleanup(func() {
		restoreEnv(key, previous, ok)
	})
}

func restoreEnv(key string, value string, ok bool) {
	if ok {
		_ = os.Setenv(key, value)
		return
	}
	_ = os.Unsetenv(key)
}

func configKeys() []string {
	return []string{
		"BASE_URL",
		"PORT",
		"ALLOW_INSECURE_TLS",
		"HTTP_TIMEOUT",
		"RETRY_COUNT",
		"RETRY_DELAY",
		"TIMEZONE",
		"SHOW_SCHEDULE",
		"HIDE_SCHEDULE",
		"MQTT_BROKER",
		"MQTT_USERNAME",
		"MQTT_PASSWORD",
		"MQTT_CLIENT_ID",
		"MQTT_DISCOVERY_PREFIX",
		"MQTT_BASE_TOPIC",
		"DEVICE_ID",
		"DEVICE_NAME",
	}
}
