package telemetry

import (
	"context"
	"database/sql/driver"

	"github.com/XSAM/otelsql"
)

// SQLSpanOptions records statements without the driver's row, connection, and
// transaction bookkeeping, which can otherwise dominate an evaluation trace.
func SQLSpanOptions() otelsql.SpanOptions {
	return otelsql.SpanOptions{SpanFilter: func(_ context.Context, method otelsql.Method, _ string, _ []driver.NamedValue) bool {
		switch method {
		case otelsql.MethodConnExec, otelsql.MethodConnQuery, otelsql.MethodStmtExec, otelsql.MethodStmtQuery:
			return true
		default:
			return false
		}
	}}
}
