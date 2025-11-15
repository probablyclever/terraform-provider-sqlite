package provider

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	_ "modernc.org/sqlite"
)

//go:embed testdata/drseuss.sql
var drSeussSQL []byte

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"sqlite": providerserver.NewProtocol6WithError(New()),
}

func TestAccSQLiteQueryDataSource_basic(t *testing.T) {
	dbPath := buildDrSeussDB(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccSQLiteQueryConfig(dbPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.sqlite_query.fish", "rows.#", "7"),
					resource.TestCheckResourceAttr("data.sqlite_query.fish", "rows.0.name", "star fish"),
					resource.TestCheckResourceAttr("data.sqlite_query.fish", "rows.6.name", "red fish"),
					resource.TestCheckResourceAttrSet("data.sqlite_query.fish", "result_json"),
				),
			},
		},
	})
}

func TestAccSQLiteQueryDataSource_withParams(t *testing.T) {
	dbPath := buildDrSeussDB(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccSQLiteQueryConfigWithParams(dbPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.sqlite_query.red", "rows.#", "1"),
					resource.TestCheckResourceAttr("data.sqlite_query.red", "rows.0.name", "red fish"),
					resource.TestCheckResourceAttr("data.sqlite_query.red", "rows.0.color", "red"),
					resource.TestCheckResourceAttr("data.sqlite_query.red", "rows.0.count", "1"),
				),
			},
		},
	})
}

func TestAccSQLiteQueryDataSource_rejectsNonSelect(t *testing.T) {
	dbPath := buildDrSeussDB(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccSQLiteQueryConfigNonSelect(dbPath),
				ExpectError: regexp.MustCompile(`query must be SELECT`),
			},
		},
	})
}

func TestAccSQLiteQueryDataSource_queryError(t *testing.T) {
	dbPath := buildDrSeussDB(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccSQLiteQueryConfigBadTable(dbPath),
				ExpectError: regexp.MustCompile(`query failed`),
			},
		},
	})
}

func TestAccSQLiteQueryDataSource_missingDB(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccSQLiteQueryMissingDBConfig("/path/does/not/exist.db"),
				ExpectError: regexp.MustCompile(`unable to open database file`),
			},
		},
	})
}

func TestAccSQLiteQueryDataSource_missingQueryArg(t *testing.T) {
	dbPath := buildDrSeussDB(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				// No query attribute; should trigger config diagnostics.
				Config:      testAccSQLiteQueryMissingQueryConfig(dbPath),
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
		},
	})
}

func TestAccSQLiteQueryDataSource_badParamsType(t *testing.T) {
	dbPath := buildDrSeussDB(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config:      testAccSQLiteQueryBadParamsConfig(dbPath),
				ExpectError: regexp.MustCompile(`missing named argument`),
			},
		},
	})
}

func TestAccSQLiteQueryDataSource_blobAndNull(t *testing.T) {
	dbPath := buildDrSeussDB(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccSQLiteQueryBlobNullConfig(dbPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.sqlite_query.blobnull", "rows.#", "1"),
					resource.TestCheckResourceAttr("data.sqlite_query.blobnull", "rows.0.color", ""),
					// Non-UTF8 bytes are replaced on string conversion; NUL is preserved.
					resource.TestCheckResourceAttr("data.sqlite_query.blobnull", "rows.0.blobcol", "\x00�"),
				),
			},
		},
	})
}

func buildDrSeussDB(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "drseuss.db")

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=rwc", dbPath))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	stmts := string(drSeussSQL)
	if _, err := db.Exec(stmts); err != nil {
		t.Fatalf("execute fixture SQL: %v", err)
	}

	return dbPath
}

func testAccSQLiteQueryConfig(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "fish" {
  db_path = %q
  query = <<-SQL
    SELECT name, color, count
    FROM fish
    ORDER BY count DESC, name ASC;
  SQL
}
`, dbPath)
}

func testAccSQLiteQueryConfigWithParams(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "red" {
  db_path = %q
  query = <<-SQL
    SELECT name, color, count
    FROM fish
    WHERE color = :color
  SQL
  params = {
    color = "red"
  }
}
`, dbPath)
}

func testAccSQLiteQueryConfigNonSelect(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "bad" {
  db_path = %q
  query = <<-SQL
    UPDATE fish SET count = 99;
  SQL
}
`, dbPath)
}

func testAccSQLiteQueryConfigBadTable(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "bad_table" {
  db_path = %q
  query = <<-SQL
    SELECT name FROM missing_table;
  SQL
}
`, dbPath)
}

func testAccSQLiteQueryMissingDBConfig(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "missing" {
  db_path = %q
  query   = "SELECT 1;"
}
`, dbPath)
}

func testAccSQLiteQueryMissingQueryConfig(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "missing_query" {
  db_path = %q
}
`, dbPath)
}

func testAccSQLiteQueryBadParamsConfig(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "bad_params" {
  db_path = %q
  query   = "SELECT name FROM fish WHERE count = :count"
  params = {
    poop = 1
  }
}
`, dbPath)
}

func testAccSQLiteQueryBlobNullConfig(dbPath string) string {
	return fmt.Sprintf(`
data "sqlite_query" "blobnull" {
  db_path = %q
  query   = "SELECT NULL as color, x'00ff' as blobcol;"
}
`, dbPath)
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("TF_ACC"); v == "" {
		t.Skip("TF_ACC must be set to run acceptance tests")
	}
}
