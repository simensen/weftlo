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

```bash
# 1. Install
go install github.com/simensen/weftlo/cmd/weftlo@latest

# 2. Initialize (creates ~/.config/weftlo/ with a starter profile)
weftlo init

# 3. Install the starter profile into a project
cd ~/projects/my-app
weftlo install

# 4. See what got rendered
cat .claude/CLAUDE.md
```

The `cat` prints a rendered `CLAUDE.md` with the profile's variables substituted in.
Edit the template at `~/.config/weftlo/profiles/default/default/content/CLAUDE.md.tmpl`
and run `weftlo update` in your project to sync changes.

For inheritance, variables, multi-profile composition, and routing,
see the [Product Vision](docs/product/vision.md) and the
[Configuration Reference](docs/reference/configuration.md).

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
