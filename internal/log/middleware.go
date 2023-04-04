package log

import "net/http"

var (
	// LogEntryCtxKey is the context.Context key to store the request log entry.
	// LogEntryCtxKey = &contextKey{"LogEntry"}

	// DefaultLogger is called by the Logger middleware handler to log each request.
	// Its made a package-level variable so that it can be reconfigured for custom
	// logging configurations.
	DefaultLogger func(next http.Handler) http.Handler
)

func LoggerMiddleware(next http.Handler) http.Handler {
	return DefaultLogger(next)
}
