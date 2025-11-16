# Data Source: `sqlite_query`

Runs a read-only SQL query against a SQLite database file and returns the rows as both a list of maps and a JSON string.

> Note: Only `SELECT` statements are allowed. Queries attempting to modify data will fail.

## Example

```hcl
provider "sqlite" {}

data "sqlite_query" "fish" {
  db_path = "${path.module}/drseuss.db"
  query   = <<-SQL
    SELECT name, color, count
    FROM fish
    WHERE color LIKE :color
    ORDER BY count DESC;
  SQL

  params = {
    color = "b%"
  }
}

output "fish_rows" {
  value = data.sqlite_query.fish.rows
}

output "fish_json" {
  value = data.sqlite_query.fish.result_json
}
```

## Argument Reference

- `db_path` (Required, String) Path to the SQLite database file.
- `query` (Required, String) SQL query to run. Must be a `SELECT`.
- `params` (Optional, Map of String) Named parameters passed to the query. Keys should match placeholders without the leading `:` (e.g., `params = { color = "blue" }` for `:color` in the query).

## Attributes Reference

- `rows` (List of Map(String)) Result rows, each as a map of column name to string value. Non-string values are stringified; `NULL` becomes an empty string.
- `result_json` (String) Entire result encoded as JSON array of objects.
