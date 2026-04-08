# srvdog Launcher Scripts

These scripts open an SSH tunnel to the srvdog service running on the server and then open the dashboard in your default browser.

## Files

- `open-srvdog.ps1` for Windows PowerShell
- `open-srvdog.sh` for macOS terminal

## What They Do

- check whether local port `8090` is already in use
- start an SSH tunnel to `107.174.48.241:45678`
- forward local `127.0.0.1:8090` to remote `127.0.0.1:8090`
- open `http://127.0.0.1:8090`
- keep the SSH session in the foreground

Closing the terminal closes the tunnel.

## Windows Usage

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\open-srvdog.ps1
```

## macOS Usage

```bash
chmod +x ./scripts/open-srvdog.sh
./scripts/open-srvdog.sh
```

## Optional Identity File

By default the scripts first try:

- `SRVDOG_IDENTITY_FILE`, if you set it
- then the host-specific key:
  - `~/.ssh/id_ed25519_racknerd_107_174_48_241`
- then normal SSH default key and config lookup if no explicit file is found

If you want to force a specific private key, set `SRVDOG_IDENTITY_FILE`.

### Windows

```powershell
$env:SRVDOG_IDENTITY_FILE="$HOME\.ssh\id_ed25519_racknerd_107_174_48_241"
powershell -ExecutionPolicy Bypass -File .\scripts\open-srvdog.ps1
```

### macOS

```bash
export SRVDOG_IDENTITY_FILE="$HOME/.ssh/id_ed25519_racknerd_107_174_48_241"
./scripts/open-srvdog.sh
```

## Changing Host Or Port

Edit the variables at the top of each script:

- host
- SSH port
- local bind host and port
- remote bind host and port

## Manual Fallback

If automatic browser opening fails, open this URL manually:

```text
http://127.0.0.1:8090
```
