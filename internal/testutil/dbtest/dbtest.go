package dbtest

import (
	"database/sql"
	"testing"

	"github.com/Vixel2006/panoptes/internal/infra/db"

	_ "modernc.org/sqlite"
)

func Open(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}

	if err := db.Migrate(database); err != nil {
		t.Fatal(err)
	}

	return database
}
