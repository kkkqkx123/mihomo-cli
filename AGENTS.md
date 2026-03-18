# Mihomo CLI - Project Context Documentation

## Project Overview

Mihomo CLI is a non-interactive management tool for the Mihomo proxy core. This tool provides comprehensive management capabilities over the Mihomo RESTful API via a command-line interface (CLI), delivering a stateless and scriptable proxy management solution.

### Project Goals

- Provide a pure command-line interface for Mihomo management without requiring a GUI.
- Support automated scripting and batch operations.
- Decouple from the Mihomo core process to enhance stability.

### Core Features

- **Stateless Design**: The CLI does not persist runtime state; all state queries are performed in real-time against the API.
- **Configuration Persistence**: API address and Secret are saved to a local configuration file to avoid repeated input.
- **Query vs. Mutation Separation**: Distinct operations for querying (`get`, `list`, `show`) and modifying (`set`, `switch`, `update`).
- **Controllable Output Format**: Supports both Table and JSON output formats for human readability and script parsing.

---

## Technology Stack

### Programming Language

- **Go 1.26.1**: Primary development language.

### Core Dependencies

- **github.com/spf13/cobra v1.10.2**: CLI framework for building the command structure.
- **github.com/spf13/viper v1.21.0**: Configuration management supporting config files and environment variables.
- **github.com/fatih/color v1.18.0**: Colored terminal output.
- **github.com/olekukonko/tablewriter v0.0.5**: Table formatting output.

### Project Architecture

- Modular design following standard Go project structures.
- Command tree architecture based on Cobra.
- Layered design: `cmd` (Command Layer) → `internal` (Business Logic Layer) → `pkg` (Common Types Layer).

---

## Project Structure

```text
mihomo-go/
├── cmd/                   # Command definitions entry point
│   ├── root.go            # Root command (global flags and initialization)
│   ├── mode.go            # Mode management (mode get/set)
│   ├── proxy.go           # Proxy management (list/switch/test/auto/unfix)
│   └── config.go          # CLI configuration management (init/show/set)
├── internal/              # Internal business logic implementation
│   ├── api/               # Mihomo RESTful API client encapsulation
│   │   ├── client.go      # Main API client implementation
│   │   ├── http.go        # HTTP request handling
│   │   ├── mode.go        # Mode-related API wrappers
│   │   ├── proxy.go       # Proxy-related API wrappers
│   │   └── errors.go      # API error definitions
│   ├── config/            # CLI tool configuration management
│   │   ├── config.go      # Configuration structure definitions
│   │   └── loader.go      # Configuration file loader
│   ├── proxy/             # Proxy business logic
│   │   ├── formatter.go   # Output formatting
│   │   ├── selector.go    # Node auto-selection logic
│   │   └── tester.go      # Latency testing logic
│   ├── output/            # Output formatting
│   ├── service/           # os service integration
│   ├── monitor/           # Monitoring functionality (Planned)
│   └── sysproxy/          # System proxy management
├── pkg/types/             # Common type definitions(include error type)
├── mihomo-1.19.21/        # Mihomo core reference implementation
├── main.go                # Program entry point
├── go.mod                 # Go module definition
└── README.md              # Project description
```

---

## Build & Run

### Usage

```bash
# Initialize configuration
.\mihomo-cli.exe config init

# Query current mode
.\mihomo-cli.exe mode get

# List proxies
.\mihomo-cli.exe proxy list

# Switch proxy node
.\mihomo-cli.exe proxy switch Proxy NodeName

# Test latency
.\mihomo-cli.exe proxy test Proxy

# Auto-select fastest node
.\mihomo-cli.exe proxy auto Proxy

# JSON format output
.\mihomo-cli.exe proxy list -o json
```

---

## Core Design Principles

### 1. Stateless

- The CLI does not persist runtime state.
- All state queries are performed in real-time against the API.
- Prevents state synchronization issues.

### 2. Configuration Persistence

- API address and Secret are saved to a local configuration file.
- Eliminates the need to input sensitive information repeatedly.
- Configuration files are stored in the user directory with appropriate permissions.

### 3. Query vs. Mutation Separation

- **Query Operations**: `get`, `list`, `show`.
- **Mutation Operations**: `set`, `switch`, `update`.
- Ensures clear semantics and distinct exit codes.

### 4. Controllable Output Format

- Supports both Table and JSON formats.
- Facilitates both human reading and script parsing.
- Unified output format specifications.

### 5. Permission Management

- Service management functions require Administrator privileges.
- System proxy modification requires Administrator privileges.
- Provides clear permission error messages.

## Documentation Resources

### Mihomo References

- `mihomo-1.19.21/`: Mihomo core reference implementation (Version 1.19.21).
