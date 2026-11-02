import type { AuditStatus, Severity } from "./types";

export function severityClasses(severity: Severity | string): string {
  switch (severity) {
    case "critical":
      return "border-severity-critical/30 bg-severity-critical/10 text-severity-critical";
    case "high":
      return "border-severity-high/30 bg-severity-high/10 text-severity-high";
    case "medium":
      return "border-severity-medium/30 bg-severity-medium/10 text-severity-medium";
    case "low":
      return "border-severity-low/30 bg-severity-low/10 text-severity-low";
    case "informational":
      return "border-severity-informational/30 bg-severity-informational/10 text-severity-informational";
    default:
      return "border-line bg-ink-800 text-slate-400";
  }
}

export function severityLabel(severity: string): string {
  if (!severity) return "Unknown";
  return severity.charAt(0).toUpperCase() + severity.slice(1);
}

export function isTerminal(status: AuditStatus): boolean {
  return status === "COMPLETED" || status === "FAILED";
}

export function statusLabel(status: AuditStatus): string {
  switch (status) {
    case "PENDING":
      return "Queued";
    case "PARSING":
      return "Parsing";
    case "ANALYZING":
      return "Analyzing";
    case "GENERATING_REPORT":
      return "Generating report";
    case "COMPLETED":
      return "Completed";
    case "FAILED":
      return "Failed";
    default:
      return "Unknown";
  }
}

export function confidencePercent(value: number): string {
  return `${Math.round(value * 100)}%`;
}

export function formatDuration(milliseconds: number): string {
  if (!Number.isFinite(milliseconds) || milliseconds <= 0) return "—";
  if (milliseconds < 1000) return `${Math.round(milliseconds)} ms`;
  if (milliseconds < 60_000) return `${(milliseconds / 1000).toFixed(1)} s`;
  return `${(milliseconds / 60_000).toFixed(1)} min`;
}

export function formatTimestamp(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

export function shortId(id: string): string {
  return id.length > 8 ? id.slice(0, 8) : id;
}

export function lineRange(start: number, end: number): string {
  if (!start && !end) return "—";
  if (!start) return `L${end}`;
  if (!end || end === start) return `L${start}`;
  return `L${start}–${end}`;
}