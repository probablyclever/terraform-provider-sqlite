package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type stubDB struct {
	queryFn func(string, []any) (rowIterator, error)
}

func (s *stubDB) QueryContext(ctx context.Context, query string, args ...any) (rowIterator, error) {
	return s.queryFn(query, args)
}

func (s *stubDB) Close() error { return nil }

type stubRows struct {
	cols    []string
	next    []bool
	scanErr error
	err     error
}

func (r *stubRows) Columns() ([]string, error) {
	if r.cols == nil {
		return nil, errors.New("boom columns")
	}
	return r.cols, nil
}
func (r *stubRows) Next() bool {
	if len(r.next) == 0 {
		return false
	}
	val := r.next[0]
	r.next = r.next[1:]
	return val
}
func (r *stubRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for i := range dest {
		if s, ok := dest[i].(*any); ok {
			*s = "x"
		}
	}
	return nil
}
func (r *stubRows) Err() error   { return r.err }
func (r *stubRows) Close() error { return nil }

func withStubDB(t *testing.T, stub *stubDB, fn func()) {
	t.Helper()
	orig := newSQLiteDB
	newSQLiteDB = func(string) (dbRunner, error) { return stub, nil }
	defer func() { newSQLiteDB = orig }()
	fn()
}

func TestRead_openError(t *testing.T) {
	orig := newSQLiteDB
	newSQLiteDB = func(string) (dbRunner, error) { return nil, errors.New("open boom") }
	defer func() { newSQLiteDB = orig }()

	ds := &sqliteQueryDataSource{}
	ctx := context.Background()
	model := sqliteQueryModel{
		DBPath: types.StringValue("/tmp/missing.db"),
		Query:  types.StringValue("SELECT 1"),
	}
	var resp datasource.ReadResponse
	ds.readWithModel(ctx, &model, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics on open error")
	}
}

func TestRead_columnsError(t *testing.T) {
	withStubDB(t, &stubDB{
		queryFn: func(_ string, _ []any) (rowIterator, error) {
			return &stubRows{cols: nil}, nil
		},
	}, func() {
		ds := &sqliteQueryDataSource{}
		ctx := context.Background()
		model := sqliteQueryModel{
			DBPath: types.StringValue("/tmp/any.db"),
			Query:  types.StringValue("SELECT 1"),
		}
		var resp datasource.ReadResponse
		ds.readWithModel(ctx, &model, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics on columns error")
		}
	})
}

func TestRead_scanError(t *testing.T) {
	withStubDB(t, &stubDB{
		queryFn: func(_ string, _ []any) (rowIterator, error) {
			return &stubRows{
				cols:    []string{"c"},
				next:    []bool{true},
				scanErr: errors.New("scan boom"),
			}, nil
		},
	}, func() {
		ds := &sqliteQueryDataSource{}
		ctx := context.Background()
		model := sqliteQueryModel{
			DBPath: types.StringValue("/tmp/any.db"),
			Query:  types.StringValue("SELECT 1"),
		}
		var resp datasource.ReadResponse
		ds.readWithModel(ctx, &model, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics on scan error")
		}
	})
}

func TestRead_rowsErr(t *testing.T) {
	withStubDB(t, &stubDB{
		queryFn: func(_ string, _ []any) (rowIterator, error) {
			return &stubRows{
				cols: []string{"c"},
				next: []bool{},
				err:  errors.New("rows boom"),
			}, nil
		},
	}, func() {
		ds := &sqliteQueryDataSource{}
		ctx := context.Background()
		model := sqliteQueryModel{
			DBPath: types.StringValue("/tmp/any.db"),
			Query:  types.StringValue("SELECT 1"),
		}
		var resp datasource.ReadResponse
		ds.readWithModel(ctx, &model, &resp)

		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected diagnostics on rows error")
		}
	})
}
