# Death Level Tracker - Developer Guide

> **Tibia Death & Level Tracker Discord Bot** - A Go application that monitors Tibia game worlds for player deaths and level ups, posting notifications to Discord channels.

## Table of Contents

- [Quick Start](#quick-start)
- [Development](#development)
- [Testing](#testing)
- [Docker](#docker)
- [Database](#database)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites

- **Docker & Docker Compose** (required)
- **Go 1.21+** (optional for local development)
- **Make** (optional but recommended)

### Get Started in 3 Commands

```bash
make dev-up          # Start everything (database + dev environment)
make dev-test        # Run tests to verify setup
make dev-shell       # Open shell to explore
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
├── internal/
│   ├── config/           # Configuration & validation
│   ├── handlers/         # Discord command handlers
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
make up              # Build and start production services
make logs            # View live logs
make down            # Stop all services
```

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

- `dev` - Development environment (Go, sqlc, tools)
- `bot` - Production bot service
- `postgres` - PostgreSQL database
- `migrate` - Atlas migration runner

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
DISCORD_CHANNEL_DEATH=death-level-tracker
DISCORD_CHANNEL_LEVEL=level-tracker
```

### Docker Secrets (Recommended for Production)

Store sensitive data in Docker secrets:

```bash
# Create secrets directory
mkdir -p secrets

# Add your Discord token
echo "your_discord_bot_token_here" > secrets/discord_token.txt

# Verify permissions (optional)
chmod 600 secrets/discord_token.txt
```

The bot reads secrets from `/run/secrets/` first, then falls back to environment variables:
- `secrets/discord_token.txt` → Docker secret (production)
- `DISCORD_TOKEN` env var → Fallback (development)

### Validation Rules

Configuration is validated on startup:

- **DISCORD_TOKEN**: Required, ≥50 characters
- **TRACKER_INTERVAL**: 1 minute to 24 hours
- **MIN_LEVEL_TRACK**: ≥1 (no upper limit)
- **WORKER_POOL_SIZE**: 1 to 100
- **Channel names**: 1 to 100 characters (Discord limit)

---

## Code Generation

### SQLC

Generate Go code from SQL queries:

```bash
make sqlc            # Local
make dev-sqlc        # Docker
```

Generates type-safe database access code from `sql/queries.sql`.

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
# Terminal 1: Start database
make dev-up

# Terminal 2: Run bot
make dev-shell
go run cmd/bot/main.go
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

### Logs

```bash
# Production
make logs

# Docker Compose services
docker-compose logs postgres
docker-compose logs bot
docker-compose logs migrate
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
make dev-up            # Start development
make dev-test          # Run tests
make dev-shell         # Access container
make test <package>    # Test specific package
make coverage-html     # Coverage report
make build             # Build binary
make db-new            # New migration
```

### Project Resources

- **Makefile**: Run `make help` for all commands
- **Migrations**: `sql/migrations/`
- **Queries**: `sql/queries.sql`
- **Config**: `.env` file
- **Logs**: `make logs`

---

## Getting Help

1. Check this guide
2. Run `make help`
3. Check Docker logs: `make logs`
4. Review error messages carefully
5. Ask in team chat

---

**Happy coding! 🚀**
