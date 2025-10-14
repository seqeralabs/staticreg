package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the global, exported database connection pool instance.
// It will be nil if the connection failed.
var Pool *pgxpool.Pool

// InitPool attempts to initialize the PostgreSQL connection pool.
// It logs a warning if initialization fails and sets Pool to nil, allowing the app to continue.
func InitPool() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Log warning and return if the environment variable is not set
		log.Println("WARNING: DATABASE_URL environment variable is not set. Database functions will be disabled.")
		Pool = nil
		return
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Printf("WARNING: Unable to parse DATABASE_URL configuration: %v. Database functions will be disabled.", err)
		Pool = nil
		return
	}

	// Configure connection pool settings
	config.MaxConns = 25

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt to create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Printf("WARNING: Unable to create connection pool: %v. Database functions will be disabled.", err)
		Pool = nil
		return
	}

	// Attempt to ping the database
	if err = pool.Ping(ctx); err != nil {
		log.Printf("WARNING: Database connection failed to ping: %v. Database functions will be disabled.", err)
		// Close the temporary pool if ping failed, before setting the global Pool to nil
		pool.Close()
		Pool = nil
		return
	}

	// Success
	Pool = pool
	log.Println("PostgreSQL connection pool successfully initialized.")
}

// ClosePool closes the database connection pool if it was initialized.
func ClosePool() {
	if Pool != nil {
		Pool.Close()
		log.Println("PostgreSQL connection pool closed.")
	}
}
