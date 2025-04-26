package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

// InitDB menginisialisasi koneksi database PostgreSQL
func InitDB(connString string) error {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("error parsing connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("error creating connection pool: %w", err)
	}

	// Test connection
	err = pool.Ping(context.Background())
	if err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	DB = pool
	fmt.Println("Successfully connected to database!")
	return nil
}

// CloseDB menutup koneksi database
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}