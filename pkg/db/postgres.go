package db

import (
	"context"
	"fmt"
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

	// Initialize database schema
	if err := InitSchema(ctx); err != nil {
		log.Printf("WARNING: Failed to initialize database schema: %v. Some features may not work.", err)
	}
}

// InitSchema creates the database schema if it doesn't exist
func InitSchema(ctx context.Context) error {
	if Pool == nil {
		return nil
	}

	// SQL schema for container pull metrics
	schema := `
-- Create container pull metrics table if not exists
CREATE TABLE IF NOT EXISTS container_pull_metrics (
    id BIGSERIAL PRIMARY KEY,
    pull_date DATE NOT NULL,
    repo_name TEXT NOT NULL,
    tag TEXT NOT NULL,
    digest TEXT NOT NULL,
    architecture TEXT NOT NULL,
    pull_count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(pull_date, repo_name, tag, digest, architecture)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_pull_date ON container_pull_metrics (pull_date DESC);
CREATE INDEX IF NOT EXISTS idx_repo_date ON container_pull_metrics (repo_name, pull_date DESC);
CREATE INDEX IF NOT EXISTS idx_repo_arch_date ON container_pull_metrics (repo_name, architecture, pull_date DESC);
`

	_, err := Pool.Exec(ctx, schema)
	if err != nil {
		return err
	}

	// Validate that the table was created successfully
	var tableExists bool
	err = Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'container_pull_metrics'
		)`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to verify table creation: %w", err)
	}
	if !tableExists {
		return fmt.Errorf("table container_pull_metrics was not created")
	}

	// Validate that indexes were created successfully
	expectedIndexes := []string{"idx_pull_date", "idx_repo_date", "idx_repo_arch_date"}
	for _, indexName := range expectedIndexes {
		var indexExists bool
		err = Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM pg_indexes
				WHERE schemaname = 'public'
				AND tablename = 'container_pull_metrics'
				AND indexname = $1
			)`, indexName).Scan(&indexExists)
		if err != nil {
			return fmt.Errorf("failed to verify index %s: %w", indexName, err)
		}
		if !indexExists {
			return fmt.Errorf("index %s was not created", indexName)
		}
	}

	log.Println("Database schema initialized successfully.")
	return nil
}

// ClosePool closes the database connection pool if it was initialized.
func ClosePool() {
	if Pool != nil {
		Pool.Close()
		log.Println("PostgreSQL connection pool closed.")
	}
}
