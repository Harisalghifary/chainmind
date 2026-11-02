CREATE TYPE audit_status AS ENUM (
    'PENDING',
    'PARSING',
    'ANALYZING',
    'GENERATING_REPORT',
    'COMPLETED',
    'FAILED'
);

CREATE TABLE audit_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_name TEXT NOT NULL,
    source_code TEXT NOT NULL,
    solidity_version TEXT,
    analysis_depth TEXT NOT NULL DEFAULT 'standard' CHECK (
        analysis_depth IN ('quick', 'standard', 'deep')
    ),
    status audit_status NOT NULL DEFAULT 'PENDING',
    total_chunks INTEGER NOT NULL DEFAULT 0,
    analyzed_chunks INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    llm_model TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_audit_jobs_status
ON audit_jobs (status);

CREATE INDEX idx_audit_jobs_created_at
ON audit_jobs (created_at DESC);