package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	schemasql "github.com/seqeralabs/staticreg/pkg/sql"
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
		slog.Warn("DATABASE_URL environment variable is not set. Database functions will be disabled.")
		Pool = nil
		return
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		slog.Warn("Unable to parse DATABASE_URL configuration: %v. Database functions will be disabled.", err)
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
		slog.Warn("Unable to create connection pool: %v. Database functions will be disabled.", err)
		Pool = nil
		return
	}

	// Attempt to ping the database
	if err = pool.Ping(ctx); err != nil {
		slog.Warn("Database connection failed to ping: %v. Database functions will be disabled.", err)
		// Close the temporary pool if ping failed, before setting the global Pool to nil
		pool.Close()
		Pool = nil
		return
	}

	// Success
	Pool = pool
	slog.Warn("PostgreSQL connection pool successfully initialized.")

	// Initialize database schema
	if err := InitSchema(ctx); err != nil {
		slog.Error("Failed to initialize database schema: %v. Exiting application.", err)
		os.Exit(1)
	}
}

// InitSchema creates the database schema if it doesn't exist
func InitSchema(ctx context.Context) error {
	if Pool == nil {
		return nil
	}

	// Load SQL schema from embedded file
	// Filter out DROP TABLE statements since we don't want to drop existing data during initialization
	schemaLines := strings.Split(schemasql.EventSchemaSQL, "\n")
	var filteredLines []string
	for _, line := range schemaLines {
		trimmed := strings.TrimSpace(line)
		// Skip DROP TABLE statements
		if strings.HasPrefix(strings.ToUpper(trimmed), "DROP TABLE") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	schema := strings.Join(filteredLines, "\n")

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

	slog.Info("Database schema initialized successfully.")
	return nil
}

// ClosePool closes the database connection pool if it was initialized.
func ClosePool() {
	if Pool != nil {
		Pool.Close()
		slog.Info("PostgreSQL connection pool closed.")
	}
}
