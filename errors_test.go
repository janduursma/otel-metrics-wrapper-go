package metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// dummyNetError is a simple implementation of net.Error for testing.
type dummyNetError struct {
	msg     string
	timeout bool
}

func (d dummyNetError) Error() string   { return d.msg }
func (d dummyNetError) Timeout() bool   { return d.timeout }
func (d dummyNetError) Temporary() bool { return false }

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: "canceled",
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: "timeout",
		},
		{
			name:     "network timeout",
			err:      dummyNetError{"network issue", true},
			expected: "network_timeout",
		},
		{
			name:     "network non-timeout",
			err:      dummyNetError{"network issue", false},
			expected: "network",
		},
		{
			name:     "parse error",
			err:      errors.New("failed to parse JSON input"),
			expected: "invalid_input",
		},
		{
			name:     "syntax error",
			err:      errors.New("SYNTAX error near unexpected token"),
			expected: "invalid_input",
		},
		{
			name:     "pg unique violation",
			err:      &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			expected: "db_unique_violation",
		},
		{
			name:     "pg foreign key violation",
			err:      &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation},
			expected: "db_fk_violation",
		},
		{
			name:     "pg other error",
			err:      &pgconn.PgError{Code: "99999"},
			expected: "db_error",
		},
		{
			name:     "grpc deadline exceeded",
			err:      status.Error(codes.DeadlineExceeded, "deadline exceeded"),
			expected: "grpc_timeout",
		},
		{
			name:     "grpc not found",
			err:      status.Error(codes.NotFound, "not found"),
			expected: "grpc_not_found",
		},
		{
			name:     "grpc invalid argument",
			err:      status.Error(codes.InvalidArgument, "invalid argument"),
			expected: "grpc_invalid_arg",
		},
		{
			name:     "grpc internal",
			err:      status.Error(codes.Internal, "internal error occurred"),
			expected: "grpc_Internal",
		},
		{
			name:     "unknown error",
			err:      errors.New("an unexpected error occurred"),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			require.Equal(t, tt.expected, result, "for test %q", tt.name)
		})
	}
}
