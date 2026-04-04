package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL,
    language VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    agents_config JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_ms INTEGER
);

CREATE TABLE IF NOT EXISTS agent_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    agent_name VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    findings JSONB DEFAULT '[]',
    raw_output TEXT DEFAULT '',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error TEXT,
    UNIQUE(review_id, agent_name)
);

CREATE INDEX IF NOT EXISTS idx_agent_results_review_id ON agent_results(review_id);
CREATE INDEX IF NOT EXISTS idx_reviews_created_at ON reviews(created_at DESC);
`

func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	if _, err := db.Exec(migrationSQL); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	log.Println("database migrations applied")
	return nil
}
