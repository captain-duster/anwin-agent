# ANWIN Local Code Sync Agent

The ANWIN Agent watches your local codebase and automatically syncs it with your ANWIN project whenever you make changes. Once set up, it runs silently in the background with no manual steps required.

---

## System Requirements

| Platform | Supported |
|---|---|
| macOS (Intel) | ✓ |
| macOS (Apple Silicon / M1, M2, M3) | ✓ |
| Windows 10 / 11 (64-bit) | ✓ |
| Linux (Ubuntu, Debian, Kali, CentOS, RHEL, Arch, Alpine) | ✓ |
| Raspberry Pi / ARM Linux | ✓ |

No runtime or dependencies required. The agent is a single binary file.

---

## First Time Setup

### Step 1 — Get your Agent Token

1. Log in to ANWIN
2. Open your Project
3. Go to **Project Settings → Local Agent**
4. Click **Generate Agent Token**
5. Copy the token shown — **it will only be shown once**
6. Also note your **Project ID** shown on the same screen

### Step 2 — Download the Agent

Download the correct binary for your system from the ANWIN website or your admin:

| Your System | File to Download |
|---|---|
| macOS Apple Silicon (M1/M2/M3) | `anwin-agent-mac-apple-silicon` |
| macOS Intel | `anwin-agent-mac-intel` |
| Windows 64-bit | `anwin-agent-windows-amd64.exe` |
| Linux 64-bit | `anwin-agent-linux-amd64` |
| Linux ARM (Raspberry Pi) | `anwin-agent-linux-arm64` |

### Step 3 — Make it executable (macOS and Linux only)

Open Terminal and run:

```bash
chmod +x anwin-agent-*
```

Optionally move it to a permanent location:

```bash
# macOS / Linux — move to system path so it works from anywhere
sudo mv anwin-agent-linux-amd64 /usr/local/bin/anwin-agent

# Or keep it in your home directory
mv anwin-agent-linux-amd64 ~/anwin-agent
```

On **Windows**, no extra steps needed — double-click the `.exe` or run it from Command Prompt / PowerShell.

### Step 4 — Run Setup (one time only)

```bash
anwin-agent setup
```

You will be prompted for:

```
ANWIN Server URL   : https://app.anwin.ai
Agent Token        : (paste the token you copied from Step 1)
Project ID         : (paste the project ID from ANWIN)
Directory to watch : /home/yourname/projects/my-codebase
```

The directory to watch should be the **root folder of your codebase** — the same folder you would open in your IDE.

If setup succeeds, you will see:

```
  ✓ Setup complete

  Run 'anwin-agent start' to begin syncing.
```

---

## Every Day Usage

### Start the agent

```bash
anwin-agent start
```

The agent will:
1. Connect to the ANWIN server
2. Upload all files in your codebase (initial sync)
3. Watch for changes and sync automatically as you code

You will see output like:

```
2025-03-10 09:00:01  INFO   Agent starting         version=1.0.0  project=abc123
2025-03-10 09:00:02  INFO   Server connection established
2025-03-10 09:00:02  INFO   Starting initial directory scan...
2025-03-10 09:00:04  INFO   Scan complete          files=312  skipped=14  batches=7
2025-03-10 09:00:06  INFO   Initial sync complete  files=312  duration=2.14s
2025-03-10 09:00:06  INFO   Watching for changes. Press Ctrl+C to stop.

2025-03-10 09:05:22  INFO   Change detected        file=src/service/UserService.java  event=modified
2025-03-10 09:05:24  INFO   Sync complete          files=1  duration=0.28s
```

### Stop the agent

Press `Ctrl + C` at any time. The agent will cleanly flush any pending changes before exiting.

---

## Running in the Background (so it keeps running after you close Terminal)

### macOS — run in background

```bash
nohup anwin-agent start > ~/anwin-agent.log 2>&1 &
echo "Agent running. PID: $!"
```

To stop it later:
```bash
pkill -f anwin-agent
```

### Linux — run as a systemd service (runs on boot, auto-restarts)

Create a service file:

```bash
sudo nano /etc/systemd/system/anwin-agent.service
```

Paste this (replace the paths with your actual values):

```ini
[Unit]
Description=ANWIN Local Code Sync Agent
After=network.target

[Service]
Type=simple
User=YOUR_USERNAME
ExecStart=/usr/local/bin/anwin-agent start
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable anwin-agent
sudo systemctl start anwin-agent
```

Check status:

```bash
sudo systemctl status anwin-agent
```

### Windows — run in background

Open **Command Prompt as Administrator** and run:

```cmd
sc create ANWINAgent binPath= "C:\path\to\anwin-agent-windows-amd64.exe start" start= auto
sc start ANWINAgent
```

Or simply minimize the Command Prompt window after running `anwin-agent start`.

---

## Checking Status

At any time you can check if the agent is configured and can reach the server:

```bash
anwin-agent status
```

Output:

```
  ANWIN Agent Status
  ─────────────────────────────────────
  Version  : 1.0.0
  Server   : https://app.anwin.ai
  Project  : abc123
  Watching : /home/user/my-project
  Status   : ✓ Connected
  Directory: ✓ Exists
```

---

## Reconfiguring (if your token changes or you switch projects)

### Reset and re-run setup

```bash
anwin-agent reset
anwin-agent setup
```

---

## Troubleshooting

### "Not configured" error
Run `anwin-agent setup` first.

### "Cannot reach ANWIN server"
- Check your internet connection
- Make sure the Server URL does not have a trailing slash
- Try opening the server URL in your browser

### "Authentication failed — check your agent token"
- Your token may have expired or been revoked
- Go to ANWIN → Project Settings → Local Agent and generate a new token
- Run `anwin-agent reset` then `anwin-agent setup` with the new token

### "Directory not found"
- Make sure you are using the full absolute path (e.g., `/home/user/myproject`, not `~/myproject`)
- On Windows use backslashes: `C:\Users\Name\myproject`

### "Access denied"
- Your token may have been revoked by your admin
- Contact your ANWIN administrator

### Permission denied on macOS/Linux
```bash
chmod +x anwin-agent
```

### macOS blocks the binary (unverified developer warning)
```bash
xattr -d com.apple.quarantine anwin-agent-mac-apple-silicon
```

---

## What Files Does the Agent Sync?

The agent syncs source code files only. It automatically ignores:

- `node_modules/`, `vendor/`, `.git/`, `target/`, `build/`, `dist/`
- Binary files, compiled outputs, cache files
- Any file over 1 MB

Supported file types include: `.java`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.go`, `.rs`, `.kt`, `.xml`, `.yml`, `.yaml`, `.json`, `.sql`, `.properties`, `.tf`, `.cs`, `.rb`, `.php`, `.swift`, `.scala`, `.md`, `Dockerfile`, `Makefile`, and more.

---

## Security

- Your agent token is stored **encrypted** on your machine using AES-256-GCM
- All communication with the ANWIN server uses **TLS 1.2 or higher**
- Your token is **never stored in plain text** — only a SHA-256 hash is held on the server
- The agent only **reads** your files — it never writes, moves, or deletes anything locally
- You can revoke the agent's access at any time from **ANWIN → Project Settings → Local Agent → Revoke**

---

## Uninstalling

```bash
# Remove the binary
rm /usr/local/bin/anwin-agent

# Remove saved configuration
rm -rf ~/.anwin

# Linux only — remove systemd service if created
sudo systemctl stop anwin-agent
sudo systemctl disable anwin-agent
sudo rm /etc/systemd/system/anwin-agent.service
sudo systemctl daemon-reload
```

---

## Version

```bash
anwin-agent version
```
