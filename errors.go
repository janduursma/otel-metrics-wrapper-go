package metrics

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// classifyError inspects the given error and returns a
// string-based category ("timeout", "network", "invalid_input" etc.)
// This allows tracking the number of errors that fall into the different categories.
func classifyError(err error) string {
	if err == nil {
		return "" // no error
	}

	// Context-level checks (canceled, timed out).
	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	}

	// Network errors (using net.Error interface).
	// This categorizes typical transient vs. permanent network issues.
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "network_timeout"
		}
		return "network"
	}

	// Check for parse/syntax errors.
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "parse") || strings.Contains(strings.ToLower(msg), "syntax") {
		return "invalid_input"
	}

	//  Check for known PostgreSQL DB errors.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return "db_unique_violation"
		case pgerrcode.ForeignKeyViolation:
			return "db_fk_violation"
		default:
			return "db_error"
		}
	}

	// Check for gRPC errors.
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.DeadlineExceeded:
			return "grpc_timeout"
		case codes.NotFound:
			return "grpc_not_found"
		case codes.InvalidArgument:
			return "grpc_invalid_arg"
		default:
			return "grpc_" + s.Code().String()
		}
	}

	// Default or unknown.
	return "unknown"
}
