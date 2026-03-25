# Fyers Go Trading System — Commands

## First-Time Setup (Local & EC2)

```bash
# 1. Copy the env template and fill in your Fyers credentials
cp .env.example .env
# Edit .env — set FYERS_APP_ID, FYERS_APP_SECRET, FYERS_ACCESS_TOKEN

# 2. Create the logs directory (mounted into the container)
mkdir -p logs data
```

---

## Running with Docker (Recommended — matches EC2 exactly)

```bash
# Start everything (Redis + trading app)
docker compose up -d

# Watch live logs
docker compose logs -f

# Watch only the trading app logs
docker compose logs -f fyers-trading

# Stop everything (Redis data is preserved)
docker compose down

# Stop and WIPE all data (Redis, logs, IMF cache)
docker compose down -v

# Rebuild after code changes and restart
docker compose build && docker compose up -d

# Restart just the trading app (e.g. after .env change)
docker compose restart fyers-trading
```

---

## Verify Redis is working

```bash
# Ping Redis from inside the container
docker exec fyers-redis redis-cli ping
# Expected: PONG

# Check pipeline state after a few Nifty ticks
docker exec fyers-redis redis-cli GET "fyers:pipeline:active_atm"
docker exec fyers-redis redis-cli GET "fyers:pipeline:active_symbols"

# See all persisted state
docker exec fyers-redis redis-cli KEYS "fyers:*"
```

---

## Hot-reloading Trading Config (no restart needed)

```bash
# Edit trading.json directly — app picks it up within 2 seconds
nano trading.json   # on EC2
# or open trading.json in VS Code locally
```

---

## Tests (run without Docker — no Redis needed)

```bash
# Run all tests
go test ./tests/... -count=1 -timeout 120s

# Run only Redis/infra tests
go test fyers-trading/tests/infra -v -count=1

# Run only pipeline tests
go test fyers-trading/tests/symbols -run TestPipeline -v

# Run with verbose output
go test ./tests/... -v -count=1
```

---

## Running Locally Without Docker (dev only)

```bash
# Requires Go installed locally (not needed on EC2 with Docker)
go run .

# With Redis running locally:
# Set REDIS_URL=redis://localhost:6379/0 in .env
# Then start a Redis container:
docker run -d --name local-redis -p 6379:6379 redis:7-alpine
```

---

## Analytics (analyze captured tick data)

```bash
# Local: analyze ticks.txt from a go run . session
go run ./analytics

# Docker: copy logs out from the container first
docker cp fyers-trading:/app/logs/ticks.txt .
go run ./analytics
```

---

## Dependency Management

```bash
go mod tidy
go mod download
```

---

## EC2 Initial Setup (Install Docker on Amazon Linux 2023)

Run these commands the very first time you boot your EC2 server to install Docker and the Compose plugin:

```bash
# 1. Update the system
sudo dnf update -y

# 2. Install Docker Engine
sudo dnf install -y docker

# 3. Start the Docker service and enable it on boot
sudo systemctl enable --now docker

# 4. Add your user to the docker group (so you don't need sudo)
sudo usermod -aG docker ec2-user

# 5. (Amazon Linux 2023 specific) Install Docker Compose v2 & Buildx Plugins manually
sudo mkdir -p /usr/local/lib/docker/cli-plugins
sudo curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-$(uname -m) -o /usr/local/lib/docker/cli-plugins/docker-compose
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
sudo curl -SL https://github.com/docker/buildx/releases/download/v0.21.1/buildx-v0.21.1.linux-amd64 -o /usr/local/lib/docker/cli-plugins/docker-buildx
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-buildx

# 6. Apply the group change immediately (or simply log out and back in)
newgrp docker

# 7. (Optional) Install Go natively for local debugging
sudo dnf install golang -y
go build ./...

# 7. Fix Redis Background Save / Replication warnings
sudo sysctl vm.overcommit_memory=1
echo 'vm.overcommit_memory = 1' | sudo tee -a /etc/sysctl.conf

# 8. Verify installation
docker version
docker compose version

# ─────────────────────────────────────────────
# Appendix: RingBuffer Latency Optimization
# ─────────────────────────────────────────────
If you ever want to reduce the `40% CPU utilization` down to **0%** (at the cost of adding ~1 millisecond of internal latency), modify the `Next()` function in `marketdata/ringbuffer.go`.

**1. Add "time" to imports:**
```go
import (
	"runtime"
	"sync/atomic"
	"time"
)
```

**2. Add the dynamic backoff inside `Next()`:**
```go
func (r *Reader) Next() *TickEvent {
	spinCount := 0
	for {
		writePos := atomic.LoadUint64(&r.rb.writePos)

		if r.readPos == writePos {
			if atomic.LoadUint32(&r.rb.closed) == 1 {
				return nil 
			}
			// Buffered caught up, spin wait with backoff to prevent 100% CPU lock
			if spinCount < 100 {
				runtime.Gosched()
				spinCount++
			} else {
				time.Sleep(1 * time.Millisecond) // Yield back fully to OS
			}
			continue
		}

		if writePos-r.readPos > r.rb.capacity {
			r.readPos = writePos - 1
		}

		idx := r.readPos & r.rb.mask
		tick := r.rb.buffer[idx]
		r.readPos++
		spinCount = 0 // Reset backoff on successful read
		return tick
	}
}
```
```

---

## EC2 Deployment (Terminal Only)

```bash
# First time: clone and setup
git clone https://github.com/YOUR/fyers.git && cd fyers
cp .env.example .env && nano .env
touch ticks.txt pipeline_events.txt
mkdir -p logs data
docker compose up --build -d

# Update code (subsequent deploys)
git pull && docker compose up --build -d

# Check app is running
docker compose ps
docker compose logs --tail=50 -f fyers-trading
```
