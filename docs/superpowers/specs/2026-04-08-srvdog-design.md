# srvdog Design

**Date:** 2026-04-08

**Goal:** Build a lightweight local-only server resource dashboard that runs on the VPS, is accessed through an SSH tunnel, and uses very little CPU, memory, disk, and network while still supporting on-demand realtime inspection plus low-frequency history.

## Context

- Target server is a small VPS with about 1 GB RAM.
- Existing full-featured monitoring UI felt too heavy and not suited to occasional troubleshooting.
- Access should not rely on a web login system.
- The dashboard should be reachable only through SSH port forwarding.
- The repository `srvdog` is currently empty, so the project can start with a focused structure.

## Requirements

### Functional

- Listen only on `127.0.0.1:8090`.
- Be accessed through SSH local forwarding, for example:
  - `ssh -L 8090:127.0.0.1:8090 root@107.174.48.241 -p 45678`
- Show current system resource state:
  - CPU usage
  - load average
  - memory usage
  - swap usage
  - root filesystem disk usage
  - network upload/download realtime throughput
  - network upload/download rolling average over the last few minutes
  - Docker container summary
- Keep low-frequency history for 7 days by default.
- Allow changing retention days from the page.
- Support exporting sampling history.
- Support clearing sampling history.

### Non-Functional

- Keep runtime overhead low enough for a 1 GB VPS.
- Avoid heavy external dependencies and avoid a database unless necessary.
- Keep deployment simple enough to compile locally and copy a single Linux binary to the server.
- Prefer direct reads from `/proc` and similar system files over expensive shell commands.

## Chosen Approach

Use a single Go service with:

- an embedded static frontend
- a lightweight HTTP API
- low-frequency background sampling
- higher-frequency realtime sampling only while at least one browser tab is active
- JSONL history files stored on disk by day
- in-memory ring buffers for short-window realtime data

This is the best tradeoff between simplicity, low resource use, and enough functionality for occasional troubleshooting.

## Alternatives Considered

### Python service

- Faster to prototype
- Heavier runtime dependency story on the VPS
- Less attractive than a single static Go binary for long-term maintenance

### Shell scripts plus static HTML

- Lowest implementation complexity at first
- Awkward for dynamic export, clear-history, heartbeats, and mixed low/high frequency sampling
- Harder to grow cleanly

### Go plus SQLite

- Better structured queries
- More moving parts than needed
- Extra implementation complexity without enough benefit for 7-day lightweight history

## Architecture

### Runtime Model

One process runs continuously:

- HTTP server bound to `127.0.0.1:8090`
- low-frequency sampler
- active-session tracker
- optional high-frequency sampler when viewers are present
- history retention cleanup

### Sampling Modes

#### Low-frequency background mode

- Default mode
- Interval: every 5 minutes
- Purpose: preserve coarse history with minimal overhead
- Output: append one record to the current day JSONL file

#### High-frequency realtime mode

- Enabled only when the frontend is open and sending heartbeat requests
- Interval: every 2 seconds
- Purpose: give responsive live charts during troubleshooting
- Output: store records only in memory in a fixed-size ring buffer
- Automatically disabled after a heartbeat timeout window if no active viewers remain

### Data Sources

#### Read directly from Linux proc/sys files

- CPU stats: `/proc/stat`
- Load average: `/proc/loadavg`
- Memory and swap: `/proc/meminfo`
- Network bytes: `/proc/net/dev`
- Mount stats: `statfs` or equivalent for `/`

#### Docker summary

- Lower-frequency fetch than core host metrics
- Use Docker CLI or Docker API through the local socket
- Collect:
  - container name
  - running status
  - health if available
  - restart count if cheaply available

Docker data does not need 2-second refresh; 30-second refresh is enough.

## Metrics Model

Each resource sample should include:

- `timestamp`
- `cpu_percent`
- `load_1`
- `load_5`
- `load_15`
- `mem_total_bytes`
- `mem_used_bytes`
- `mem_available_bytes`
- `swap_total_bytes`
- `swap_used_bytes`
- `disk_total_bytes`
- `disk_used_bytes`
- `disk_free_bytes`
- `net_rx_bytes_per_sec`
- `net_tx_bytes_per_sec`
- `net_rx_avg_bytes_per_sec`
- `net_tx_avg_bytes_per_sec`

Docker summary should be stored separately from core host samples to keep the main sample format compact.

## History Storage

### On-disk history

- Directory: `data/history/`
- File format: JSONL
- Rotation: one file per UTC day
- Example:
  - `data/history/2026-04-08.jsonl`

Reason:

- append-only writes are simple and cheap
- export is straightforward
- retention cleanup is easy
- no DB dependency

### In-memory realtime buffer

- Keep the last 30 to 60 minutes of realtime samples
- Implement as a ring buffer
- No disk persistence for high-frequency samples

Reason:

- avoids constant disk writes
- keeps realtime mode cheap
- enough for a short troubleshooting session

## HTTP Surface

### Page routes

- `GET /`
  - serves the dashboard

### API routes

- `GET /api/summary`
  - returns latest current values and current Docker summary
- `GET /api/history`
  - returns historical resource samples for the selected time window
- `GET /api/realtime`
  - returns in-memory high-frequency samples
- `POST /api/heartbeat`
  - marks a viewer as active
- `POST /api/settings/retention`
  - updates retention days
- `GET /api/export`
  - exports history data as JSON or CSV
- `POST /api/history/clear`
  - clears stored low-frequency history and in-memory realtime buffers

### Safety

- No auth in the app itself
- Safety boundary is local bind plus SSH tunnel
- No route should bind to `0.0.0.0`

## Frontend Design

### Layout

#### Top summary cards

- CPU
- Load
- Memory
- Swap
- Disk
- Network realtime up/down
- Network rolling average up/down

#### Main charts

- CPU history
- Memory history
- Disk usage history
- Network throughput history

#### Lower panels

- Docker container table
- Retention settings
- Export controls
- Clear-history control

### Frontend technology

- Plain HTML/CSS/JS
- No large frontend framework
- Small charting layer only if needed; otherwise simple SVG or canvas rendering

### Interaction model

- On load:
  - fetch summary
  - fetch selected history range
  - start heartbeat timer
  - start realtime polling
- On close/inactivity:
  - heartbeat stops
  - server eventually falls back to low-frequency mode

## Network Speed Definition

The dashboard will show both:

- realtime upload/download rate
- rolling average upload/download rate over the last few minutes

Implementation:

- compute rate using byte deltas from `/proc/net/dev`
- maintain a short recent window in memory for moving average calculation
- use a single primary interface selected automatically from the default route or most active non-loopback interface

## Retention Rules

- Default retention: 7 days
- User may change retention from the page
- Allowed bounds should be constrained, for example 1 to 30 days
- Retention applies only to low-frequency on-disk history
- Cleanup runs on startup and on a periodic housekeeping timer

## Export Behavior

- Export includes:
  - selected low-frequency history range
  - optionally the current realtime buffer if requested
- Supported formats:
  - JSON first
  - CSV if trivial to add in the same implementation pass

Recommended first pass:

- JSON export mandatory
- CSV export optional if it does not complicate implementation

## Clear-History Behavior

- Remove all low-frequency history files
- Clear in-memory realtime buffers
- Keep current application settings
- Require explicit confirmation in the UI

## Project Structure

- `cmd/watchsrv/main.go`
  - process entrypoint
- `internal/app/`
  - service wiring and lifecycle
- `internal/collector/`
  - CPU, memory, disk, load, network, Docker collectors
- `internal/history/`
  - JSONL persistence, retention cleanup, export
- `internal/realtime/`
  - ring buffer and active-viewer tracking
- `internal/httpapi/`
  - API and page handlers
- `web/`
  - HTML, CSS, JS assets
- `data/`
  - runtime-generated history files, not committed

## Deployment Model

### Local build

- Build Linux binary locally from the repo
- Copy binary and static assets to the server

#### Primary build target

- Development host: Windows
- Runtime host: Linux x86_64 VPS
- Recommended cross-compile target:
  - `GOOS=linux`
  - `GOARCH=amd64`
  - `CGO_ENABLED=0`

Example:

```powershell
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o watchsrv-linux-amd64 ./cmd/watchsrv
```

Server run example:

```bash
chmod +x watchsrv-linux-amd64
./watchsrv-linux-amd64
```

Reason:

- avoids installing Go on the server
- keeps server runtime simple
- works cleanly with the chosen pure-Go plus JSONL architecture

### Server run model

- Run as a systemd service
- Bind to `127.0.0.1:8090`
- Access only through SSH port forwarding

## Risks and Mitigations

### Risk: excessive sampling overhead

- Mitigation:
  - low-frequency background mode
  - high-frequency mode only while viewers are active
  - direct `/proc` reads
  - lower Docker polling frequency

### Risk: Docker metrics becoming expensive

- Mitigation:
  - keep Docker collection coarse and separate from host sampling
  - avoid per-container deep inspection every 2 seconds

### Risk: retention growth consuming disk

- Mitigation:
  - default 7-day retention
  - startup and periodic cleanup
  - compact JSONL records

### Risk: unsafe remote exposure

- Mitigation:
  - bind only to loopback
  - no public listener
  - require SSH access to view the page

## Testing Strategy

- Unit tests for:
  - CPU delta calculation
  - network throughput calculation
  - moving average logic
  - JSONL encode/decode
  - retention cleanup logic
  - active-viewer timeout logic
- Manual verification on Linux:
  - service starts on `127.0.0.1:8090`
  - SSH tunnel access works
  - summary values match system commands
  - history export downloads correctly
  - clear-history removes stored data
  - idle mode returns to low-frequency sampling

## Open Decisions Resolved

- Access model: SSH tunnel only
- UI exposure: local bind on server, no public bind
- Build model: compile locally, run on server
- History retention default: 7 days, adjustable in UI
- Network metric: realtime plus rolling average over the last few minutes
- Log export/clear scope: resource sampling history only

## Implementation Recommendation

Implement the first version with:

- JSON export required
- CSV export only if it stays small and clean
- host metrics first
- Docker summary second
- native browser UI with no heavy frontend framework

This keeps the first usable version small, fast, and appropriate for the target VPS.
