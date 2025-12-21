# Weftlo

**Weftlo** is a CLI tool for managing configuration profiles for AI coding assistants. It supports profile inheritance, template rendering, and configuration management across multiple projects.

> **⚠️ Early Development**: This project is in active early development. APIs, configuration formats, and features may change without notice. Use at your own risk.

## Installation

Install weftlo using Go:

```bash
go install github.com/simensen/weftlo/cmd/weftlo@latest
```

**Requirements:** Go 1.24+

Verify the installation:

```bash
weftlo version
```

## Quick Start

### 1. Initialize Configuration

```bash
weftlo init
```

This creates the configuration directory (`~/.config/weftlo/` or `~/.weftlo/`) with a default profile.

### 2. Install a Profile to Your Project

```bash
cd ~/projects/my-app
weftlo install
```

This renders templates from your profile into the project and creates tracking files (`.weftlo.yaml` and `.weftlo.manifest.json`).

### 3. Update After Template Changes

```bash
weftlo update
```

Syncs changes from your profile templates, preserving any local modifications.

## Documentation

Detailed documentation is available in the [`docs/`](docs/) directory:

- **[Product Vision](docs/product/vision.md)** — What Weftlo is and why it exists
- **[Architecture Overview](docs/architecture/overview.md)** — Technical architecture and design
- **[CLI Reference](docs/reference/cli.md)** — Complete command reference
- **[Configuration Reference](docs/reference/configuration.md)** — Configuration file formats
- **[Template Reference](docs/reference/templates.md)** — Template syntax and functions

## Development

### Prerequisites

- Go 1.24+
- Make (optional, for convenience commands)

### Setup

```bash
git clone https://github.com/simensen/weftlo.git
cd weftlo
go build ./...
```

### Running Tests

```bash
# All tests
make test

# Or directly with go
go test ./...

# With coverage
go test -cover ./...
```

### Code Quality

```bash
make fmt      # Format code
make vet      # Run go vet
make lint     # Run linter
```

### Building

```bash
# Build binary
go build -o weftlo ./cmd/weftlo

# Build with version
go build -ldflags "-X main.version=1.0.0" -o weftlo ./cmd/weftlo
```

## License

MIT License - see [LICENSE](LICENSE) for details.
