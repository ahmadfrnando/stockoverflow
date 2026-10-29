package testutil

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set, skip open test db")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test db: %v", err)
	}
	const reset = `TRUNCATE products, stock_movements, sales_orders, sales_order_items RESTART IDENTITY CASCADE`

	if _, err := db.Exec(reset); err != nil {
		t.Fatalf("reset test db: %v", err)
	}

	return db

}