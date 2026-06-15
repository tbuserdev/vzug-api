# V-Zug Display Clock Controller

A Bun-based HTTP controller for a V-Zug display clock, with scheduled show/hide automation and manual control endpoints.

This is an unofficial personal project and is not affiliated with, endorsed by, or supported by V-ZUG.

## Configuration

Create a local `.env` file from the example below before running the service:

```bash
cp .env.example .env
```

Environment variables:

- `BASE_URL`: Base URL for the V-Zug device, for example `http://device.local`
- `PORT`: HTTP port for this service, default `3000`
- `HOST_PORT`: Host port when using Docker Compose, default `9999`
- `ALLOW_INSECURE_TLS`: Set to `true` only if you need to talk to a self-signed HTTPS endpoint

## Features

- **Scheduled Tasks**: Automatically show the clock at 22:00 and hide it at 06:00 daily.
- **Manual Control**: Use the HTTP API to toggle, show, or hide the clock manually.
- **Retry Logic**: Handles temporary server unavailability with retries.

## Installation

Install dependencies using Bun:

```bash
bun install
```

## Usage

To run the project:

```bash
bun start
```

To build and run a local Docker image:

```bash
bun run docker:build
IMAGE=vzug-api docker compose up
```

## Quality Checks

Run the same checks used by CI:

```bash
bun run check
```

Individual commands are also available:

- `bun run format:check`
- `bun run lint`
- `bun run typecheck`
- `bun run build`

## Docker Image

GitHub Actions builds the Docker image on every pull request and publishes a multi-architecture image to GitHub Container Registry on pushes to `main`:

```text
ghcr.io/tbuserdev/vzug-api:latest
```

For home deployment, create a `.env` file and start the published image:

```bash
cp .env.example .env
docker compose pull
docker compose up -d
```

If `docker compose pull` returns `denied`, make the GHCR package public in GitHub:

1. Open the `vzug-api` package page under the repository or account packages.
2. Open package settings.
3. Change package visibility to public.

Alternatively, keep the package private and run `docker login ghcr.io` on the home server with a token that has `read:packages` access.

## HTTP API Endpoints

- **Toggle Clock**: `GET /toggle?value=true|false`
- **Show Clock**: `GET /show`
- **Hide Clock**: `GET /hide`
- **Cron Schedule**: `GET /cron`

The server runs on port `3000` by default. Set the `PORT` environment variable to use a different port.

## Notes

- Ensure the V-Zug device is reachable at the configured `BASE_URL`.
- TLS certificate verification is enabled by default. Set `ALLOW_INSECURE_TLS=true` only for local devices that require a self-signed HTTPS certificate.

## License

MIT
