# device-content-sync

A service that runs on IoT devices to keep locally stored content in sync with a cloud manifest. It polls a manifest endpoint for changes, downloads new or updated content, removes stale content, and publishes events to notify other on-device services.

## Prerequisites

- Go 1.26+
- Docker and Docker Compose (if choosing to run with one)

## Running Locally (without Docker)

Start the stub manifest server:
```bash
go run cmd/stub-server/main.go
```

In a separate terminal, start the syncer:
```bash
go run cmd/syncer/main.go
```

Downloaded content will appear in `/tmp/winnow`.

## Running with Docker Compose
```bash
docker compose up --build
```

This starts both the stub server and the syncer. The syncer polls every 10 seconds by default.

## Testing Changes

The stub server exposes admin endpoints to simulate changes between poll cycles. For example, if using curl:
```bash
curl -X POST "http://localhost:8080/admin?action=add-icon"
curl -X POST "http://localhost:8080/admin?action=update-menu"
curl -X POST "http://localhost:8080/admin?action=remove-icon"
curl -X POST "http://localhost:8080/admin?action=unavailable-icons"
curl -X POST "http://localhost:8080/admin?action=available-icons"
```

## Running Tests
```bash
go test ./...
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|---|---|---|
| MANIFEST_URL | http://localhost:8080 | Base URL of the manifest server |
| DEVICE_TOKEN | test-device-001 | Device authentication token |
| CONTENT_DIR | /tmp/winnow | Directory for downloaded content |
| STATE_PATH | /tmp/winnow/.sync-state.json | Path for state persistence file |
| POLL_INTERVAL | 30s | How often to poll the manifest |

## Project Structure
```
cmd/
  syncer/          Service entrypoint
  stub-server/     Mock manifest and content server for testing
internal/
  manifest/        HTTP client and response models
  syncer/          Core polling, diffing, and orchestration logic
  downloader/      Content download with atomic file writes
  publisher/       Event publishing interface (stdout implementation)
```

## Design Decisions

- **Dynamic content types**: The manifest response is decoded as a map rather than a fixed struct, so new content types (beyond menus and icons) are handled without code changes.
- **Atomic file writes**: Content is written to a temporary file and renamed into place, preventing corrupted files if the device loses power mid-download.
- **State persistence**: A JSON state file tracks what has been downloaded and its ETag. On restart, the service resumes without re-downloading unchanged content.
- **Graceful shutdown**: The service listens for SIGINT/SIGTERM and saves state before exiting.
- **Stale over missing**: If a download fails or a content type is marked unavailable, existing content is preserved rather than removed.
- **ETag-based caching**: The manifest ETag is sent via If-None-Match to avoid unnecessary payload transfers across thousands of devices.
- **Interface-driven design**: The downloader and publisher are defined as interfaces, making them independently testable and swappable (e.g. replacing the stdout publisher with a real MQTT client).