# SQLite

This package contains the repository interfaces' SQLite implementation of all services.

It uses package `github.com/Masterminds/squirrel` for working with sql.

It uses package `github.com/golang-migrate/migrate` for migrations.

Tables have foreign key to each other if are related.
