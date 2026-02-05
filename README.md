# 🔨 Forge Platform

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen)
![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-blue)

**A Unified Engineering Platform in a Single Binary**

_CLI • TUI • TSDB • WebAssembly • AI_

[Installation](#-installation) •
[Quick Start](#-quick-start) •
[Documentation](#-documentation) •
[Roadmap](ROADMAP.md) •
[Contributing](#-contributing)

</div>

---

## 🎯 Overview

**Forge Platform** is a next-generation engineering tool that consolidates five critical capabilities into a single, portable binary:

| Component | Description                     | Technology             |
| --------- | ------------------------------- | ---------------------- |
| **CLI**   | Powerful command-line interface | Cobra + Viper          |
| **TUI**   | Rich terminal dashboard         | Bubble Tea + Lip Gloss |
| **TSDB**  | Embedded time-series database   | SQLite (WAL + UUIDv7)  |
| **Wasm**  | Secure plugin extensibility     | wazero (Zero CGO)      |
| **AI**    | Local LLM integration           | Ollama + LangChainGo   |

### Why Forge?

- **🎒 Single Binary**: No dependencies, no containers, no infrastructure
- **🔒 Local-First**: Your data stays on your machine
- **⚡ High Performance**: SQLite TSDB with 100K+ writes/sec
- **🧩 Extensible**: WebAssembly plugins with sandboxed execution
- **🤖 AI-Powered**: Local LLMs for intelligent automation
- **🏗️ Clean Architecture**: Hexagonal design for maintainability

## 📦 Installation

### From Source (Recommended)

```bash
# Clone the repository
git clone https://github.com/forge-platform/forge.git
cd forge

# Build
make build

# Install to PATH
make install
```

### Pre-built Binaries

```bash
# Linux (amd64)
curl -L https://github.com/forge-platform/forge/releases/latest/download/forge-linux-amd64 -o forge
chmod +x forge && sudo mv forge /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/forge-platform/forge/releases/latest/download/forge-darwin-arm64 -o forge
chmod +x forge && sudo mv forge /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/forge-platform/forge/releases/latest/download/forge-windows-amd64.exe" -OutFile "forge.exe"
```

### Docker

```bash
docker pull ghcr.io/forge-platform/forge:latest
docker run -it --rm -v ~/.forge:/home/forge/.forge forge
```

## 🚀 Quick Start

```bash
# Initialize Forge (creates ~/.forge directory)
forge init

# Start the daemon (background service)
forge start

# Check status
forge status

# Open the Terminal UI
forge ui

# Record a metric
forge metric record cpu_usage 75.5 --tags host=server1

# Create a task
forge task create --type maintenance --payload '{"action": "cleanup"}'

# Chat with AI (requires Ollama)
forge ai chat "Analyze the last hour of metrics"

# Stop the daemon
forge stop
```

## 📖 Documentation

| Document                             | Description                                 |
| ------------------------------------ | ------------------------------------------- |
| [Architecture](docs/ARCHITECTURE.md) | Hexagonal architecture and design decisions |
| [Development](docs/DEVELOPMENT.md)   | Guide for contributors                      |
| [Plugins](docs/PLUGINS.md)           | WebAssembly plugin development              |
| [Roadmap](ROADMAP.md)                | Project roadmap and milestones              |

## 🏛️ Architecture

Forge follows **Hexagonal Architecture** (Ports and Adapters) to maintain clean separation:

```
┌─────────────────────────────────────────────────────────────────┐
│                        ADAPTERS (Drivers)                       │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────────┐ │
│  │   CLI   │  │   TUI   │  │  HTTP   │  │   gRPC (Daemon)     │ │
│  │ (Cobra) │  │(Bubble) │  │  (API)  │  │  (Unix Socket)      │ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └──────────┬──────────┘ │
└───────┼────────────┼────────────┼──────────────────┼────────────┘
        │            │            │                  │
        ▼            ▼            ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                          PORTS (Interfaces)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ TaskRepository│  │MetricRepository│ │   WasmRuntime       │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
        │            │            │                  │
        ▼            ▼            ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                         DOMAIN (Core)                           │
│  ┌────────┐  ┌────────┐  ┌────────┐  ┌──────────┐  ┌─────────┐  │
│  │  Task  │  │ Metric │  │ Plugin │  │Conversation│ │Workflow │  │
│  └────────┘  └────────┘  └────────┘  └──────────┘  └─────────┘  │
└─────────────────────────────────────────────────────────────────┘
        │            │            │                  │
        ▼            ▼            ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                       ADAPTERS (Driven)                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────────┐ │
│  │ SQLite  │  │ wazero  │  │ Ollama  │  │    File System      │ │
│  │ (TSDB)  │  │ (Wasm)  │  │  (AI)   │  │    (Plugins)        │ │
│  └─────────┘  └─────────┘  └─────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## 🔧 Configuration

Forge uses a YAML configuration file at `~/.forge/config.yaml`:

```yaml
# Forge Platform Configuration
daemon:
  socket: ~/.forge/forge.sock
  pid_file: ~/.forge/forge.pid

tsdb:
  path: ~/.forge/data/forge.db
  wal_mode: true
  cache_size: 100MB
  retention:
    raw: 7d
    medium: 30d
    long: 365d

ai:
  provider: ollama
  endpoint: http://localhost:11434
  model: llama3.2

plugins:
  directory: ~/.forge/plugins
  auto_load: true
```

## 🧩 Plugin System

Create powerful extensions with WebAssembly:

```go
// plugin.go - Compile with TinyGo
package main

import "github.com/forge-platform/forge/pkg/sdk"

func main() {
    sdk.Info("Plugin initialized!")
    sdk.RecordMetric("custom_metric", 42.0)
}

//export on_tick
func onTick() {
    // Called periodically by Forge
}
```

Build and install:

```bash
tinygo build -o my-plugin.wasm -target wasi plugin.go
forge plugin install my-plugin.wasm
```

## 🤖 AI Integration

Forge integrates with local LLMs via Ollama:

```bash
# Install Ollama (if not installed)
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull llama3.2

# Use AI in Forge
forge ai chat "What caused the CPU spike at 10am?"
forge ai analyze --metric cpu_usage --window 1h
```

## 📊 TSDB Features

- **UUIDv7 Primary Keys**: Monotonic, time-ordered for optimal B-Tree performance
- **WAL Mode**: Concurrent reads/writes without blocking
- **Automatic Downsampling**: Raw → 1min → 1hour aggregations
- **100K+ writes/sec**: Optimized for high-throughput ingestion

## 🛠️ Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Lint code
make lint

# Format code
make fmt

# Build for all platforms
make build-all
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md).

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [wazero](https://github.com/tetratelabs/wazero) - WebAssembly runtime
- [LangChainGo](https://github.com/tmc/langchaingo) - LLM orchestration
- [SQLite](https://sqlite.org/) - Embedded database

---

<div align="center">
  <sub>Built with ❤️ by the Forge Team</sub>
</div>
