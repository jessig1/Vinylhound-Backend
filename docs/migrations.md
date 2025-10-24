# Database Migrations

This project uses the [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) CLI to manage schema changes. Migrations live in the `migrations/` directory and follow the `<version>_<name>.up.sql / <version>_<name>.down.sql` naming convention.

## Installing the CLI

```powershell
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
# make sure $GOPATH/bin (or %USERPROFILE%\go\bin on Windows) is on your PATH
```

## Applying migrations

```powershell
set DATABASE_URL=postgres://user:pass@localhost:54320/vinylhound?sslmode=disable
migrate -path migrations -database %DATABASE_URL% up
```

To roll back the most recent migration:

```powershell
migrate -path migrations -database %DATABASE_URL% down 1
```

Refer to the [migrate documentation](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) for additional commands and flags.
