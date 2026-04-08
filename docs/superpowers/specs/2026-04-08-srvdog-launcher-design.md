# srvdog Launcher Design

**Date:** 2026-04-08

**Goal:** Add simple local launcher scripts for Windows and macOS that open an SSH tunnel to the srvdog service and automatically open the dashboard in the default browser.

## Context

- `srvdog` is deployed on the VPS and listens only on `127.0.0.1:8090`.
- The dashboard is intentionally not exposed to the public network.
- Access should remain based on SSH access rather than adding a separate web login system.
- The user wants a simpler workflow than manually typing the tunnel command each time.
- The launcher should prefer standard SSH defaults instead of hardcoding a private key path.

## Requirements

### Functional

- Provide a Windows launcher script.
- Provide a macOS launcher script.
- Each launcher must:
  - check whether local port `8090` is already in use
  - open an SSH local-forward tunnel from local `127.0.0.1:8090` to remote `127.0.0.1:8090`
  - use:
    - host `107.174.48.241`
    - port `45678`
    - user `root`
  - rely on the local SSH client’s default key selection and config behavior
  - open `http://127.0.0.1:8090` in the system default browser after the tunnel is established
  - keep the SSH process in the foreground so errors remain visible
- If the tunnel fails, the script must print the error and exit.
- If local port `8090` is already occupied, the script must stop and print a clear message.

### Non-Functional

- No extra runtime dependency beyond standard platform tools.
- No custom login flow.
- No background daemon management.
- No hardcoded private key path by default.

## Chosen Approach

Add two scripts:

- `scripts/open-srvdog.ps1`
- `scripts/open-srvdog.sh`

Each script will:

1. define a few top-level configuration variables
2. verify the local port is free
3. start an SSH tunnel in the foreground
4. open the local dashboard URL in the default browser
5. print a short usage note explaining that closing the terminal also closes the tunnel

This is the lowest-complexity solution with the least maintenance burden.

## Alternatives Considered

### One cross-platform script

- Rejected because it would likely require Python, Node, or more complex conditional logic.
- Adds dependencies without meaningful benefit for this small feature.

### Background tunnel manager

- Rejected because it complicates lifecycle management.
- Makes debugging harder when SSH fails.
- User explicitly benefits from seeing the tunnel session directly in the terminal.

### Hardcoded private key path

- Rejected because standard SSH defaults are more portable.
- Keeps the launcher compatible with `~/.ssh/config` and normal key discovery.

## Launcher Behavior

### Shared configuration

Both scripts should define these defaults near the top:

- `HOST=107.174.48.241`
- `PORT=45678`
- `USER=root`
- `LOCAL_HOST=127.0.0.1`
- `LOCAL_PORT=8090`
- `REMOTE_HOST=127.0.0.1`
- `REMOTE_PORT=8090`
- `URL=http://127.0.0.1:8090`

These should be easy to edit later if deployment details change.

### Port check

Before opening the tunnel:

- Windows script checks whether local `127.0.0.1:8090` or port `8090` is already bound
- macOS script checks with a standard local port probe

If occupied:

- print a concise error
- do not try to reuse or kill the existing process

### SSH invocation

Base SSH behavior:

- use `ssh -N`
- use local port forwarding `-L`
- do not specify `-i` by default
- let the local SSH client use normal key/config resolution

Example target command shape:

```bash
ssh -N -L 8090:127.0.0.1:8090 -p 45678 root@107.174.48.241
```

### Browser opening

#### Windows

- Use the default browser through PowerShell or `Start-Process`

#### macOS

- Use:

```bash
open http://127.0.0.1:8090
```

The browser-open step should happen after a short delay or after the tunnel process has started successfully enough that the browser does not race too early.

### Shutdown model

- SSH stays attached to the terminal
- Closing the terminal or interrupting the process closes the tunnel
- No PID file
- No background persistence

## Error Handling

### Port already in use

Message should tell the user:

- local port `8090` is already occupied
- they should close the conflicting process or change the local port in the script

### SSH authentication failure

The script should not hide SSH output.

This is important because:

- some machines may not have the correct SSH key loaded
- users need to see `Permission denied (publickey)` and similar errors directly

### Browser open failure

If browser launch fails:

- tunnel should still continue
- print the local URL so the user can open it manually

## File Layout

- `scripts/open-srvdog.ps1`
- `scripts/open-srvdog.sh`
- `scripts/README.md`

## Documentation

`scripts/README.md` should include:

- what the scripts do
- Windows usage
- macOS usage
- how to stop the tunnel
- how to change host, port, and local bind settings
- how to add a private key manually if the SSH default path is not enough

## Security Notes

- The launchers do not reduce SSH security.
- Access still depends on who can authenticate over SSH.
- The dashboard remains bound to `127.0.0.1` on the server.
- The tunnel only exposes the page to the user’s local machine while the SSH session is active.

## Testing Plan

### Windows

- Run `open-srvdog.ps1`
- Verify local browser opens `http://127.0.0.1:8090`
- Verify dashboard loads
- Verify closing the terminal drops access
- Verify failure message when port `8090` is already occupied

### macOS

- Run `./open-srvdog.sh`
- Verify browser opens the same URL
- Verify dashboard loads
- Verify failure mode and closure behavior

## Scope Boundaries

This feature does not include:

- background tunnel services
- tray icons
- web authentication
- server-side changes
- SSH config file modification

## Implementation Recommendation

Keep the launcher scripts very small and explicit.

They should act as convenience wrappers around the existing SSH tunnel workflow, not as a second deployment or security layer.
