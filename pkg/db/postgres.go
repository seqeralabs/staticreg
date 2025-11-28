package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	schemasql "github.com/seqeralabs/staticreg/pkg/sql"
)

// InitPool attempts to initialize the PostgreSQL connection pool.
// It logs a warning if initialization fails and returns nil, allowing the app to continue.
// Returns the connection pool or nil if initialization failed.
func InitPool() *pgxpool.Pool {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Log warning and return if the environment variable is not set
		slog.Warn("DATABASE_URL environment variable is not set. Database functions will be disabled.")
		return nil
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		slog.Warn("Unable to parse DATABASE_URL configuration: %v. Database functions will be disabled.", logger.ErrAttr(err))
		return nil
	}

	// Configure connection pool settings
	config.MaxConns = 25

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt to create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Warn("Unable to create connection pool: %v. Database functions will be disabled.", logger.ErrAttr(err))
		return nil
	}

	// Attempt to ping the database
	if err = pool.Ping(ctx); err != nil {
		slog.Warn("Database connection failed to ping: %v. Database functions will be disabled.", logger.ErrAttr(err))
		// Close the pool if ping failed
		pool.Close()
		return nil
	}

	// Success
	slog.Warn("PostgreSQL connection pool successfully initialized.")

	// Initialize database schema
	if err := initSchema(ctx, pool); err != nil {
		slog.Error("Failed to initialize database schema: %v. Exiting application.", logger.ErrAttr(err))
		pool.Close()
		os.Exit(1)
	}

	return pool
}

// initSchema creates the database schema if it doesn't exist
func initSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return nil
	}

	slog.Info("Initializing database schema...")

	// Execute the SQL schema directly (includes DROP TABLE IF EXISTS which is safe)
	_, err := pool.Exec(ctx, schemasql.EventSchemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Validate that the table was created successfully
	var tableExists bool
	err = pool.QueryRow(ctx, `
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
		err = pool.QueryRow(ctx, `
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

	slog.Info("Database schema initialized successfully.")
	return nil
}
