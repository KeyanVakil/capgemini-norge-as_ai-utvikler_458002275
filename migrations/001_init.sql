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
