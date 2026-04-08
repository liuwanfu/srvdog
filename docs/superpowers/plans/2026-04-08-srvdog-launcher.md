# srvdog Launcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add simple Windows and macOS launcher scripts that open an SSH tunnel to srvdog and launch the dashboard in the default browser.

**Architecture:** Two small platform-native scripts wrap the existing SSH tunnel workflow. They check whether local port `8090` is free, start the tunnel in the foreground using the local SSH client, and trigger the default browser after a short delay.

**Tech Stack:** PowerShell, POSIX shell, OpenSSH client, native browser open commands

---

### Task 1: Add launcher scripts

**Files:**
- Create: `C:\Users\liuwa\srvdog\scripts\open-srvdog.ps1`
- Create: `C:\Users\liuwa\srvdog\scripts\open-srvdog.sh`
- Create: `C:\Users\liuwa\srvdog\scripts\README.md`

- [ ] **Step 1: Add Windows launcher**
- [ ] **Step 2: Add macOS launcher**
- [ ] **Step 3: Add script usage documentation**

### Task 2: Update project docs

**Files:**
- Modify: `C:\Users\liuwa\srvdog\README.md`

- [ ] **Step 1: Add launcher section**
- [ ] **Step 2: Describe Windows and macOS usage**

### Task 3: Verify and commit

**Files:**
- Modify: `C:\Users\liuwa\srvdog\scripts\open-srvdog.ps1`
- Modify: `C:\Users\liuwa\srvdog\scripts\open-srvdog.sh`

- [ ] **Step 1: Verify PowerShell script parses**
- [ ] **Step 2: Verify shell script parses with `bash -n`**
- [ ] **Step 3: Commit launcher changes**
