### TODO

Improvements that would be made with more time:

#### Reliability
- Add retry with exponential backoff on failed downloads
- Add checksum verification for downloaded content (if the manifest included a hash field)
- Implement a healthcheck endpoint for container orchestration

#### Observability
- Add structured logging (e.g. slog) with log levels instead of plain log.Printf
- Add Prometheus metrics for poll count, download latency, failure rate, and content staleness

#### Testing
- Add integration tests that spin up the stub server and syncer together
- Add manifest client tests covering 304, 401, and 500 responses
- Add downloader tests verifying atomic write behaviour and cleanup on failure

#### Scalability
- Add jitter to the poll interval to prevent thundering herd across thousands of devices
- Support bandwidth throttling for large content downloads on constrained networks
- Add concurrent downloads with a configurable worker pool

#### Security
- Validate TLS certificates on content download URIs
- Add request signing or mutual TLS for device authentication

#### MQTT
- Replace the stdout publisher with a real MQTT client (e.g. eclipse/paho.mqtt.golang)
- Support configurable MQTT broker address and QoS levels