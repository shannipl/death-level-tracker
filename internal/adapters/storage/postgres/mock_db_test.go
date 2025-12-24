package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockDB implements db.DBTX interface
type MockDB struct {
	ExecFunc     func(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryFunc    func(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRowFunc func(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

func (m *MockDB) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockDB) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, arguments...)
	}
	return &MockRows{}, nil
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, arguments...)
	}
	return &MockRow{}
}

// MockRow implements pgx.Row
type MockRow struct {
	ScanFunc func(dest ...interface{}) error
}

func (m *MockRow) Scan(dest ...interface{}) error {
	if m.ScanFunc != nil {
		return m.ScanFunc(dest...)
	}
	return nil
}

// MockRows implements pgx.Rows
type MockRows struct {
	NextFunc  func() bool
	ScanFunc  func(dest ...interface{}) error
	CloseFunc func()
	ErrFunc   func() error
}

func (m *MockRows) Close() {
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

func (m *MockRows) Err() error {
	if m.ErrFunc != nil {
		return m.ErrFunc()
	}
	return nil
}

func (m *MockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (m *MockRows) Next() bool {
	if m.NextFunc != nil {
		return m.NextFunc()
	}
	return false
}

func (m *MockRows) Scan(dest ...interface{}) error {
	if m.ScanFunc != nil {
		return m.ScanFunc(dest...)
	}
	return nil
}

func (m *MockRows) Values() ([]any, error) { return nil, nil }
func (m *MockRows) RawValues() [][]byte    { return nil }

func (m *MockRows) Conn() *pgx.Conn { return nil }
