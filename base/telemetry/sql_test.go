package telemetry

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/XSAM/otelsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newSQLTestDB(t *testing.T) (*sql.DB, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	require.NoError(t, initWithExporter(exporter))
	t.Cleanup(func() { require.NoError(t, Shutdown(context.Background())) })
	db := otelsql.OpenDB(sqlTestConnector{}, otelsql.WithSpanOptions(SQLSpanOptions()))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, exporter
}

func TestSQLStatementSpans(t *testing.T) {
	db, exporter := newSQLTestDB(t)
	ctx, parent := Tracer().Start(t.Context(), "evaluate upload")
	require.NoError(t, db.PingContext(ctx))
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "UPDATE packages SET synced = true")
	require.NoError(t, err)
	var value int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT 1").Scan(&value))
	assert.Equal(t, 1, value)
	stmt, err := tx.PrepareContext(ctx, "SELECT 1")
	require.NoError(t, err)
	require.NoError(t, stmt.QueryRowContext(ctx).Scan(&value))
	_, err = stmt.ExecContext(ctx)
	require.NoError(t, err)
	require.NoError(t, stmt.Close())
	require.NoError(t, tx.Commit())
	tx, err = db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	_, err = db.ExecContext(ctx, "invalid statement")
	require.ErrorIs(t, err, errSQLTest)
	parent.End()

	spans := exporter.GetSpans()
	names := make([]string, 0, len(spans))
	errorSpans := 0
	for _, span := range spans {
		names = append(names, span.Name)
		if span.Name == "evaluate upload" {
			continue
		}
		assert.Equal(t, parent.SpanContext().TraceID(), span.SpanContext.TraceID())
		assert.Equal(t, parent.SpanContext().SpanID(), span.Parent.SpanID())
		if span.Status.Code == codes.Error {
			errorSpans++
			assert.Equal(t, "invalid statement", attr(span, "db.statement"))
			require.NotEmpty(t, span.Events)
		}
	}
	assert.ElementsMatch(t, []string{
		"evaluate upload", "sql.conn.exec", "sql.conn.query", "sql.stmt.query", "sql.stmt.exec", "sql.conn.exec",
	}, names)
	assert.Equal(t, 1, errorSpans)
}

var errSQLTest = errors.New("invalid SQL statement")

type sqlTestConnector struct{}

func (sqlTestConnector) Connect(context.Context) (driver.Conn, error) { return sqlTestConn{}, nil }
func (sqlTestConnector) Driver() driver.Driver                        { return sqlTestDriver{} }

type sqlTestDriver struct{}

func (sqlTestDriver) Open(string) (driver.Conn, error) { return sqlTestConn{}, nil }

type sqlTestConn struct{}

func (sqlTestConn) Prepare(string) (driver.Stmt, error) { return sqlTestStmt{}, nil }
func (sqlTestConn) Close() error                        { return nil }
func (sqlTestConn) Begin() (driver.Tx, error)           { return sqlTestTx{}, nil }
func (sqlTestConn) Ping(context.Context) error          { return nil }
func (sqlTestConn) ResetSession(context.Context) error  { return nil }

func (sqlTestConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if query == "invalid statement" {
		return nil, errSQLTest
	}
	return driver.RowsAffected(1), nil
}

func (sqlTestConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &sqlTestRows{}, nil
}

type sqlTestTx struct{}

func (sqlTestTx) Commit() error   { return nil }
func (sqlTestTx) Rollback() error { return nil }

type sqlTestStmt struct{}

func (sqlTestStmt) Close() error  { return nil }
func (sqlTestStmt) NumInput() int { return 0 }
func (sqlTestStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}
func (sqlTestStmt) Query([]driver.Value) (driver.Rows, error) { return &sqlTestRows{}, nil }

type sqlTestRows struct{ read bool }

func (*sqlTestRows) Columns() []string { return []string{"value"} }
func (*sqlTestRows) Close() error      { return nil }
func (r *sqlTestRows) Next(values []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	values[0] = int64(1)
	return nil
}
