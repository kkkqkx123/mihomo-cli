# Multi-Stream Log Redirection Solution Analysis

Based on code analysis, the current project possesses basic file output capabilities (`cmd/root.go:60-72`), but lacks the functionality to output simultaneously to both the terminal and a file. The following is a detailed modification plan.

## 1. Current Implementation Analysis

### 1.1 Existing Features

**File Output Support** (`cmd/root.go:60-72`):

```go
if outputFile != "" {
    if appendMode {
        outputFileHandle, err = os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    } else {
        outputFileHandle, err = os.Create(outputFile)
    }
    output.SetGlobalStdout(outputFileHandle)
    output.SetGlobalStderr(outputFileHandle)
}
```

**Command Line Arguments**:

- `--file`, `-f`: Output to a specified file.
- `--append`: Append mode.

### 1.2 Current Issues

| Issue                             | Description                                                                        |
| :-------------------------------- | :--------------------------------------------------------------------------------- |
| **Single Stream Only**            | When outputting to a file, no information is displayed on the terminal.            |
| **Missing Dual-Stream Mode**      | Unable to output to both the terminal and a file simultaneously.                   |
| **No Configuration File Support** | Log configuration relies solely on command-line arguments; it cannot be persisted. |

---

## 2. Modification Plan

### Overview

**Core Concept**: Utilize Go's standard library `io.MultiWriter` to implement multi-stream redirection without introducing third-party libraries.

**Supported Output Modes**:

1.  **Console Mode (Default)**: Output only to the terminal.
2.  **File Mode**: Output only to a file.
3.  **Dual-Stream Mode**: Output simultaneously to both the terminal and a file.

---

## 3. Files Requiring Modification

### Modification 1: Extend Configuration Structure

**File**: `internal/config/config.go`

Add the log configuration structure:

```go
// CLIConfig CLI Tool Configuration
type CLIConfig struct {
    API    APIConfig   `mapstructure:"api"`
    Proxy  ProxyConfig `mapstructure:"proxy"`
    Log    LogConfig   `mapstructure:"log"` // New
}

// LogConfig Log Configuration
type LogConfig struct {
    File   string `mapstructure:"file"`   // Log file path
    Mode   string `mapstructure:"mode"`   // Output mode: console/file/both
    Append bool   `mapstructure:"append"` // Whether to use append mode
}
```

**Validation Function**:

```go
// Validate validates the log configuration
func (l *LogConfig) Validate() error {
    if l.Mode != "" && l.Mode != "console" && l.Mode != "file" && l.Mode != "both" {
        return errors.ErrConfig("log mode must be console, file, or both", nil)
    }
    if (l.Mode == "file" || l.Mode == "both") && l.File == "" {
        return errors.ErrConfig("log file path is required when mode is file or both", nil)
    }
    return nil
}
```

**Default Configuration**:

```go
func GetDefaultConfig() *CLIConfig {
    return &CLIConfig{
        // ... existing config
        Log: LogConfig{
            File:   "",
            Mode:   "console",
            Append: false,
        },
    }
}
```

### Modification 2: Implement Multi-Stream Writer

**File**: `internal/output/multi_writer.go` (New File)

```go
package output

import (
    "io"
    "os"
    "sync"
)

// MultiWriter Multi-stream writer
type MultiWriter struct {
    writers []io.Writer
    mu      sync.Mutex
}

// NewMultiWriter creates a multi-stream writer
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
    return &MultiWriter{
        writers: writers,
    }
}

// Write implements the io.Writer interface
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
    mw.mu.Lock()
    defer mw.mu.Unlock()

    for _, w := range mw.writers {
        n, err = w.Write(p)
        if err != nil {
            return
        }
    }
    return len(p), nil
}

// AddWriter adds a writer
func (mw *MultiWriter) AddWriter(w io.Writer) {
    mw.mu.Lock()
    defer mw.mu.Unlock()
    mw.writers = append(mw.writers, w)
}

// Close closes all closable writers
func (mw *MultiWriter) Close() error {
    mw.mu.Lock()
    defer mw.mu.Unlock()

    for _, w := range mw.writers {
        if closer, ok := w.(io.Closer); ok && w != os.Stdout && w != os.Stderr {
            if err := closer.Close(); err != nil {
                return err
            }
        }
    }
    return nil
}
```

### Modification 3: Encapsulate Log Initialization

**File**: `internal/output/log_init.go` (New File)

```go
package output

import (
    "errors"
    "os"
    "path/filepath"
)

// LogOutput Log output manager
type LogOutput struct {
    file   *os.File
    writer Writer // Assuming 'Writer' is an alias or interface defined elsewhere in your package
}

// InitLogOutput initializes log output
// mode: console - terminal only, file - file only, both - terminal + file
func InitLogOutput(filePath, mode string, append bool) (*LogOutput, error) {
    lo := &LogOutput{}

    switch mode {
    case "console", "":
        // Terminal only
        lo.writer = os.Stdout
        return lo, nil

    case "file":
        // File only
        if filePath == "" {
            return nil, errors.New("log file path is required")
        }

        file, err := lo.openFile(filePath, append)
        if err != nil {
            return nil, err
        }
        lo.file = file
        lo.writer = file
        return lo, nil

    case "both":
        // Terminal + File
        if filePath == "" {
            return nil, errors.New("log file path is required")
        }

        file, err := lo.openFile(filePath, append)
        if err != nil {
            return nil, err
        }
        lo.file = file
        lo.writer = NewMultiWriter(os.Stdout, file)
        return lo, nil

    default:
        return nil, errors.New("invalid log mode: " + mode)
    }
}

// openFile opens the log file
func (lo *LogOutput) openFile(filePath string, append bool) (*os.File, error) {
    // Ensure directory exists
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }

    // Open file
    if append {
        return os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    }
    return os.Create(filePath)
}

// GetWriter returns the writer
func (lo *LogOutput) GetWriter() Writer {
    return lo.writer
}

// Close closes the log output
func (lo *LogOutput) Close() error {
    if lo.file != nil {
        return lo.file.Close()
    }
    return nil
}
```

### Modification 4: Update Command Line Arguments

**File**: `cmd/root.go`

**Update Global Variables**:

```go
var (
    cfgFile   string
    outputFmt string
    apiURL    string
    secret    string
    timeout   int
    // Delete outputFile, appendMode; use config file instead
)
```

**Update `preRun` Function**:

```go
func preRun(cmd *cobra.Command, args []string) error {
    // Initialize configuration
    initConfig()

    // Set output format
    output.SetGlobalFormat(outputFmt)

    // Initialize log output (read from config file)
    cfg, err := config.LoadFromViper()
    if err == nil && cfg.Log.Mode != "console" && cfg.Log.Mode != "" {
        logOutput, err := output.InitLogOutput(cfg.Log.File, cfg.Log.Mode, cfg.Log.Append)
        if err != nil {
            return pkgerrors.ErrService("Failed to initialize log output", err)
        }

        // Set global output
        output.SetGlobalStdout(logOutput.GetWriter())
        output.SetGlobalStderr(logOutput.GetWriter())

        // Store in global variable for postRun cleanup
        // ...
    }

    // ... other initializations
}
```

**Update `init` Function**:

```go
func init() {
    // ... existing code

    // Remove the following lines (switched to config file)
    // rootCmd.PersistentFlags().StringVarP(&outputFile, "file", "f", "", "Output to specified file")
    // rootCmd.PersistentFlags().BoolVar(&appendMode, "append", false, "Append mode")
}
```

### Modification 5: Update Configuration Loader

**File**: `internal/config/loader.go`

**Update `Save` Function**:

```go
func (l *Loader) Save(cfg *CLIConfig, configPath string) error {
    // ... existing code

    // Set log configuration
    l.v.Set("log.file", cfg.Log.File)
    l.v.Set("log.mode", cfg.Log.Mode)
    l.v.Set("log.append", cfg.Log.Append)

    // ... subsequent code
}
```

---

## 4. Configuration File Example (`config.yaml`)

```yaml
api:
  address: http://127.0.0.1:9090
  secret: ""
  timeout: 10

proxy:
  test_url: ""
  timeout: 10000
  concurrent: 10

log:
  file: ~/.mihomo-cli/mihomo-cli.log # Log file path
  mode: both # Output mode: console/file/both
  append: true # Whether to use append mode
```

---

## 5. Usage Examples

### Scenario 1: Terminal Only (Default)

```yaml
log:
  mode: console
# Or simply omit the 'log' field in the configuration file.
```

### Scenario 2: File Only

```yaml
log:
  file: /var/log/mihomo-cli.log
  mode: file
  append: true
```

### Scenario 3: Terminal + File (Dual Stream)

```yaml
log:
  file: ~/.mihomo-cli/mihomo-cli.log
  mode: both
  append: true
```

---

## 6. Modified Files Checklist

| File                              | Modification Type | Description                                      |
| :-------------------------------- | :---------------- | :----------------------------------------------- |
| `internal/config/config.go`       | Modified          | Added `LogConfig` structure and validation logic |
| `internal/output/multi_writer.go` | Created           | Implementation of `MultiWriter` type             |
| `internal/output/log_init.go`     | Created           | Encapsulation of log initialization logic        |
| `cmd/root.go`                     | Modified          | Updated initialization logic, removed CLI flags  |
| `internal/config/loader.go`       | Modified          | Added log configuration save logic               |

---

## 7. Implementation Steps

1.  **Extend Configuration Structure**: Modify `internal/config/config.go`. Add `LogConfig` structure and validation logic.
2.  **Implement Multi-Stream Writer**: Create `internal/output/multi_writer.go`. Implement the `MultiWriter` type.
3.  **Encapsulate Log Initialization**: Create `internal/output/log_init.go`. Implement the `InitLogOutput` function.
4.  **Update Command Entry Point**: Modify `cmd/root.go`. Remove `--file` and `--append` parameters; initialize logs using the configuration file.
5.  **Update Configuration Loader**: Modify `internal/config/loader.go`. Add log configuration save logic.
6.  **Testing & Verification**: Test all three output modes, verify append mode behavior, and confirm configuration file loading.

---

## 8. Advantages Analysis

| Advantage                  | Description                                                            |
| :------------------------- | :--------------------------------------------------------------------- |
| **Zero Dependencies**      | Uses only Go standard libraries; no third-party packages required.     |
| **Simplicity**             | Controlled via configuration files; no need for complex CLI arguments. |
| **High Flexibility**       | Supports three output modes to meet various scenarios.                 |
| **Backward Compatibility** | Default behavior remains unchanged (terminal output only).             |
| **Maintainability**        | Clear code structure with single responsibilities.                     |

---

## 9. Summary

Through the proposed modifications, the project will possess complete log output capabilities:

- **Console Mode**: Default behavior, preserving the existing user experience.
- **File Mode**: Suitable for automation scripts and background operations.
- **Dual-Stream Mode**: Ideal for debugging and troubleshooting.

All modifications are based on the existing architecture, leveraging the `io.Writer` interface and `io.MultiWriter` without introducing complex logging libraries, thereby maintaining simplicity and maintainability.
