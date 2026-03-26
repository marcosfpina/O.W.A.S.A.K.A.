# O.W.A.S.A.K.A. SIEM

> **Open Watchful Air-gapped Security Analysis Kit & Architecture**

A zero-trust, air-gapped SIEM platform built for surgical precision monitoring with enterprise-grade security - running locally on dedicated hardware.

---

## Philosophy

> "A SIEM should be like a butler: invisible until needed, impeccably informed when called upon, and never presumptuous about what matters."

**Core Principles:**
- **Isolation-First Design**: Air-gapped by architecture, not configuration
- **Defense in Depth**: Layered security at every level
- **Elegance Over Complexity**: Clean UX, minimal footprint, maximum insight
- **Signal over Noise**: Optimize for what matters

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    O.W.A.S.A.K.A. SIEM                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐    ┌──────────────────┐                 │
│  │  Modern Web UI   │───▶│  WebSocket API   │                 │
│  │  (Svelte + TS)   │    │   (Real-time)    │                 │
│  └──────────────────┘    └──────────────────┘                 │
│                                   │                             │
│  ┌────────────────────────────────▼─────────────────────────┐ │
│  │           Golang Core Engine                             │ │
│  ├──────────────────────────────────────────────────────────┤ │
│  │ Network Intelligence │ Discovery Engine │ Analytics      │ │
│  │  • DNS Resolver      │  • Physical      │  • Correlation │ │
│  │  • Proxy/DPI         │  • Virtual       │  • ML Anomaly  │ │
│  │  • Topology Map      │  • Attack Surface│  • Alerting    │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                   │                             │
│  ┌────────────────────────────────▼─────────────────────────┐ │
│  │         Secure Storage Layer (NAS Integration)           │ │
│  │  • Encrypted at rest (AES-256-GCM)                       │ │
│  │  • Immutable audit logs                                  │ │
│  │  • Integrity verification (Merkle trees)                 │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
        │                          │                    │
        ▼                          ▼                    ▼
   [Physical]              [Virtual/Containers]    [Network]
   Devices                 Docker/VMs              Services
```

---

## Features

### Network Intelligence
- **Custom DNS Resolver** with query logging and anomaly detection
- **Transparent Proxy** with mTLS inspection and protocol detection
- **Network Topology Mapper** with active/passive discovery
- **Deep Packet Inspection** for traffic analysis

### Asset Discovery
- **Multi-layer Discovery**: Physical, virtual, containerized assets
- **Attack Surface Mapping**: ALL ports (0-65535), including dormant services
- **Continuous Reconciliation**: Detect changes and drift in real-time
- **Ghost Port Detection**: Find development/debug endpoints

### Security
- **Air-gapped Architecture**: No external network exposure by design
- **Self-hosted Firefox** with enforced security policies
- **Encrypted Storage**: AES-256-GCM with Argon2id key derivation
- **Immutable Audit Logs**: Tamper-proof event recording

### User Experience
- **Modern, Clean Interface**: Dark mode by default
- **Real-time Updates**: WebSocket-powered live data
- **Customizable Dashboards**: Widget-based layout
- **Low Resource Usage**: <500MB memory footprint (idle)

---

## Project Structure

```
O.W.A.S.A.K.A./
├── cmd/
│   └── oswaka/              # Application entry point
├── internal/                # Private application logic
│   ├── network/            # Network intelligence layer
│   │   ├── dns/           # DNS resolver & logging
│   │   ├── proxy/         # Transparent proxy & DPI
│   │   ├── discovery/     # Network scanning
│   │   └── topology/      # Network graph
│   ├── discovery/         # Asset discovery engine
│   │   ├── physical/      # Physical device enumeration
│   │   ├── virtual/       # VM/container scanning
│   │   ├── attack_surface/# Attack surface mapper
│   │   └── reconciler/    # Change detection
│   ├── browser/           # Firefox integration
│   │   ├── firefox/       # Browser launcher
│   │   ├── policies/      # Security policy enforcer
│   │   └── automation/    # WebDriver integration
│   ├── storage/           # Data persistence
│   │   ├── nas/          # NAS connector
│   │   ├── crypto/       # Encryption/decryption
│   │   └── integrity/    # Verification & checksums
│   └── analytics/         # Intelligence engine
│       ├── stream/       # Event processing
│       ├── correlation/  # Rule engine
│       └── ml/          # Anomaly detection
├── pkg/                   # Public libraries
│   ├── config/           # Configuration management
│   ├── logging/          # Structured logging
│   └── metrics/          # Prometheus metrics
├── web/                   # Frontend application
│   ├── src/
│   │   ├── components/   # Svelte components
│   │   ├── stores/       # State management
│   │   └── lib/         # Utilities
│   └── public/           # Static assets
├── configs/               # Configuration files
│   ├── examples/         # Example configurations
│   └── policies/         # Security policies
└── docs/                  # Documentation
    ├── architecture/     # Design docs
    ├── api/             # API documentation
    └── deployment/      # Deployment guides
```

---

## Quick Start

### Prerequisites

**Option 1: Nix Flakes (Recommended for Development)**
- **Nix with Flakes** - Reproducible development environment
- All dependencies managed automatically
- See [Nix Development Guide](docs/development/NIX_GUIDE.md)

**Option 2: Manual Installation**
- **Go 1.22+** (tested on 1.24.7)
- **Node.js 18+** (for web UI)
- **Dedicated hardware** (no shared environments)
- **NAS cluster** (for persistent storage)
- **Firefox ESR** (for browser integration)

### Installation with Nix (Recommended)

```bash
# Clone the repository
git clone https://github.com/marcosfpina/O.W.A.S.A.K.A.git
cd O.W.A.S.A.K.A

# Enter development environment (all dependencies auto-installed)
nix develop

# You'll see the O.W.A.S.A.K.A. welcome banner!
# Now you have access to all tools: Go, Node.js, network tools, etc.

# Build the project
oswaka-dev build

# Or use make directly
make build

# Run the SIEM
oswaka-dev run

# Hot reload development mode
oswaka-dev watch

# Show all available commands
oswaka-dev help
```

**What's included in Nix environment:**
- Go 1.22+, Node.js 20, Firefox ESR
- Network tools: nmap, tcpdump, tshark, dig
- Go tools: gopls, delve, golangci-lint, air
- Development utilities: jq, ripgrep, bat, htop
- Custom scripts and aliases

See the [complete Nix guide](docs/development/NIX_GUIDE.md) for advanced usage.

### Installation (Manual)

```bash
# Clone the repository
git clone https://github.com/marcosfpina/O.W.A.S.A.K.A.git
cd O.W.A.S.A.K.A

# Build the project
make build

# Run tests
make test

# Start the SIEM
./bin/oswaka --config configs/examples/default.yaml
```

### Development

```bash
# Install dependencies
make deps

# Run in development mode
make dev

# Run linters
make lint

# Generate documentation
make docs
```

---

## Configuration

Example configuration (`configs/examples/default.yaml`):

```yaml
# Coming soon - PHASE 0 in progress
```

---

## Development Status

### PHASE 0: Foundation & Environment Setup ✅ (Completed)
- [x] Repository structure
- [x] Go module initialization
- [x] Build system setup and Nix derivations
- [x] Configuration templates and validation
- [x] Architecture documentation

### PHASE 1: Network Intelligence Layer 🚧
- [x] High-Performance DNS Resolver
- [ ] Transparent Proxy Engine
- [x] Network Topology Mapper
- [x] BoltDB Event Persistence

### PHASE 2: Asset Discovery 🚧
- [ ] Multi-layer discovery (Physical)
- [x] Virtual/Container discovery (Zero-Dependency Docker Scanner)
- [ ] Attack surface mapping
- [ ] Continuous reconciliation

### PHASE 3: Browser Integration 🚧
- [x] Hardened Firefox configuration launcher
- [ ] WebDriver remote automation
- [ ] Forensic logging

### PHASE 4: Modern Frontend 🚀
- [x] SvelteKit dashboard (Crimson Red / Glassmorphism)
- [x] Real-time WebSocket pipeline integration
- [x] D3.js Network Topology Visualization
- [x] Threat Alert HUD

### PHASE 5: Analytics Engine 🚧
- [x] In-memory Event Pipeline (Pub/Sub)
- [x] Real-time Correlation rules evaluation
- [ ] ML-based anomaly detection

### PHASE 6: SPECTRE Fleet SDK Integration 🚀
- [x] Rust Proxy NATS EventBus bridge
- [x] JWT Authentication & Rate Limiting (Axum)
- [ ] Distributed OpenTelemetry

---

## Performance Targets

- **UI Response Time**: <100ms (p95)
- **Memory Footprint**: <500MB (idle)
- **Network Overhead**: <5% of bandwidth
- **Discovery Scan**: <60s for 1000 assets

---

## Security Model

### Threat Assumptions
- Physical access is controlled
- NAS is in trusted network segment
- Operator is non-malicious (insider threat out of scope)

### Protections
- Memory-safe language (Golang)
- Input validation everywhere
- No external dependencies at runtime
- Reproducible builds
- Encrypted data at rest
- Immutable audit logs

---

## Contributing

This is a personal security infrastructure project. If you're interested in similar work:

1. Fork the repository
2. Study the architecture in `/docs/architecture`
3. Build your own variant
4. Share learnings (not code) back

---

## License

**Proprietary** - Personal security infrastructure
Not licensed for commercial use or distribution.

---

## Acknowledgments

Built with inspiration from:
- The Art of Monitoring (James Turnbull)
- Security Engineering (Ross Anderson)
- Designing Data-Intensive Applications (Martin Kleppmann)

---

## Contact

Project maintained by: Marcos Pina
Repository: https://github.com/marcosfpina/O.W.A.S.A.K.A

---

**Status**: 🚀 Voo de Cruzeiro - Modulos Core Integrados

Last Updated: 2026-03-26
