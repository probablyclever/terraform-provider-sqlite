provider "sqlite" {}

# Example query against the Dr. Seuss fish table.
data "sqlite_query" "fish" {
  # Point to the Dr. Seuss sample database (generated in tests/CI).
  db_path = "${path.module}/../../internal/provider/testdata/drseuss.db"

  query = <<-SQL
    SELECT name, color, count
    FROM fish
    ORDER BY count DESC, name ASC;
  SQL
}

output "fish_rows" {
  value = data.sqlite_query.fish.rows
}

output "fish_json" {
  value = data.sqlite_query.fish.result_json
}
