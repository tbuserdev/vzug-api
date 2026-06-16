package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("BASE_URL", "")
	t.Setenv("PORT", "")
	t.Setenv("MQTT_BROKER", "")

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
	t.Setenv("MQTT_BROKER", "tcp://mqtt.local:1883")
	t.Setenv("MQTT_CLIENT_ID", "V-ZUG bridge")
	t.Setenv("DEVICE_ID", "Kitchen V-ZUG")

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
