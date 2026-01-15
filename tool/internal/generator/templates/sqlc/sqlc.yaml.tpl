version: "2"
sql:
  - engine: "{{.Engine}}"
    queries: "query/"
    schema: "{{.SchemaPath}}"
    gen:
      go:
        package: "db"
        out: "internal/data/db"
        sql_package: "{{.SQLPackage}}"
        emit_json_tags: true
        emit_interface: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true
        overrides:
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "timestamp"
            go_type: "time.Time"