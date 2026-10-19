package telemetry

import (
	"context"
	"database/sql/driver"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/otel/trace"
)

// SQLSpanOptions records statements without the driver's row, connection, and
// transaction bookkeeping, which can otherwise dominate an evaluation trace.
// SQL only extends existing traces; startup and untraced background work must
// not create a separate trace for each statement.
func SQLSpanOptions() otelsql.SpanOptions {
	return otelsql.SpanOptions{
		SpanFilter: func(ctx context.Context, method otelsql.Method, _ string, _ []driver.NamedValue) bool {
			if !trace.SpanContextFromContext(ctx).IsValid() {
				return false
			}
			switch method {
			case otelsql.MethodConnExec, otelsql.MethodConnQuery, otelsql.MethodStmtExec, otelsql.MethodStmtQuery:
				return true
			default:
				return false
			}
		},
	}
}
