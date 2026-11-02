CREATE TABLE findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_id UUID NOT NULL REFERENCES audit_jobs(id) ON DELETE CASCADE,
    vulnerability_type TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (
        severity IN ('critical', 'high', 'medium', 'low', 'informational')
    ),
    confidence REAL NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    affected_function TEXT NOT NULL,
    line_start INTEGER,
    line_end INTEGER,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    vulnerable_code TEXT,
    recommended_fix TEXT,
    fixed_code TEXT,
    matched_pattern_id UUID REFERENCES exploit_patterns(id),
    matched_pattern_name TEXT,
    similarity_score REAL,
    "references" TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_findings_audit_id
ON findings (audit_id);

CREATE INDEX idx_findings_severity
ON findings (severity);