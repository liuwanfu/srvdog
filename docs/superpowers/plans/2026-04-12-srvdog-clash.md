# srvdog Clash Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `Clash` management tab to `srvdog` for subscription status, config editing, script editing, geodata updates, log viewing, and immediate token rotation.

**Architecture:** Keep `srvdog` as a single Go binary. Add a focused `internal/clash` package to own subscription files, drafts, publish flow, token rotation, and log aggregation, then expose it through new HTTP API routes and a new frontend tab inside the existing embedded UI.

**Tech Stack:** Go standard library, existing `srvdog` HTTP server, plain HTML/CSS/JS, Linux filesystem operations, selected external commands (`docker run ... mihomo -t`, `update-mihomo-geodata.sh`)

---

### Task 1: Add Clash domain model and status read path

**Files:**
- Create: `internal/clash/manager.go`
- Create: `internal/clash/manager_test.go`
- Modify: `internal/app/service.go`

- [ ] Write failing tests for reading token, published config paths, subscription URLs, and log file metadata from a temp directory fixture.
- [ ] Run `go test ./internal/clash -run TestStatus -v` and confirm it fails because the package does not exist.
- [ ] Implement the minimal `Manager`, config paths, and `Status()` behavior to make the tests pass.
- [ ] Run `go test ./internal/clash -run TestStatus -v`.

### Task 2: Add config/script draft, validate, and publish flows

**Files:**
- Modify: `internal/clash/manager.go`
- Modify: `internal/clash/manager_test.go`

- [ ] Write failing tests for saving drafts, publishing config, publishing script text, and rejecting invalid publish attempts.
- [ ] Run the targeted tests and confirm they fail for the right reason.
- [ ] Implement draft persistence under `data/clash/`, temporary publish staging, and command-runner hooks for validation.
- [ ] Re-run the targeted tests until they pass.

### Task 3: Add geodata update, token rotation, and log aggregation

**Files:**
- Modify: `internal/clash/manager.go`
- Modify: `internal/clash/manager_test.go`

- [ ] Write failing tests for geodata update command dispatch, immediate token rotation, old-token directory removal, and log tail aggregation.
- [ ] Run the targeted tests and confirm the failure is expected.
- [ ] Implement command dispatch and file mutation behavior.
- [ ] Re-run `go test ./internal/clash -v`.

### Task 4: Expose Clash APIs

**Files:**
- Modify: `internal/httpapi/server.go`
- Modify: `internal/httpapi/server_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/service.go`

- [ ] Write failing HTTP tests for `GET /api/clash/status`, config/script read-write routes, validate/publish routes, geodata update, token rotation, and logs.
- [ ] Run `go test ./internal/httpapi -v` and confirm failure.
- [ ] Implement dependency wiring and handlers with minimal JSON contracts.
- [ ] Re-run `go test ./internal/httpapi -v`.

### Task 5: Add Clash tab UI

**Files:**
- Modify: `web/index.html`
- Modify: `web/app.js`
- Modify: `web/styles.css`

- [ ] Write a minimal failing frontend-facing test if practical; otherwise lock behavior with API tests first and implement the smallest UI layer.
- [ ] Add top-level tab navigation and the `Clash` screen.
- [ ] Add panels for status, YAML editor, script editor, operation logs, geodata update, and token rotation.
- [ ] Verify the embedded UI still loads and existing overview behavior remains intact.

### Task 6: Build and publish

**Files:**
- Modify: `README.md`

- [ ] Run `go test ./...`.
- [ ] Build Linux binary with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o srvdog-linux-amd64 ./cmd/srvdog`.
- [ ] SSH to the server, back up `/opt/srvdog/srvdog`, upload the new binary, restart `srvdog.service`, and verify `127.0.0.1:8090`.
- [ ] Commit the repo changes and push if remote permissions allow.
