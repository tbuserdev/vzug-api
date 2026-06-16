# V-ZUG Home Assistant Bridge

A small Go daemon that exposes the V-ZUG display clock as a native Home Assistant MQTT switch. It keeps the old scheduled show/hide behavior, removes the browser frontend entirely, and publishes retained MQTT discovery/state/availability messages so Home Assistant can manage the device directly.

This is an unofficial personal project and is not affiliated with, endorsed by, or supported by V-ZUG.

## How It Works

- Sends the V-ZUG device command `setDisplayXclock` over the appliance HTTP endpoint.
- Publishes a Home Assistant MQTT discovery payload for `switch.vzug_display_clock_display_clock`.
- Subscribes to `vzug/display_clock/set` and accepts `ON`, `OFF`, `true`, `false`, `1`, or `0`.
- Publishes retained state to `vzug/display_clock/state`.
- Publishes retained availability to `vzug/display_clock/availability` with MQTT last-will support.
- Runs scheduled show/hide jobs in the configured timezone.
- Keeps a minimal HTTP API for health checks and manual debugging; there is no frontend.

## Configuration

Create a local `.env` file from the example below before running the service:

```bash
cp .env.example .env
```

Environment variables:

- `BASE_URL`: Base URL for the V-ZUG device, for example `http://device.local`
- `PORT`: HTTP port for this service, default `3000`
- `HOST_PORT`: Host port when using Docker Compose, default `9999`
- `ALLOW_INSECURE_TLS`: Set to `true` only if you need to talk to a self-signed HTTPS endpoint
- `HTTP_TIMEOUT`: Per-request timeout for the V-ZUG HTTP call, default `10s`
- `RETRY_COUNT`: Number of retries after a failed device command, default `3`
- `RETRY_DELAY`: Delay between retries, default `5s`
- `TIMEZONE`: Schedule timezone, default `Europe/Zurich`
- `SHOW_SCHEDULE`: Cron expression for turning the clock on, default `0 22 * * *`
- `HIDE_SCHEDULE`: Cron expression for turning the clock off, default `0 6 * * *`
- `MQTT_BROKER`: MQTT broker URL, for example `tcp://homeassistant.local:1883`
- `MQTT_USERNAME`: Optional MQTT username
- `MQTT_PASSWORD`: Optional MQTT password
- `MQTT_CLIENT_ID`: MQTT client ID, default `vzug-ha`
- `MQTT_DISCOVERY_PREFIX`: Home Assistant MQTT discovery prefix, default `homeassistant`
- `MQTT_BASE_TOPIC`: Runtime MQTT topic prefix, default `vzug/display_clock`
- `DEVICE_ID`: Stable Home Assistant device identifier, default `vzug_display_clock`
- `DEVICE_NAME`: Device name shown in Home Assistant, default `V-ZUG Display Clock`

## Installation

Build and run locally:

```bash
go run ./cmd/vzug-ha
```

Run checks:

```bash
go test ./...
go vet ./...
gofmt -w .
```

## Docker

```bash
cp .env.example .env
docker compose up -d
```

To build a local image:

```bash
docker build -t vzug-api .
IMAGE=vzug-api docker compose up
```

GitHub Actions publishes a multi-architecture image to GitHub Container Registry on pushes to `main`:

```text
ghcr.io/tbuserdev/vzug-api:latest
```

## Home Assistant Setup

1. Install and configure the MQTT integration in Home Assistant. The Mosquitto broker add-on is the common choice for Home Assistant OS.
2. Make sure MQTT discovery is enabled. Home Assistant's default discovery prefix is `homeassistant`.
3. Create a `.env` file for this bridge and point it at the same MQTT broker:

```dotenv
BASE_URL=http://your-vzug-device.local
MQTT_BROKER=tcp://homeassistant.local:1883
MQTT_USERNAME=your_mqtt_user
MQTT_PASSWORD=your_mqtt_password
```

4. Start the bridge:

```bash
docker compose up -d
```

5. In Home Assistant, go to Settings -> Devices & services -> MQTT. The device should appear as `V-ZUG Display Clock` with a `Display Clock` switch.

The bridge listens to Home Assistant birth messages on `homeassistant/status` and republishes discovery/state when Home Assistant comes online.

### MQTT Topics

With the default `MQTT_BASE_TOPIC=vzug/display_clock`:

```text
vzug/display_clock/set           # command topic: ON or OFF
vzug/display_clock/state         # retained state: ON or OFF
vzug/display_clock/attributes    # retained JSON attributes
vzug/display_clock/availability  # retained online/offline
```

Discovery is published retained to:

```text
homeassistant/switch/vzug_display_clock_display_clock/config
```

Manual MQTT test:

```bash
mosquitto_pub -h homeassistant.local -t vzug/display_clock/set -m ON
mosquitto_sub -h homeassistant.local -v -t 'vzug/display_clock/#'
```

### Manual YAML Fallback

MQTT discovery is preferred. If discovery is disabled, add this to Home Assistant's `configuration.yaml` and restart:

```yaml
mqtt:
  - switch:
      unique_id: vzug_display_clock_display_clock
      name: Display Clock
      command_topic: vzug/display_clock/set
      state_topic: vzug/display_clock/state
      availability_topic: vzug/display_clock/availability
      payload_on: "ON"
      payload_off: "OFF"
      payload_available: online
      payload_not_available: offline
      optimistic: false
      retain: false
      qos: 1
```

Home Assistant's MQTT documentation is useful background:

- [MQTT integration and discovery](https://www.home-assistant.io/integrations/mqtt/)
- [MQTT switch configuration](https://www.home-assistant.io/integrations/switch.mqtt/)

## HTTP API

- `GET /healthz`
- `GET /state`
- `GET /toggle?value=true|false`
- `GET /show`
- `GET /hide`
- `GET /cron`

The HTTP API is intended for health checks and debugging. Home Assistant should control the bridge through MQTT.

## Notes

- Ensure the V-ZUG device is reachable from the bridge container at the configured `BASE_URL`.
- TLS certificate verification is enabled by default. Set `ALLOW_INSECURE_TLS=true` only for local devices that require a self-signed HTTPS certificate.
- The displayed state is command-confirmed. On startup the bridge does not guess the device state; it publishes switch state only after a V-ZUG HTTP command succeeds.
- Choose a stable `DEVICE_ID`; changing it creates a new device/entity in Home Assistant.

## License

MIT
