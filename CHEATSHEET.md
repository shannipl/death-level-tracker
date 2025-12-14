# Death Level Tracker - Developer Guide

> **Tibia Death & Level Tracker Discord Bot** - A Go application that monitors Tibia game worlds for player deaths and level ups, posting notifications to Discord channels with production-grade observability.

## Table of Contents

- [Quick Start](#quick-start)
- [Development](#development)
- [Testing](#testing)
- [Docker](#docker)
- [Database](#database)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites

- **Docker & Docker Compose** (required)
- **Go 1.21+** (optional for local development)
- **Make** (optional but recommended)

### Get Started in 3 Commands

```bash
make up              # Start everything (includes monitoring!)
make logs            # View live logs
# Visit http://localhost:3000 for Grafana (admin / admin_secret_local)
```

### View All Commands

```bash
make help            # List all available commands
```

---

## Development

### Directory Structure

```
death-level-tracker/
├── cmd/bot/              # Application entry point
├── config/               # Configuration files (organized by service)
│   ├── grafana/          # Grafana dashboard + provisioning
│   ├── prometheus/       # Prometheus scrape config
│   └── sqlc.yaml         # sqlc code generation config
├── internal/
│   ├── config/           # Configuration & validation
│   ├── handlers/         # Discord command handlers
│   ├── metrics/          # Prometheus metrics definitions
│   ├── storage/          # Database layer
│   ├── tracker/          # Core tracking logic
│   └── tibiadata/        # TibiaData API client
├── sql/
│   ├── migrations/       # Database migrations (Atlas)
│   └── queries.sql       # SQL queries (sqlc)
├── Makefile              # Build automation
└── docker-compose.yml    # Service orchestration
```

### Local Development (with Go installed)

```bash
# Setup
make tidy              # Download dependencies
make fmt               # Format code
make vet               # Lint code

# Build
make build             # Build binary to bin/death-level-tracker
make clean             # Remove build artifacts

# Run quality checks
make all               # Run tidy, fmt, vet, and build
```

### Docker Development (no local Go needed)

```bash
# Start/Stop
make dev-up            # Start dev container + PostgreSQL
make dev-down          # Stop dev container
make dev-shell         # Open shell inside container

# Inside container, you can:
go run cmd/bot/main.go
go mod tidy
go fmt ./...
```

---

## Testing

### Run All Tests

```bash
# Local
make test

# Docker
make dev-test
```

### Test Specific Package

```bash
# Local
make test death-level-tracker/internal/config
make test death-level-tracker/internal/tracker
make test death-level-tracker/cmd/bot

# Docker
make dev-test death-level-tracker/internal/config
```

### Coverage Reports

```bash
# Summary
make coverage                                    # All packages
make coverage death-level-tracker/internal/tracker     # Specific package

# HTML Report
make coverage-html                               # Opens coverage.html

# Docker
make dev-coverage death-level-tracker/internal/config
```

---

## Docker

### Production Deployment

```bash
make up              # Build and start ALL services (bot, db, prometheus, grafana)
make logs            # View live logs
make down            # Stop all services
```

**All monitoring services start automatically with `make up`!**

### Development

```bash
make dev-up          # Start dev environment
make dev-down        # Stop dev environment
make dev-shell       # Access container shell
make dev-test        # Run tests in container
make dev-coverage    # Coverage in container
make dev-sqlc        # Generate code in container
```

### Containers

- `bot` - Production bot service (exposes :2112 for Prometheus)
- `postgres` - PostgreSQL database
- `migrate` - Atlas migration runner
- `prometheus` - Time-series metrics database
- `grafana` - Visualization and dashboards
- `dev` - Development environment (Go, sqlc, tools)

---

## Database

### Migrations

We use **Atlas** for schema management with manual SQL patches.

#### Create New Migration

```bash
make db-new          # Creates timestamped SQL file in sql/migrations/
```

Enter a descriptive name (e.g., `add_users_table`).

#### Apply Migrations

Migrations run automatically on service start. To apply manually:

```bash
make dev-up          # Migrations run during startup
# OR
make up              # Production startup with migrations
```

#### Update Hash (Required After Editing)

```bash
make db-hash         # Update atlas.sum integrity file
```

**⚠️ Always run `make db-hash` after editing migration files!**

#### Reset Database

```bash
make db-reset        # ⚠️ WARNING: Destroys all data!
```

### Schema

- `worlds` - Tracked Tibia worlds per Discord guild
- `players` - Player tracking with last seen timestamps
- `player_levels` - Historical level data for level-up detection

---

## Configuration

### Environment Variables

Create `.env` file for non-sensitive config:

```bash
# Optional (with defaults shown)
TRACKER_INTERVAL=5m
MIN_LEVEL_TRACK=500
WORKER_POOL_SIZE=10
DISCORD_CHANNEL_DEATH=death-tracker
DISCORD_CHANNEL_LEVEL=level-tracker
USE_TIBIACOM_FOR_LEVELS=true  # Use tibia.com for level tracking (default: true)
```

#### Data Source Configuration

**USE_TIBIACOM_FOR_LEVELS** controls which data source is used for level tracking:

- `true` (default) — Fetches online player levels from tibia.com HTML
  - Reduces TibiaData API calls significantly (1 request per world vs hundreds)
  - Still uses TibiaData for death tracking (requires detailed character info)
  - Automatically falls back to TibiaData if tibia.com fails
  
- `false` — Uses TibiaData API exclusively
  - All data (levels + deaths) comes from TibiaData API
  - More API calls but simpler data flow

**Offline players** always use TibiaData API regardless of this setting.

### Docker Secrets (Recommended for Production)

Store sensitive data in Docker secrets:

```bash
# Create secrets directory
mkdir -p secrets

# Add your Discord token
echo "your_discord_bot_token_here" > secrets/discord_token.txt

# Add Grafana password
echo "your_secure_password" > secrets/grafana_password.txt

# Verify permissions (optional)
chmod 600 secrets/*.txt
```

The bot reads secrets from `/run/secrets/` first, then falls back to environment variables:
- `secrets/discord_token.txt` → Docker secret (production)
- `secrets/grafana_password.txt` → Grafana admin password
- `DISCORD_TOKEN` env var → Fallback (development)

### Validation Rules

Configuration is validated on startup:

- **DISCORD_TOKEN**: Required, ≥50 characters
- **TRACKER_INTERVAL**: 1 minute to 24 hours
- **MIN_LEVEL_TRACK**: ≥1 (no upper limit)
- **WORKER_POOL_SIZE**: 1 to 100
- **Channel names**: 1 to 100 characters (Discord limit)
- **USE_TIBIACOM_FOR_LEVELS**: Boolean (true/false)

---

## Monitoring

### Overview

The application includes **production-grade observability** with Prometheus (metrics) and Grafana (dashboards).

### Accessing Dashboards

**Local Development:**
```bash
# After running `make up`:
# Grafana: http://localhost:3000
# Prometheus: http://localhost:9090
# Credentials: admin / (password from secrets/grafana_password.txt)
```

**Production (VPS):**
```bash
# Ports are bound to localhost for security
# Use SSH tunneling:
ssh -L 3000:localhost:3000 -L 9090:localhost:9090 user@your-vps

# Then access locally:
# http://localhost:3000 (Grafana)
# http://localhost:9090 (Prometheus)
```

### Pre-configured Dashboard

The **"Death Level Tracker - Ops View"** dashboard is auto-provisioned and includes:

#### 🚦 Executive Summary
- Service uptime
- Error rate (gauge)
- P99 latency for API calls
- Current bot status

#### 📈 Business Intelligence
- Total deaths and level-ups tracked
- Hourly event velocity (deaths/level-ups per hour)

#### 🌐 External API Performance
- TibiaData API requests per second (by endpoint)
- Throughput by HTTP status code
- Latency heatmap (visualize distribution)

#### 🔧 Runtime Internals
- Go goroutines count
- Memory usage (heap, stack)
- Garbage collection activity

### Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `death_tracker_deaths_total` | Counter | Total deaths tracked |
| `death_tracker_level_ups_total` | Counter | Total level-ups tracked |
| `tibiadata_requests_total{endpoint,status}` | Counter | API calls by endpoint/status |
| `tibiadata_request_duration_seconds{endpoint,status}` | Histogram | API latency distribution |
| `up{job="death-tracker"}` | Gauge | Service health (1=up, 0=down) |
| `go_goroutines` | Gauge | Active goroutines |
| `go_memstats_heap_alloc_bytes` | Gauge | Heap memory allocated |

### Querying Metrics (PromQL Examples)

```promql
# Deaths per hour
increase(death_tracker_deaths_total[1h])

# Average API latency (5m window)
sum(rate(tibiadata_request_duration_seconds_sum[5m])) 
  / sum(rate(tibiadata_request_duration_seconds_count[5m]))

# Error rate (5xx responses)
sum(rate(tibiadata_requests_total{status=~"5.."}[5m]))
  / sum(rate(tibiadata_requests_total[5m]))

# 99th percentile latency
histogram_quantile(0.99, 
  sum(rate(tibiadata_request_duration_seconds_bucket[5m])) by (le))
```

### Customizing Dashboards

The dashboard JSON is located at:
```
config/grafana/dashboards/dashboard.json
```

You can:
1. Edit the JSON directly
2. Modify in Grafana UI and export
3. Restart Grafana to reload: `docker-compose restart grafana`

---

## Code Generation

### SQLC

Generate Go code from SQL queries:

```bash
make sqlc            # Local (uses config/sqlc.yaml)
make dev-sqlc        # Docker
```

Generates type-safe database access code from `sql/queries.sql`.

**Note:** `sqlc` now uses `-f config/sqlc.yaml` to find its config.

---

## Workflows

### Adding a New Feature

1. Create feature branch
2. Write tests first (TDD)
3. Implement feature
4. Run tests: `make test`
5. Run linter: `make vet`
6. Format code: `make fmt`
7. Build: `make build`
8. Create PR

### Making Database Changes

1. Create migration: `make db-new`
2. Edit the generated SQL file
3. Update hash: `make db-hash`
4. Test locally: `make dev-up`
5. Commit migration files + atlas.sum

### Running Locally for Development

```bash
# Terminal 1: Start all services
make up

# Terminal 2: View logs
make logs

# Terminal 3 (optional): Access dev shell
make dev-shell
```

---

## Troubleshooting

### Common Issues

#### "Database connection failed"

```bash
make dev-up          # Ensure PostgreSQL is running
docker-compose ps    # Check service status
```

#### "Migration failed"

```bash
make db-reset        # Reset database (loses data!)
make dev-up          # Apply migrations fresh
```

#### "Tests failing"

```bash
make clean           # Remove build artifacts
go clean -testcache  # Clear test cache
make test            # Run tests again
```

#### "Module not found"

```bash
make tidy            # Download dependencies
```

#### "Port already in use"

```bash
make down            # Stop all services
docker-compose down  # Force stop
```

#### "Grafana shows no data"

```bash
# Check if Prometheus is scraping
curl http://localhost:9090/api/v1/targets

# Verify bot metrics endpoint
curl http://localhost:2112/metrics

# Restart Grafana
docker-compose restart grafana
```

### Logs

```bash
# Production
make logs

# Specific service
docker-compose logs bot
docker-compose logs postgres
docker-compose logs prometheus
docker-compose logs grafana
```

---

## Best Practices

### Code Style

- Run `make fmt` before committing
- Run `make vet` to catch issues
- Follow Go idioms and conventions
- Keep functions small and focused
- Use interfaces for testability

### Testing

- Aim for >90% coverage on business logic
- Write table-driven tests
- Use mocks for external dependencies
- Test error paths, not just happy paths

### Git Workflow

```bash
git checkout -b feature/my-feature
make all                # Verify everything works
git add .
git commit -m "feat: add my feature"
git push origin feature/my-feature
```

### Before Pushing

```bash
make all               # Runs: tidy, fmt, vet, build
make test              # Run all tests
make clean             # Clean up
```

---

## Quick Reference

### Most Used Commands

```bash
make help              # Show all commands
make up                # Start production (all services)
make dev-up            # Start development
make dev-test          # Run tests
make dev-shell         # Access container
make test <package>    # Test specific package
make coverage-html     # Coverage report
make build             # Build binary
make db-new            # New migration
make logs              # View logs
```

### Project Resources

- **Makefile**: Run `make help` for all commands
- **Migrations**: `sql/migrations/`
- **Queries**: `sql/queries.sql`
- **Config**: `.env` file + `config/` directory
- **Monitoring**: Grafana on :3000, Prometheus on :9090
- **Logs**: `make logs`

---

## Production Deployment

### GitHub Actions Setup

The project includes a deployment workflow (`.github/workflows/deploy.yml`) that requires:

**GitHub Secrets:**
- `VPS_HOST` - Your server IP/hostname
- `VPS_USER` - SSH username
- `VPS_SSH_KEY` - Private SSH key
- `VPS_PORT` - SSH port (usually 22)
- `GRAFANA_PASSWORD` - Grafana admin password

The workflow automatically:
1. Builds the Docker image
2. Waits for manual approval (production environment)
3. Deploys via SSH
4. Injects secrets into `secrets/` directory on VPS
5. Starts all services with `docker compose up`

### Manual VPS Deployment

```bash
# On your VPS
cd ~/death-level-tracker
git pull origin master

# Create secrets
mkdir -p secrets
echo "your_discord_token" > secrets/discord_token.txt
echo "your_grafana_password" > secrets/grafana_password.txt

# Deploy
docker compose down
docker compose up -d --build

# Verify
docker compose ps
docker compose logs --tail=50 bot
```

---

## Getting Help

1. Check this guide
2. Run `make help`
3. Check Docker logs: `make logs`
4. Review error messages carefully
5. Check Grafana for metrics insights
6. Ask in team chat

---

**Happy coding! 🚀**
