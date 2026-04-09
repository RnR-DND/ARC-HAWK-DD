# Agent Install Guide

The Hawk Agent is a lightweight binary that runs on the machine where your data lives. It executes scheduled scans, buffers results locally when the server is unreachable, and syncs when connectivity is restored.

---

## Prerequisites

- **Admin or root privileges** are required for installation and to run as a system service.
- Outbound HTTPS access to your ARC-Hawk server (default port 443 or 8080).
- For Linux systemd install: `systemd` v232+ (standard on Ubuntu 18.04+, Debian 9+, CentOS 7+).

---

## Download

Binaries are built for each release. Download the binary for your platform from the releases page or your internal artifact store:

| Platform | Binary name |
|----------|-------------|
| macOS (ARM64 / Apple Silicon) | `hawk-agent-mac` |
| Linux (x86-64) | `hawk-agent-linux` |
| Windows (x86-64) | `hawk-agent.exe` |

---

## macOS Install

```bash
# 1. Download the binary
curl -Lo hawk-agent-mac https://releases.arc-hawk.io/latest/hawk-agent-mac

# 2. Make it executable
chmod +x hawk-agent-mac

# 3. Run the agent (requires admin for privileged paths)
sudo ./hawk-agent-mac
```

The agent exposes a local health endpoint at `http://localhost:9090/health` once running.

To run as a LaunchDaemon on startup, create `/Library/LaunchDaemons/io.arc-hawk.agent.plist` and load it with `launchctl`.

---

## Linux Install (systemd)

```bash
# 1. Download the binary
curl -Lo /usr/local/bin/hawk-agent-linux \
  https://releases.arc-hawk.io/latest/hawk-agent-linux

# 2. Make it executable
chmod +x /usr/local/bin/hawk-agent-linux

# 3. Create a dedicated system user
useradd --system --no-create-home hawk

# 4. Create the data directory
mkdir -p /var/lib/hawk-agent
chown hawk:hawk /var/lib/hawk-agent

# 5. Create the systemd unit file
cat > /etc/systemd/system/hawk-agent.service <<'EOF'
[Unit]
Description=ARC-Hawk Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=hawk
EnvironmentFile=/etc/hawk-agent/env
ExecStart=/usr/local/bin/hawk-agent-linux
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=hawk-agent

[Install]
WantedBy=multi-user.target
EOF

# 6. Create the environment file (see Configuration section)
mkdir -p /etc/hawk-agent
cat > /etc/hawk-agent/env <<'EOF'
HAWK_SERVER_URL=https://your-arc-hawk-server:8080
HAWK_AGENT_ID=your-agent-id
HAWK_AGENT_CLIENT_SECRET=your-client-secret
HAWK_DATA_DIR=/var/lib/hawk-agent
EOF
chmod 600 /etc/hawk-agent/env

# 7. Enable and start
systemctl daemon-reload
systemctl enable hawk-agent
systemctl start hawk-agent

# 8. Verify
systemctl status hawk-agent
curl http://localhost:9090/health
```

---

## Windows Install

**Run as Administrator** using one of two approaches:

### Option A: Interactive (development/testing)

```powershell
# Download
Invoke-WebRequest -Uri https://releases.arc-hawk.io/latest/hawk-agent.exe `
  -OutFile hawk-agent.exe

# Set environment variables, then run
$env:HAWK_SERVER_URL = "https://your-arc-hawk-server:8080"
$env:HAWK_AGENT_ID = "your-agent-id"
$env:HAWK_AGENT_CLIENT_SECRET = "your-client-secret"
$env:HAWK_DATA_DIR = "C:\ProgramData\hawk-agent"

.\hawk-agent.exe
```

### Option B: Task Scheduler (production)

1. Open **Task Scheduler** as Administrator.
2. Create a new task: **Action > Create Task**.
3. Set **General > Run whether user is logged on or not** and **Run with highest privileges**.
4. **Actions > New**: Program = `C:\Program Files\hawk-agent\hawk-agent.exe`.
5. **Triggers**: At system startup, repeat indefinitely.
6. Set environment variables via the **Environment Variables** tab or via a wrapper `.bat` file that sets `HAWK_*` vars before launching the binary.

---

## Configuration

The agent reads configuration from `$HAWK_DATA_DIR/config.yaml` and from environment variables. Environment variables take precedence over the config file.

### config.yaml

```yaml
server_url: "https://your-arc-hawk-server:8080"
agent_id: "your-agent-id"
agent_client_secret: "your-client-secret"
data_dir: "/var/lib/hawk-agent"

# Scan schedule in standard 5-field cron format (minute hour dom month dow).
# Default: weekly, Sunday at midnight.
scan_schedule: "0 0 * * 0"

# Offline buffer: maximum SQLite file size in MB before new scans are paused.
buffer_max_mb: 512
```

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `HAWK_SERVER_URL` | Base URL of the ARC-Hawk backend | `https://hawk.internal:8080` |
| `HAWK_AGENT_ID` | Unique identifier for this agent instance | `agent-prod-db-01` |
| `HAWK_AGENT_CLIENT_SECRET` | Client secret for agent authentication | `sk-...` |
| `HAWK_DATA_DIR` | Directory for SQLite buffer and config | `/var/lib/hawk-agent` |

---

## Offline Mode and SQLite Buffer

When the agent cannot reach the server it switches to **offline mode** automatically. Scan results are written to a SQLite database at `$HAWK_DATA_DIR/pending_results.db` using WAL mode for durability.

**How the buffer works:**

1. The sync loop polls the server every 30 seconds.
2. After 5 consecutive successful health checks the agent switches back to **online** and drains the queue in batches of 100 results.
3. If the buffer exceeds `buffer_max_mb` (default 512 MB), the agent purges already-sent rows first, then failed rows. If the buffer is still full, **new scans are paused** until space is reclaimed.
4. On graceful shutdown the agent attempts one final sync before exiting.

**Check queue depth:**

```bash
curl -s http://localhost:9090/health | jq '{queue_depth, connectivity_status, buffer_size_mb, status}'
```

Example healthy response:
```json
{
  "queue_depth": 0,
  "connectivity_status": "online",
  "buffer_size_mb": 0.02,
  "status": "ok"
}
```

If `status` is `degraded_buffer_full`, the offline buffer is at capacity. Ensure connectivity to the server or increase `buffer_max_mb`.

---

## Scan Schedule

Scans run on a **cron schedule** configured via `scan_schedule` in `config.yaml` or the corresponding environment variable.

The format is standard 5-field cron: `minute hour dom month dow`.

| Example | Meaning |
|---------|---------|
| `0 0 * * 0` | Weekly, Sunday at midnight (default) |
| `0 2 * * *` | Daily at 02:00 |
| `0 */6 * * *` | Every 6 hours |
| `30 1 1 * *` | Monthly, 1st day at 01:30 |

If a scan is already running when the next schedule fires, the new run is skipped and a warning is logged. The agent also skips scheduled runs when the offline buffer is at capacity.

---

## Health Endpoint Reference

`GET http://localhost:9090/health`

| Field | Type | Description |
|-------|------|-------------|
| `agent_id` | string | Configured agent ID |
| `uptime` | string | Human-readable uptime |
| `uptime_seconds` | float | Uptime in seconds |
| `last_scan_at` | string | RFC3339 timestamp of last completed scan |
| `queue_depth` | int | Number of pending (unsynced) results |
| `connectivity_status` | string | `online` or `offline` |
| `buffer_size_mb` | float | Current SQLite buffer file size in MB |
| `status` | string | `ok` or `degraded_buffer_full` |

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Agent exits immediately | Missing required env vars | Set `HAWK_SERVER_URL`, `HAWK_AGENT_ID`, `HAWK_AGENT_CLIENT_SECRET` |
| `status: degraded_buffer_full` | Buffer at capacity | Check server connectivity; increase `buffer_max_mb` |
| Scans never run | Invalid cron expression | Verify `scan_schedule` using [crontab.guru](https://crontab.guru) |
| `connectivity_status: offline` persists | Firewall blocking outbound | Allow outbound TCP to `HAWK_SERVER_URL` host and port |
| High `queue_depth` after reconnect | Large backlog | Wait for sync loop to drain; check server logs for 4xx/5xx on sync endpoint |
