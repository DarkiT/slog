package multi

import (
	"log/slog"
)

// Pipe defines a chain of Middleware.
type PipeBuilder struct {
	middlewares []Middleware
}

// Pipe builds a chain of Middleware.
// Eg: rewrite log.Record on the fly for privacy reason.
func Pipe(middlewares ...Middleware) *PipeBuilder {
	copied := make([]Middleware, len(middlewares))
	copy(copied, middlewares)
	return &PipeBuilder{middlewares: copied}
}

// Implements slog.Handler
func (h *PipeBuilder) Pipe(middleware Middleware) *PipeBuilder {
	h.middlewares = append(h.middlewares, middleware)
	return h
}

// Implements slog.Handler
func (h *PipeBuilder) Handler(handler slog.Handler) slog.Handler {
	if len(h.middlewares) == 0 {
		return handler
	}

	chain := make([]Middleware, len(h.middlewares))
	copy(chain, h.middlewares)

	for i := len(chain) - 1; i >= 0; i-- {
		handler = chain[i](handler)
	}

	return handler
}
