# Death Level Tracker

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Discord bot that monitors Tibia game worlds for player deaths and level-ups, posting real-time notifications to designated channels.

## Features

- 🎮 **Real-time Tracking** — Monitors online players across configured Tibia worlds
- 💀 **Death Notifications** — Detects and posts player deaths with killer information
- 📈 **Level-up Alerts** — Tracks and announces level changes for high-level players
- ⚡ **Concurrent Processing** — Worker pool for efficient API fetching
- 🔧 **Per-Guild Configuration** — Each Discord server tracks its own worlds
- 📊 **Production Monitoring** — Prometheus metrics + Grafana dashboards
- 🐳 **Docker Ready** — Full containerized deployment with PostgreSQL

## Quick Start

```bash
# 1. Clone and configure
git clone https://github.com/yourusername/death-level-tracker.git
cd death-level-tracker

# 2. Set up secrets (required)
mkdir -p secrets
echo "your_discord_bot_token" > secrets/discord_token.txt
echo "admin_secret_local" > secrets/grafana_password.txt

# 3. Start all services (includes monitoring)
make up

# 4. Access dashboards
# Grafana: http://localhost:3000 (admin / admin_secret_local)
# Prometheus: http://localhost:9090
```

**For detailed development instructions, see [CHEATSHEET.md](CHEATSHEET.md).**

### Data Sources

The tracker uses two complementary data sources for optimal performance:

- **Tibia.com HTML** (default) — Fetches all online player levels in a single request per world
- **TibiaData API** — Used for death tracking (requires detailed character information)
- **Offline players** — Always use TibiaData API for both levels and deaths

This hybrid approach minimizes API calls while maintaining full functionality.

### Key Components

| Package | Purpose |
|---------|---------|
| `cmd/bot` | Application entry point and lifecycle |
| `internal/tracker` | Core tracking service and worker pool |
| `internal/handlers` | Discord slash command handlers |
| `internal/storage` | Database layer (sqlc generated) |
| `internal/tibiadata` | TibiaData API client |
| `internal/config` | Configuration loading and validation |

## Discord Commands

| Command | Description |
|---------|-------------|
| `/track-world <name>` | Set the Tibia world to track for this server |
| `/stop-tracking` | Stop tracking kills |

## Configuration

### Docker Secrets (Required)

```bash
mkdir -p secrets
echo "your_discord_bot_token" > secrets/discord_token.txt
```

### Environment Variables (Optional)

```bash
TRACKER_INTERVAL=5m           # Polling interval (1m-24h)
MIN_LEVEL_TRACK=500           # Minimum level to track
WORKER_POOL_SIZE=10           # Concurrent workers (1-100)
DISCORD_CHANNEL_DEATH=death-tracker
DISCORD_CHANNEL_LEVEL=level-tracker
USE_TIBIACOM_FOR_LEVELS=true  # Use tibia.com HTML for level tracking (default: true)
```

#### Data Source Selection

- `USE_TIBIACOM_FOR_LEVELS=true` (default) — Fetches online player levels from tibia.com HTML, reducing TibiaData API calls
- `USE_TIBIACOM_FOR_LEVELS=false` — Uses TibiaData API exclusively for both levels and deaths

See [CHEATSHEET.md](CHEATSHEET.md#configuration) for validation rules and details.

## Monitoring & Observability

The application includes production-grade monitoring with Prometheus and Grafana:

### Metrics Exposed

- **Business Metrics**
  - `death_tracker_deaths_total` — Total player deaths tracked
  - `death_tracker_level_ups_total` — Total level-ups tracked
  
- **API Health**
  - `tibiadata_requests_total{endpoint, status}` — API call count by endpoint/status
  - `tibiadata_request_duration_seconds{endpoint, status}` — Latency histogram

- **Runtime Metrics**
  - Standard Go runtime metrics (heap, goroutines, GC)

### Accessing Dashboards

**Local Development:**
```bash
# Grafana: http://localhost:3000
# Prometheus: http://localhost:9090
# Credentials: admin / admin_secret_local
```

**Production (VPS):**
```bash
# Use SSH tunneling (ports bound to localhost for security)
ssh -L 3000:localhost:3000 -L 9090:localhost:9090 user@your-vps
```

The Grafana dashboard ("Death Level Tracker - Ops View") is auto-provisioned with:
- Executive Summary (SLAs, uptime, P99 latency)
- Business Intelligence (deaths/level-ups trends)
- External API Performance (TibiaData latency heatmaps)
- Runtime Internals (Go heap, goroutines, GC)


## Development

```bash
make help              # Show all available commands
make dev-up            # Start development environment
make test              # Run all tests
make coverage-html     # Generate coverage report
```

**Full development guide: [CHEATSHEET.md](CHEATSHEET.md)**

## Tech Stack

- **Language:** Go 1.25+
- **Database:** PostgreSQL 15
- **Migrations:** Atlas
- **Discord:** discordgo
- **Code Gen:** sqlc
- **Monitoring:** Prometheus + Grafana
- **Containers:** Docker & Docker Compose

## Project Structure

```
death-level-tracker/
├── cmd/bot/           # Application entry point
├── config/            # Configuration files
│   ├── grafana/       # Dashboard definitions + provisioning
│   ├── prometheus/    # Prometheus config
│   └── sqlc.yaml      # sqlc configuration
├── internal/
│   ├── config/        # Configuration & validation
│   ├── handlers/      # Discord command handlers
│   ├── metrics/       # Prometheus metrics definitions
│   ├── storage/       # Database layer
│   ├── tracker/       # Core tracking logic
│   └── tibiadata/     # TibiaData API client
├── sql/
│   ├── migrations/    # Atlas migrations
│   └── queries.sql    # sqlc queries
├── Makefile           # Build automation
├── docker-compose.yml # Service orchestration
└── CHEATSHEET.md      # Developer guide
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`make test`)
4. Commit changes (`git commit -m 'feat: add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [TibiaData API](https://tibiadata.com/) — Game data provider
- [discordgo](https://github.com/bwmarrin/discordgo) — Discord API wrapper
- [Atlas](https://atlasgo.io/) — Database migrations
- [sqlc](https://sqlc.dev/) — Type-safe SQL

---

**Made with ❤️ for the Tibia community**
