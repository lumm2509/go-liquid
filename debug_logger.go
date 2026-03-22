package liquid

import "fmt"

// StdoutLogger is a convenience DebugLogger for development; production callers should use
// their own logger (slog, zap, logrus, etc.)
type StdoutLogger struct{}

func (l StdoutLogger) Log(e DebugEvent) {
	fmt.Printf("[liquid:%s] %v\n", e.Event, e.Data)
}
