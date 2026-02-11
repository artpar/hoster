package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Handler processes a command dispatched by the state machine.
type Handler func(ctx context.Context, deps *Deps, data map[string]any) error

// Deps holds dependencies available to all command handlers.
type Deps struct {
	Store  *Store
	Logger *slog.Logger
	// Additional dependencies are set by the application
	Extra map[string]any
}

// Bus implements CommandBus by dispatching to registered handlers.
type Bus struct {
	handlers map[string]Handler
	deps     *Deps
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewBus creates a new command bus.
func NewBus(store *Store, logger *slog.Logger) *Bus {
	if logger == nil {
		logger = slog.Default()
	}
	return &Bus{
		handlers: make(map[string]Handler),
		deps: &Deps{
			Store:  store,
			Logger: logger,
			Extra:  make(map[string]any),
		},
		logger: logger,
	}
}

// SetExtra sets an extra dependency available to all handlers.
func (b *Bus) SetExtra(key string, value any) {
	b.deps.Extra[key] = value
}

// Register registers a handler for a command name.
func (b *Bus) Register(command string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[command] = handler
}

// Dispatch dispatches a command to its registered handler.
func (b *Bus) Dispatch(ctx context.Context, command string, data map[string]any) error {
	b.mu.RLock()
	handler, ok := b.handlers[command]
	b.mu.RUnlock()

	if !ok {
		b.logger.Warn("no handler registered for command", "command", command)
		return nil // Don't fail — just log
	}

	b.logger.Debug("dispatching command", "command", command)
	if err := handler(ctx, b.deps, data); err != nil {
		b.logger.Error("command failed", "command", command, "error", err)
		return fmt.Errorf("command %s: %w", command, err)
	}

	return nil
}
