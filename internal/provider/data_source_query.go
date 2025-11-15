package provider

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	_ "modernc.org/sqlite"
)

type sqliteQueryDataSource struct{}

func NewSQLiteQueryDataSource() datasource.DataSource {
	return &sqliteQueryDataSource{}
}

// newSQLiteDB is a test seam so unit tests can stub database behavior.
var newSQLiteDB = func(dsn string) (dbRunner, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &sqlDBRunner{db: db}, nil
}

type dbRunner interface {
	QueryContext(ctx context.Context, query string, args ...any) (rowIterator, error)
	Close() error
}

type rowIterator interface {
	Columns() ([]string, error)
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

type sqlDBRunner struct {
	db *sql.DB
}

func (r *sqlDBRunner) QueryContext(ctx context.Context, query string, args ...any) (rowIterator, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsAdapter{rows: rows}, nil
}

func (r *sqlDBRunner) Close() error { return r.db.Close() }

type sqlRowsAdapter struct {
	rows *sql.Rows
}

func (r *sqlRowsAdapter) Columns() ([]string, error) { return r.rows.Columns() }
func (r *sqlRowsAdapter) Next() bool                 { return r.rows.Next() }
func (r *sqlRowsAdapter) Scan(dest ...any) error     { return r.rows.Scan(dest...) }
func (r *sqlRowsAdapter) Err() error                 { return r.rows.Err() }
func (r *sqlRowsAdapter) Close() error               { return r.rows.Close() }

type sqliteQueryModel struct {
	DBPath     types.String `tfsdk:"db_path"`
	Query      types.String `tfsdk:"query"`
	Params     types.Map    `tfsdk:"params"`
	Rows       types.List   `tfsdk:"rows"`
	ResultJSON types.String `tfsdk:"result_json"`
}

func (d *sqliteQueryDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "sqlite_query"
}

func (d *sqliteQueryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Execute a read-only SQLite query and return rows.",
		Attributes: map[string]dsschema.Attribute{
			"db_path": dsschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to the SQLite database file.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"query": dsschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "SQL query to run. Use named params like `:env`.",
			},
			"params": dsschema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Named parameters passed to the query (map of string). Keys should match placeholders without the `:` prefix.",
			},
			"rows": dsschema.ListAttribute{
				Computed:    true,
				ElementType: types.MapType{ElemType: types.StringType},
				MarkdownDescription: "Query result as a list of rows. Each row is a map(string). " +
					"Non-string values are stringified.",
			},
			"result_json": dsschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entire result encoded as JSON (array of objects).",
			},
		},
	}
}

func (d *sqliteQueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg sqliteQueryModel
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	d.readWithModel(ctx, &cfg, resp)
}

func (d *sqliteQueryDataSource) readWithModel(ctx context.Context, cfg *sqliteQueryModel, resp *datasource.ReadResponse) {
	dbPath := cfg.DBPath.ValueString()
	query := cfg.Query.ValueString()

	var args []any
	if !cfg.Params.IsNull() {
		p := map[string]string{}
		diags := cfg.Params.ElementsAs(ctx, &p, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for name, val := range p {
			args = append(args, sql.Named(name, val))
		}
	}

	tflog.Debug(ctx, "sqlite_query opening database", map[string]any{"db_path": dbPath})
	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath) // Open R/O since that's all we want this provider to be
	db, err := newSQLiteDB(dsn)
	if err != nil {
		resp.Diagnostics.AddError("open sqlite db failed", err.Error())
		return
	}
	defer db.Close()

	// Read-only safety: enforce query starts with SELECT.
	if len(query) < 6 || (query[:6] != "SELECT" && query[:6] != "select") {
		resp.Diagnostics.AddError("query must be SELECT", "Only read-only SELECT queries are allowed.")
		return
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		resp.Diagnostics.AddError("query failed", err.Error())
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		resp.Diagnostics.AddError("columns failed", err.Error())
		return
	}

	out := make([]map[string]string, 0, 16)
	for rows.Next() {
		// Scan all cols into []any, then stringify.
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			resp.Diagnostics.AddError("scan failed", err.Error())
			return
		}
		row := make(map[string]string, len(cols))
		for i, c := range cols {
			switch v := vals[i].(type) {
			case []byte:
				row[c] = string(v)
			case nil:
				row[c] = ""
			default:
				row[c] = fmt.Sprint(v)
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		resp.Diagnostics.AddError("row iteration failed", err.Error())
		return
	}

	var rowElems []attr.Value
	for _, r := range out {
		elems := make(map[string]attr.Value, len(r))
		for k, v := range r {
			elems[k] = types.StringValue(v)
		}
		rowElems = append(rowElems, types.MapValueMust(types.StringType, elems))
	}
	rowsList := types.ListValueMust(
		types.MapType{ElemType: types.StringType},
		rowElems,
	)
	jsonBytes, _ := json.Marshal(out)

	cfg.Rows = rowsList
	cfg.ResultJSON = types.StringValue(string(jsonBytes))

	diags := resp.State.Set(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
}
