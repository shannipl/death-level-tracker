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
- 🐳 **Docker Ready** — Full containerized deployment with PostgreSQL

## Quick Start

```bash
# 1. Clone and configure
git clone https://github.com/yourusername/death-level-tracker.git
cd death-level-tracker

# 2. Set up Discord token (required)
mkdir -p secrets
echo "your_discord_bot_token" > secrets/discord_token.txt

# 3. Start services
make dev-up

# 4. Verify
make dev-test
```

**For detailed development instructions, see [CHEATSHEET.md](CHEATSHEET.md).**

## Architecture

```
┌─────────────────┐     ┌───────────────────────┐     ┌─────────────────┐
│  Discord API    │◄────│   Death Level Tracker │────►│  TibiaData API  │
│  (discordgo)    │     │     (Go 1.25+)        │     │  (REST client)  │
└─────────────────┘     └────────┬──────────────┘     └─────────────────┘
                                 │
                        ┌────────▼─────────┐
                        │   PostgreSQL     │
                        │  (player data)   │
                        └──────────────────┘
```

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
DISCORD_CHANNEL_DEATH=death-level-tracker
DISCORD_CHANNEL_LEVEL=level-tracker
```

See [CHEATSHEET.md](CHEATSHEET.md#configuration) for validation rules and details.


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
- **Containers:** Docker & Docker Compose

## Project Structure

```
death-level-tracker/
├── cmd/bot/           # Application entry point
├── internal/
│   ├── config/        # Configuration & validation
│   ├── handlers/      # Discord command handlers
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
