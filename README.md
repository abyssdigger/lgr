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
- Implements `io.Writer` interface for use with `fmt.Fprintf(...)`* etc.

_\*Be careful with **io.Writer** usage: fmt module is not thread-safe, so unpredictable side effects can happen when calling **fmt.Frint\*(client, "message")** from separated goroutines. Good enough for a configurations with one logging goroutine, but for multi-goroutines use thread-safe **client.Log\*()** methods instead._

## Basic Usage

### Initialization (simple)

```go
import "lgr"
file, _ := os.Open("file.log")
// Start with default buffer size and two outputs (stdout and custom file).
// Logger-wide ERROR log level is set by default (DEFAULT_LOG_LEVEL).
logger := lgr.InitAndStart(DEFAULT_MSG_BUFF, os.Stdout, file) 
defer logger.StopAndWait() // Ensure graceful shutdown
logger.SetMinLevel(LVL_UNKNOWN) // All levels are allowed per logger
```

### Output Customization

```go
// Line by line:
logger.SetOutputLevelPrefix(os.Stdout, lgr.LevelShortNames, ": ") // Short level names
logger.SetOutputLevelColor(os.Stdout, lgr.LevelColorOnBlackMap)   // ANSI term colors
logger.SetOutputTimeFormat(file, "2006-01-02 15:04:05", " ")      // Timestamp
// Every setter returns logger, so can be called in chains -
// here level numeric codes and full level names are set in one line:
logger.ShowOutputLevelCode(file).SetOutputLevelPrefix(file, lgr.LevelFullNames, "|")
```

### Creating a Client

```go
client := logger.NewClient("my-service")
```

### Logging Messages

```go
client.LogDebug("Custom preparations") // written to all outputs
client.LogInfo("Service started")      // written to all outputs
client.LogWarn("Low disk space")       // written to all outputs
client.LogError("Could not open file") // written to all outputs
```

### Change client minimal log level

```go
logger.SetClientMinLevel(client, LVL_WARN)
// ...
// in client's goroutine:
client.LogDebug("Custom preparations") // >>> ignored
client.LogInfo("Service started")      // >>> ignored
client.LogWarn("Low disk space")       // written to all outputs
client.LogError("Could not open file") // written to all outputs
```

### Change output minimal log level

```go
logger.SetOutputMinLevel(file, LVL_ERROR)
// ...
// in client's goroutine:
client.LogDebug("Custom preparations") // ignored due to previous client restrictions
client.LogInfo("Service started")      // ignored due to previous client restrictions
client.LogWarn("Low disk space")       // >>> ignored by file, written to os.Stdout
client.LogError("Could not open file") // written to all outputslevel 
```

### Using io.Writer Interface

```go
// Do not use in goroutines! fmt.Fprint*() is not thread-safe!
fmt.Fprintf(client.Lvl(lgr.LVL_WARN), "disk space low: %d%%\n", percent)
```

### Handling Errors

For reliable error handling, use the `_with_err` log variants:

```go
t, err := client.Log_with_err(lgr.LVL_ERROR, "critical failure")
if err != nil {
    // fallback or alert
}
```

## Log Level Filtering

- Each client and output can have its own minimum log level.
- Messages below the configured levels are just ignored (the ignore can be catched by zero publish time returned).

## Error Handling

- If a log cannot be delivered, errors are sent to the fallback writer.
- Use `_with_err` methods to handle errors directly.

## License

MIT

_This file is based on CoPilot (GPT-4.1) generation_