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

// buildConnectionString constructs a PostgreSQL connection string from environment variables.
// It prioritizes STATICREG_DB_URL if set, otherwise builds the connection string from individual
// STATICREG_DB_* environment variables. Returns empty string if no configuration is provided.
func buildConnectionString() string {
	// First, check if STATICREG_DB_URL is provided (highest priority)
	if connStr := os.Getenv("STATICREG_DB_URL"); connStr != "" {
		return connStr
	}

	// Build connection string from individual components
	host := os.Getenv("STATICREG_DB_HOST")
	port := os.Getenv("STATICREG_DB_PORT")
	user := os.Getenv("STATICREG_DB_USER")
	password := os.Getenv("STATICREG_DB_PASSWORD")
	dbname := os.Getenv("STATICREG_DB_NAME")
	sslmode := os.Getenv("STATICREG_DB_SSLMODE")

	// Require at minimum: host, user, and dbname
	if host == "" || user == "" || dbname == "" {
		return ""
	}

	// Build the connection string with required fields
	connStr := fmt.Sprintf("host=%s user=%s dbname=%s", host, user, dbname)

	// Add optional fields if provided
	if port != "" {
		connStr += fmt.Sprintf(" port=%s", port)
	}
	if password != "" {
		connStr += fmt.Sprintf(" password=%s", password)
	}
	if sslmode != "" {
		connStr += fmt.Sprintf(" sslmode=%s", sslmode)
	}

	return connStr
}

// InitPool attempts to initialize the PostgreSQL connection pool.
// It logs a warning if initialization fails and returns nil, allowing the app to continue.
// Returns the connection pool or nil if initialization failed.
func InitPool() *pgxpool.Pool {
	connStr := buildConnectionString()
	if connStr == "" {
		// Log warning and return if the environment variable is not set
		slog.Warn("Database configuration not provided. Set STATICREG_DB_URL or individual STATICREG_DB_* variables. Database functions will be disabled.")
		return nil
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		slog.Warn("Unable to parse database configuration: %v. Database functions will be disabled.", logger.ErrAttr(err))
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
	slog.Info("PostgreSQL connection pool successfully initialized.")

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
