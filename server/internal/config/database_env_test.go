package config

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestComposeDatabaseCredentialsUseEnvironmentWithoutURLInterpolation(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db:5432/?sslmode=disable")
	t.Setenv("PGUSER", `new"; SELECT 1; -- role`)
	t.Setenv("PGPASSWORD", `quoted'\:@/?#%-$-password`)
	t.Setenv("PGDATABASE", "host=unreachable dbname='app'")
	t.Setenv("PGSERVICE", "")
	connection, err := pgx.ParseConfig(Load().DatabaseURL)
	if err != nil {
		t.Fatal("failed to parse Compose database configuration")
	}
	if connection.Host != "db" || connection.Port != 5432 || connection.User != `new"; SELECT 1; -- role` ||
		connection.Password != `quoted'\:@/?#%-$-password` || connection.Database != "host=unreachable dbname='app'" {
		t.Fatal("Compose connection did not preserve the literal credentials and database name")
	}
}
