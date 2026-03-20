package liquid

import "fmt"

// StdoutLogger es un DebugLogger de conveniencia para desarrollo.
// Escribe cada evento a stdout. En producción, el consumer debe
// implementar su propio DebugLogger usando su stack de logging (slog, zap, logrus, etc.).
type StdoutLogger struct{}

func (l StdoutLogger) Log(e DebugEvent) {
	fmt.Printf("[liquid:%s] %v\n", e.Event, e.Data)
}
