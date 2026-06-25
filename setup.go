//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Println("error conectando:", err)
		os.Exit(1)
	}
	defer pool.Close()

	_, err = pool.Exec(context.Background(), `
CREATE TABLE IF NOT EXISTS recommended_books (
    book_id          UUID PRIMARY KEY,
    google_volume_id VARCHAR(64) NOT NULL,
    title            TEXT NOT NULL,
    authors          TEXT[] NOT NULL DEFAULT '{}',
    thumbnail        TEXT,
    info_link        TEXT,
    description      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`)
	if err != nil {
		fmt.Println("error creando tabla:", err)
		os.Exit(1)
	}
	fmt.Println("tabla recommended_books creada (o ya existía)")
}
