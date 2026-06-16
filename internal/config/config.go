package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const Version = "0.1.0"

type Config struct {
	BaseURL          string
	Port             int
	AllowInsecureTLS bool
	HTTPTimeout      time.Duration
	RetryCount       int
	RetryDelay       time.Duration
	Timezone         string
	ShowSchedule     string
	HideSchedule     string
	MQTT             MQTTConfig
}

type MQTTConfig struct {
	Broker          string
	Username        string
	Password        string
	ClientID        string
	DiscoveryPrefix string
	BaseTopic       string
	DeviceID        string
	DeviceName      string
}

func Load() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	port, err := getenvInt("PORT", 3000)
	if err != nil {
		return Config{}, err
	}
	allowInsecureTLS, err := getenvBool("ALLOW_INSECURE_TLS", false)
	if err != nil {
		return Config{}, err
	}
	httpTimeout, err := getenvDuration("HTTP_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	retryCount, err := getenvInt("RETRY_COUNT", 3)
	if err != nil {
		return Config{}, err
	}
	retryDelay, err := getenvDuration("RETRY_DELAY", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		BaseURL:          strings.TrimRight(getenv("BASE_URL", "http://device.local"), "/"),
		Port:             port,
		AllowInsecureTLS: allowInsecureTLS,
		HTTPTimeout:      httpTimeout,
		RetryCount:       retryCount,
		RetryDelay:       retryDelay,
		Timezone:         getenv("TIMEZONE", "Europe/Zurich"),
		ShowSchedule:     getenv("SHOW_SCHEDULE", "0 22 * * *"),
		HideSchedule:     getenv("HIDE_SCHEDULE", "0 6 * * *"),
		MQTT: MQTTConfig{
			Broker:          os.Getenv("MQTT_BROKER"),
			Username:        os.Getenv("MQTT_USERNAME"),
			Password:        os.Getenv("MQTT_PASSWORD"),
			ClientID:        sanitizeID(getenv("MQTT_CLIENT_ID", "vzug-ha")),
			DiscoveryPrefix: trimTopic(getenv("MQTT_DISCOVERY_PREFIX", "homeassistant")),
			BaseTopic:       trimTopic(getenv("MQTT_BASE_TOPIC", "vzug/display_clock")),
			DeviceID:        sanitizeID(getenv("DEVICE_ID", "vzug_display_clock")),
			DeviceName:      getenv("DEVICE_NAME", "V-ZUG Display Clock"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.BaseURL == "" {
		return errors.New("BASE_URL is required")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("BASE_URL must be an absolute URL, got %q", c.BaseURL)
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535, got %d", c.Port)
	}
	if c.HTTPTimeout <= 0 {
		return errors.New("HTTP_TIMEOUT must be positive")
	}
	if c.RetryCount < 0 {
		return errors.New("RETRY_COUNT must be zero or greater")
	}
	if c.RetryDelay < 0 {
		return errors.New("RETRY_DELAY must be zero or greater")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return fmt.Errorf("TIMEZONE is invalid: %w", err)
	}
	if c.MQTT.Broker == "" {
		return nil
	}
	if c.MQTT.ClientID == "" {
		return errors.New("MQTT_CLIENT_ID must contain at least one alphanumeric character")
	}
	if c.MQTT.DiscoveryPrefix == "" {
		return errors.New("MQTT_DISCOVERY_PREFIX cannot be empty")
	}
	if c.MQTT.BaseTopic == "" {
		return errors.New("MQTT_BASE_TOPIC cannot be empty")
	}
	if c.MQTT.DeviceID == "" {
		return errors.New("DEVICE_ID must contain at least one alphanumeric character")
	}
	return nil
}

func (c Config) MQTTEnabled() bool {
	return c.MQTT.Broker != ""
}

func loadDotEnv() error {
	if err := godotenv.Load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load .env: %w", err)
	}
	return nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func getenvInt(key string, fallback int) (int, error) {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer, got %q", key, v)
		}
		return parsed, nil
	}
	return fallback, nil
}

func getenvBool(key string, fallback bool) (bool, error) {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, fmt.Errorf("%s must be a boolean, got %q", key, v)
		}
		return parsed, nil
	}
	return fallback, nil
}

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		parsed, err := time.ParseDuration(strings.TrimSpace(v))
		if err == nil {
			return parsed, nil
		}
		if seconds, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return time.Duration(seconds) * time.Second, nil
		}
		return 0, fmt.Errorf("%s must be a duration such as 10s or integer seconds, got %q", key, v)
	}
	return fallback, nil
}

func trimTopic(topic string) string {
	return strings.Trim(strings.TrimSpace(topic), "/")
}

func sanitizeID(value string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		case r == '_' || r == '-':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_-")
}
