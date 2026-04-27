package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/seqeralabs/staticreg/pkg/observability/logger"
	schemasql "github.com/seqeralabs/staticreg/pkg/sql"
)

// defaultSchema is the postgres schema used when STATICREG_DB_SCHEMA is unset.
const defaultSchema = "staticreg"

// schemaIdentRegexp validates that a schema name is a safe postgres identifier.
// Only lowercase letters, digits, and underscores; must not start with a digit.
// Identifier is interpolated into DDL (CREATE SCHEMA, search_path) since pgx
// parameters cannot bind identifiers — strict validation prevents injection.
var schemaIdentRegexp = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// resolveSchema returns the validated schema name from STATICREG_DB_SCHEMA or
// the default. Returns an error if the env var is set but invalid.
func resolveSchema() (string, error) {
	schema := os.Getenv("STATICREG_DB_SCHEMA")
	if schema == "" {
		return defaultSchema, nil
	}
	if len(schema) > 63 || !schemaIdentRegexp.MatchString(schema) {
		return "", fmt.Errorf("invalid STATICREG_DB_SCHEMA %q: must match %s and be <=63 chars", schema, schemaIdentRegexp)
	}
	return schema, nil
}

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

	schema, err := resolveSchema()
	if err != nil {
		slog.Warn("Invalid database schema configuration. Database functions will be disabled.", logger.ErrAttr(err))
		return nil
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		slog.Warn("Unable to parse database configuration. Database functions will be disabled.", logger.ErrAttr(err))
		return nil
	}

	// Pin every pooled connection to the dedicated schema so unqualified
	// identifiers in queries (and goose's bookkeeping table) resolve there.
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema

	// Configure connection pool settings
	config.MaxConns = 25

	// Short timeout for pool construction + initial ping. A separate, longer
	// budget is used for migrations below — running goose under the same
	// 10s deadline can produce spurious "context deadline exceeded" failures
	// at startup even when the database is healthy.
	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer poolCancel()

	pool, err := pgxpool.NewWithConfig(poolCtx, config)
	if err != nil {
		slog.Warn("Unable to create connection pool. Database functions will be disabled.", logger.ErrAttr(err))
		return nil
	}

	if err = pool.Ping(poolCtx); err != nil {
		slog.Warn("Database connection failed to ping. Database functions will be disabled.", logger.ErrAttr(err))
		pool.Close()
		return nil
	}

	slog.Info("PostgreSQL connection pool successfully initialized.", slog.String("schema", schema))

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), time.Minute)
	defer migrationCancel()
	if err := initSchema(migrationCtx, pool, schema); err != nil {
		slog.Error("Failed to initialize database schema. Exiting application.", logger.ErrAttr(err))
		pool.Close()
		os.Exit(1)
	}

	return pool
}

// initSchema ensures the dedicated schema exists and runs all pending migrations.
// Schema name has already been validated by resolveSchema before this is called.
func initSchema(ctx context.Context, pool *pgxpool.Pool, schema string) error {
	if pool == nil {
		return nil
	}

	// Schema name was validated against schemaIdentRegexp, so direct interpolation is safe.
	// Goose's bookkeeping table (goose_db_version) lands inside this schema because the
	// pool's search_path was set above, before the first connection was acquired.
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		return fmt.Errorf("failed to create schema %q: %w", schema, err)
	}

	// stdlib.OpenDBFromPool returns a *sql.DB that wraps the pool; closing it
	// does not close the pool (per pgx docs).
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetBaseFS(schemasql.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}
	goose.SetLogger(goose.NopLogger())

	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	slog.Info("Database schema ready.", slog.String("schema", schema))
	return nil
}
