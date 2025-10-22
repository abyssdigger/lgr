# lgr

A lightweight, multi-out logging package for Go.  
Processes all output logs in a separate goroutine to minimize log caller delay.
Provides per-client and per-output configuration for timestamp, level names, ansi colors etc.

## Features

- Multiple log levels: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, UNMASKABLE
- Thread-safe*, buffered logging with background processing for minimal caller i/o waiting
- In-app multiple disengageable logger clients with own names and log level settings
- Global, per-client and per-output level-based filtering
- Color and prefix customization per output
- Fallback writer for logger error reporting
- Error-returning and convenience logging methods
- Implements `io.Writer` interface for use with `fmt.Fprintf`* etc.

_*Be careful with `io.Writer` usage: fmt module is not thread-safe, so unpredictable side effects can happen when calling `fmt.Frint*(client, "message")` from separated goroutines. Good for a configurations with one logging goroutine. For multi-goroutines use thread-safe `client.Log*()` methods instead._

## Basic Usage

### Initialization

```go
import "lgr"

logger := lgr.InitAndStart(32, os.Stdout, os.Stderr) // Start with buffer size 32 and two outputs
defer logger.StopAndWait() // Ensure graceful shutdown
```

### Creating a Client

```go
client := logger.NewClient("my-service")
```

### Logging Messages

```go
client.LogInfo("Service started")
client.LogWarn("Low disk space")
client.LogError("Could not open file")
client.LogErr(errors.New("network unreachable"))
```

### Using io.Writer Interface

```go
// Do not use in goroutines! fmt.Fprint*() is not thread-safe!
fmt.Fprintf(client.Lvl(lgr.LVL_WARN), "disk space low: %d%%\n", percent)
```

### Handling Errors

For reliable error handling, use the `_with_err` variants:

```go
t, err := client.Log_with_err(lgr.LVL_ERROR, "critical failure")
if err != nil {
    // fallback or alert
}
```

### Output Customization

```go
// Line by line:
logger.SetOutputLevelPrefix(os.Stdout, lgr.LevelShortNames, ": ")
logger.SetOutputLevelColor(os.Stdout, lgr.LevelColorOnBlackMap)
logger.SetOutputTimeFormat(os.Stdout, "2006-01-02 15:04:05 ")
logger.ShowOutputLevelCode(os.Stdout)
// All together:
logger.SetOutputLevelPrefix(myLogFile, lgr.LevelFullNames, "| ").SetOutputTimeFormat(myLogFile, "2006-01-02 15:04:05 ").ShowOutputLevelCode(myLogFile)
```

## Log Level Filtering

- Each client and output can have its own minimum log level.
- Messages below the configured levels are just ignored.

## Error Handling

- If a log cannot be delivered, errors are sent to the fallback writer.
- Use `_with_err` methods to handle errors directly.

## License

MIT

_This file is based on CoPilot (GPT-4.1) generation_