# speedtest-local-go

A self-hosted, speedtest.net-style web page that measures ping, jitter, download and upload speed on your local network. One static Go binary, with the web UI embedded. No Python, no runtime, no extra files.

## Build

Requires Go 1.21+.

```bash
./build.sh                      # all targets -> ./dist
./build.sh android/arm64        # only one target
# or manually:
go build -trimpath -ldflags "-s -w" -o speedtest .
```

Targets: `linux/amd64`, `linux/arm64`, `android/arm64`, `android/amd64`, `windows/amd64`, `darwin/arm64`. Binaries are about 5 MB. `android/amd64` needs cgo in Go, so `build.sh` builds it as a static `linux/amd64` binary, which runs fine on Android x86_64. If `android/arm64` misbehaves on your device, `linux/arm64` also runs on rooted Android.

## Run

```bash
./speedtest
```

Open `http://<server-lan-ip>:8080` from any device on the network. The URLs are printed on startup.

## Configuration

Environment variables, or a `.env` file next to the binary or in the working directory (see `.env.example`). Real environment variables win.

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | Bind address |
| `PORT` | `8080` | Port |
| `SPEEDTEST_ISP` | `Local Network` | Name shown next to the client IP |
| `SPEEDTEST_SERVER_NAME` | hostname | Server name shown in the UI |
| `SPEEDTEST_LOCATION` | `LAN` | Server location shown in the UI |
| `MAX_DOWNLOAD_MB` | `1024` | Max size of one download request |
| `TRUST_PROXY` | `0` | Set `1` behind nginx/cloudflared to read the real client IP |

## API

- `GET /api/info` - client IP and server info
- `GET /api/ping` - empty response for latency
- `GET /api/download?size=<bytes>` - streams random data
- `POST /api/upload` - discards the body, returns byte count
- `GET /healthz` - health check

## Notes

- The UI in `static/` is embedded at build time. Edit the files, then rebuild.
- Multi mode uses parallel streams (default 6, configurable in Settings); Single uses one.
- The first 1.5 s of each phase is excluded to skip TCP slow start.
