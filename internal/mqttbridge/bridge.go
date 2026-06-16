package mqttbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/tbuserdev/vzug-api/internal/config"
	"github.com/tbuserdev/vzug-api/internal/state"
)

type StateReader interface {
	Snapshot() state.Snapshot
}

type CommandHandler func(ctx context.Context, visible bool, action string) error

type command struct {
	visible bool
}

type Bridge struct {
	cfg      config.MQTTConfig
	state    StateReader
	handler  CommandHandler
	client   mqtt.Client
	logger   *slog.Logger
	ctx      context.Context
	commands chan command
}

func New(cfg config.MQTTConfig, store StateReader, handler CommandHandler, logger *slog.Logger) *Bridge {
	return &Bridge{
		cfg:     cfg,
		state:   store,
		handler: handler,
		logger:  logger,
	}
}

func (b *Bridge) Start(ctx context.Context) error {
	b.ctx = ctx
	b.commands = make(chan command, 32)
	go b.runCommands()

	opts := mqtt.NewClientOptions().
		AddBroker(b.cfg.Broker).
		SetClientID(b.cfg.ClientID).
		SetUsername(b.cfg.Username).
		SetPassword(b.cfg.Password).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetCleanSession(true).
		SetOrderMatters(true).
		SetWill(b.availabilityTopic(), "offline", 1, true)

	opts.OnConnect = func(client mqtt.Client) {
		b.logger.Info("connected to MQTT broker", "broker", b.cfg.Broker)
		if err := b.publishAvailability("online"); err != nil {
			b.logger.Error("failed to publish MQTT availability", "error", err)
		}
		if err := b.PublishDiscovery(); err != nil {
			b.logger.Error("failed to publish MQTT discovery", "error", err)
		}
		if err := b.PublishState(); err != nil {
			b.logger.Error("failed to publish MQTT state", "error", err)
		}
		if token := client.Subscribe(b.commandTopic(), 1, b.handleCommand); !token.WaitTimeout(10*time.Second) || token.Error() != nil {
			b.logger.Error("failed to subscribe to MQTT command topic", "topic", b.commandTopic(), "error", token.Error())
		}
		if token := client.Subscribe(b.cfg.DiscoveryPrefix+"/status", 0, b.handleHABirth); !token.WaitTimeout(10*time.Second) || token.Error() != nil {
			b.logger.Error("failed to subscribe to Home Assistant birth topic", "error", token.Error())
		}
	}
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		b.logger.Error("lost MQTT connection", "error", err)
	}

	b.client = mqtt.NewClient(opts)
	token := b.client.Connect()
	if !token.WaitTimeout(30 * time.Second) {
		return errors.New("timed out connecting to MQTT broker")
	}
	if err := token.Error(); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		if b.client != nil && b.client.IsConnected() {
			_ = b.publishAvailability("offline")
			b.client.Disconnect(250)
		}
	}()
	return nil
}

func (b *Bridge) PublishDiscovery() error {
	payload := map[string]any{
		"name":                  "Display Clock",
		"unique_id":             b.cfg.DeviceID + "_display_clock",
		"object_id":             b.cfg.DeviceID + "_display_clock",
		"command_topic":         b.commandTopic(),
		"state_topic":           b.stateTopic(),
		"json_attributes_topic": b.attributesTopic(),
		"availability_topic":    b.availabilityTopic(),
		"payload_on":            "ON",
		"payload_off":           "OFF",
		"payload_available":     "online",
		"payload_not_available": "offline",
		"icon":                  "mdi:clock-digital",
		"optimistic":            false,
		"qos":                   1,
		"device": map[string]any{
			"identifiers":  []string{b.cfg.DeviceID},
			"name":         b.cfg.DeviceName,
			"manufacturer": "V-ZUG",
			"model":        "Display clock bridge",
			"sw_version":   config.Version,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return b.publish(b.discoveryTopic(), body, true)
}

func (b *Bridge) PublishState() error {
	snapshot := b.state.Snapshot()
	if snapshot.Known {
		if err := b.publish(b.stateTopic(), []byte(state.Payload(snapshot.Visible)), true); err != nil {
			return err
		}
	}
	attributes := map[string]any{
		"last_updated": snapshot.LastUpdated.Format(time.RFC3339),
		"last_action":  snapshot.LastAction,
		"state_known":  snapshot.Known,
	}
	if snapshot.LastError != "" {
		attributes["last_error"] = snapshot.LastError
	}
	body, err := json.Marshal(attributes)
	if err != nil {
		return err
	}
	return b.publish(b.attributesTopic(), body, true)
}

func (b *Bridge) handleCommand(_ mqtt.Client, message mqtt.Message) {
	visible, err := parseSwitchPayload(string(message.Payload()))
	if err != nil {
		b.logger.Warn("ignoring invalid MQTT command", "topic", message.Topic(), "payload", string(message.Payload()))
		return
	}
	select {
	case b.commands <- command{visible: visible}:
	case <-b.ctx.Done():
		b.logger.Warn("dropping MQTT command during shutdown", "visible", visible)
	}
}

func (b *Bridge) runCommands() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case command := <-b.commands:
			b.applyCommand(command)
		}
	}
}

func (b *Bridge) applyCommand(command command) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := b.handler(ctx, command.visible, "mqtt_command"); err != nil {
		b.logger.Error("failed to apply MQTT command", "visible", command.visible, "error", err)
	}
}

func (b *Bridge) handleHABirth(_ mqtt.Client, message mqtt.Message) {
	if strings.EqualFold(strings.TrimSpace(string(message.Payload())), "online") {
		b.logger.Info("Home Assistant birth message received; republishing discovery")
		go func() {
			time.Sleep(2 * time.Second)
			if err := b.PublishDiscovery(); err != nil {
				b.logger.Error("failed to republish MQTT discovery", "error", err)
			}
			if err := b.PublishState(); err != nil {
				b.logger.Error("failed to republish MQTT state", "error", err)
			}
		}()
	}
}

func parseSwitchPayload(payload string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(payload)) {
	case "on", "true", "1":
		return true, nil
	case "off", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("unsupported switch payload %q", payload)
	}
}

func (b *Bridge) publishAvailability(payload string) error {
	return b.publish(b.availabilityTopic(), []byte(payload), true)
}

func (b *Bridge) publish(topic string, payload []byte, retained bool) error {
	if b.client == nil || !b.client.IsConnected() {
		return errors.New("MQTT client is not connected")
	}
	token := b.client.Publish(topic, 1, retained, payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("timed out publishing to %s", topic)
	}
	return token.Error()
}

func (b *Bridge) discoveryTopic() string {
	return fmt.Sprintf("%s/switch/%s_display_clock/config", b.cfg.DiscoveryPrefix, b.cfg.DeviceID)
}

func (b *Bridge) commandTopic() string {
	return b.cfg.BaseTopic + "/set"
}

func (b *Bridge) stateTopic() string {
	return b.cfg.BaseTopic + "/state"
}

func (b *Bridge) attributesTopic() string {
	return b.cfg.BaseTopic + "/attributes"
}

func (b *Bridge) availabilityTopic() string {
	return b.cfg.BaseTopic + "/availability"
}
