# srvdog

Lightweight server resource dashboard for small Linux VPS hosts.

## What It Does

- Binds to `127.0.0.1:8090`
- Intended to be accessed through an SSH tunnel
- Keeps low-frequency background history
- Switches to high-frequency realtime sampling while the page is open
- Shows:
  - CPU
  - load average
  - memory
  - swap
  - root disk usage
  - network realtime throughput
  - network rolling average throughput
  - Docker container summary
- Supports:
  - retention-day changes
  - export as JSON or CSV
  - clearing sampling history

## Access Model

The app is intentionally local-only on the server.

Open an SSH tunnel:

```bash
ssh -L 8090:127.0.0.1:8090 root@107.174.48.241 -p 45678
```

Then open:

```text
http://127.0.0.1:8090
```

## Local Build On Windows For Linux

```powershell
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
& "C:\Program Files\Go\bin\go.exe" build -o srvdog-linux-amd64 ./cmd/srvdog
```

## Test

```powershell
& "C:\Program Files\Go\bin\go.exe" test ./...
```

## Run On Linux Server

```bash
chmod +x srvdog-linux-amd64
./srvdog-linux-amd64
```

The service writes runtime data to:

- `data/settings.json`
- `data/history/*.jsonl`

## Sampling Model

- Low-frequency background sampling:
  - every 5 minutes
- High-frequency realtime sampling:
  - every 2 seconds while an active browser is sending heartbeats
- Docker summary refresh:
  - every 30 seconds

## History

- Default retention: 7 days
- Retention can be changed in the page
- Old history files are cleaned up automatically

## Export And Clear

- Export supports JSON and CSV
- Export includes low-frequency history and the current realtime buffer
- Clear history removes stored sampling history and the in-memory realtime buffer

## Notes

- The runtime target is Linux
- The code is compiled on Windows and deployed to Linux
- Docker summary collection depends on the `docker` CLI being available on the server
