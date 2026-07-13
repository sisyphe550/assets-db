package model

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
)

type conflictNullDriver struct{}

func (conflictNullDriver) Open(string) (driver.Conn, error) { return conflictNullConn{}, nil }

type conflictNullConn struct{}

func (conflictNullConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (conflictNullConn) Close() error              { return nil }
func (conflictNullConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (conflictNullConn) QueryContext(
	context.Context,
	string,
	[]driver.NamedValue,
) (driver.Rows, error) {
	return &conflictNullRows{}, nil
}

type conflictNullRows struct{ returned bool }

func (r *conflictNullRows) Columns() []string {
	return []string{
		"id", "task_id", "asset_no", "asset_id", "status", "candidates",
		"resolved_source", "resolved_operator_id", "resolved_actual_location",
		"resolved_notes", "resolved_by", "resolved_at",
	}
}
func (r *conflictNullRows) Close() error { return nil }
func (r *conflictNullRows) Next(dest []driver.Value) error {
	if r.returned {
		return io.EOF
	}
	r.returned = true
	copy(dest, []driver.Value{
		int64(1), int64(12), "EQUIP-2026-0001", int64(1), int64(0),
		[]byte(`[{"operatorId":10003,"actualLocation":"一号实验楼103"}]`),
		nil, nil, nil, nil, nil, nil,
	})
	return nil
}

func TestListConflictsAcceptsNullResolutionFieldsForPendingRows(t *testing.T) {
	sql.Register("inventory-conflict-null", conflictNullDriver{})
	db, err := sql.Open("inventory-conflict-null", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conflicts, err := NewInvModel(db).ListConflicts(context.Background(), 12, true)
	if err != nil {
		t.Fatalf("pending conflict with NULL resolution fields must be readable: %v", err)
	}
	if len(conflicts) != 1 || conflicts[0].AssetNo != "EQUIP-2026-0001" {
		t.Fatalf("unexpected conflicts: %#v", conflicts)
	}
}
