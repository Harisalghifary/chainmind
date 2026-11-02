ALTER TABLE audit_jobs
    ADD COLUMN rag_patterns_used INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN warnings TEXT[] NOT NULL DEFAULT '{}';