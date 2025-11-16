# SQLite Provider

Use this provider to run read-only SQL queries against SQLite databases from Terraform.

## Example usage

```hcl
terraform {
  required_providers {
    sqlite = {
      source  = "registry.terraform.io/probablyclever/sqlite"
      version = "1.0.0"
    }
  }
}

provider "sqlite" {}
```

## Data Sources

- [`sqlite_query`](data-sources/sqlite_query.md) – execute a read-only query and return rows or JSON.
