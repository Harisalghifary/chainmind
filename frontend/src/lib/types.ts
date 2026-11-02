export type AnalysisDepth = "quick" | "standard" | "deep";

export type AuditStatus =
  | "PENDING"
  | "PARSING"
  | "ANALYZING"
  | "GENERATING_REPORT"
  | "COMPLETED"
  | "FAILED";

export type Severity =
  | "critical"
  | "high"
  | "medium"
  | "low"
  | "informational";

export interface CreateAuditRequest {
  contract_name: string;
  source_code: string;
  solidity_version?: string;
  analysis_depth?: AnalysisDepth;
}

export interface AuditCreated {
  audit_id: string;
  status: AuditStatus;
  estimated_seconds: number;
  created_at: string;
}

export interface AuditProgress {
  total_chunks: number;
  analyzed_chunks: number;
  current_stage: string;
}

export interface AuditStatusResponse {
  audit_id: string;
  status: AuditStatus;
  progress: AuditProgress;
  error_message: string | null;
  created_at: string;
  updated_at: string;
  completed_at: string | null;
}

export interface MatchedPattern {
  name: string;
  real_world_case: string;
  similarity_score: number;
}

export interface Finding {
  id: string;
  vulnerability_type: string;
  severity: Severity;
  confidence: number;
  affected_function: string;
  line_start: number;
  line_end: number;
  title: string;
  description: string;
  vulnerable_code: string;
  recommended_fix: string;
  fixed_code: string;
  matched_exploit_pattern?: MatchedPattern;
  references: string[];
}

export interface ReportSummary {
  total_findings: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  informational: number;
  confidence_score: number;
}

export interface ReportMetadata {
  llm_model: string;
  analysis_depth: string;
  rag_patterns_used: number;
  processing_time_ms: number;
  warnings?: string[];
}

export interface AuditReport {
  audit_id: string;
  contract_name: string;
  summary: ReportSummary;
  findings: Finding[];
  metadata: ReportMetadata;
}

export interface ExploitPattern {
  id: string;
  name: string;
  category: string;
  swc_id: string;
  severity: Severity;
  description: string;
  real_world_case: string;
}

export interface ExploitListResponse {
  patterns: ExploitPattern[];
  total: number;
  page: number;
  limit: number;
}