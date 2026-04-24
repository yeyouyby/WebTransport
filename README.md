# WebTransport Matrix Gateway

High-performance multimedia encrypted distribution gateway with WebTransport primary transport and HTTP/2 fallback path.

## Highlights

- Stream-based decryption pipeline with offset-aligned ChaCha20 processing.
- Storage scheduler with SA weighted rotation, rate limiting, and retry backoff.
- WebTransport request handling with reliable stream path integration.
- HTTP/2 fallback with range forwarding and exact-SNI TLS certificate matching.
- Media sharding for image, video, and audio with browser-side shard client script.

## Project Layout

- `cmd/gateway`: service entrypoint and dependency wiring.
- `internal/crypto`: stream decrypt logic and offset alignment.
- `internal/storage`: GDrive range fetch and SA pool scheduler.
- `internal/transport`: provider abstraction and WebTransport server/provider.
- `internal/fallback`: HTTP/2 fallback server, shard config API, static shard client script.
- `internal/sharding`: deterministic shard host selection by resource type.
- `scripts`: install, pressure test, and external validation scripts.

## Quick Start

### Linux one-click install

```bash
bash scripts/install_linux_oneclick.sh
```

### Windows one-click install

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install_windows_oneclick.ps1
```

### Run pressure test

```bash
bash scripts/run_pressure_test.sh --url https://127.0.0.1:8443/fallback --requests 1000 --concurrency 20
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/run_pressure_test.ps1 -Url https://127.0.0.1:8443/fallback -Requests 1000 -Concurrency 20
```

## Notes

- `third_party/quic-go-brutal` is vendored and referenced via `go.mod replace`.
- External weak-network benchmark and H2 anti-coalescing validation guides are under `ops/external`.
