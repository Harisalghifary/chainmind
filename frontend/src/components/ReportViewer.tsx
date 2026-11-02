import { FindingCard } from "@/components/FindingCard";
import { SeverityBadge } from "@/components/SeverityBadge";
import type { AuditReport } from "@/lib/types";
import { confidencePercent, formatDuration } from "@/lib/utils";

const summaryTiles = [
  { key: "critical", label: "Critical", tone: "text-severity-critical" },
  { key: "high", label: "High", tone: "text-severity-high" },
  { key: "medium", label: "Medium", tone: "text-severity-medium" },
  { key: "low", label: "Low", tone: "text-severity-low" },
  {
    key: "informational",
    label: "Info",
    tone: "text-severity-informational",
  },
] as const;

export function ReportViewer({ report }: { report: AuditReport }) {
  const { summary, metadata } = report;

  return (
    <div className="space-y-6">
      <section className="panel overflow-hidden">
        <div className="border-b border-line/70 bg-ink-850/60 px-5 py-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="font-mono text-[11px] uppercase tracking-[0.16em] text-slate-400">
              Audit report
            </span>
            <span className="font-mono text-[10px] text-slate-600">
              {report.audit_id}
            </span>
          </div>
        </div>

        <div className="p-5 sm:p-6">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="mono-label">Contract</p>
              <h2 className="mt-1 text-2xl font-semibold tracking-tight text-white">
                {report.contract_name}
              </h2>
            </div>
            <div className="text-right">
              <p className="mono-label">Confidence</p>
              <p className="mt-1 font-mono text-2xl text-signal">
                {confidencePercent(summary.confidence_score)}
              </p>
            </div>
          </div>

          <div className="mt-6 grid grid-cols-3 gap-px overflow-hidden rounded-xl border border-line bg-line sm:grid-cols-6">
            <div className="bg-ink-950/90 p-4">
              <p className="mono-label">Total</p>
              <p className="mt-1 font-mono text-xl text-white">
                {summary.total_findings}
              </p>
            </div>
            {summaryTiles.map((tile) => (
              <div key={tile.key} className="bg-ink-950/90 p-4">
                <p className="mono-label">{tile.label}</p>
                <p className={`mt-1 font-mono text-xl ${tile.tone}`}>
                  {summary[tile.key]}
                </p>
              </div>
            ))}
          </div>

          <div className="mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-line/60 pt-4 font-mono text-[10px] text-slate-500">
            <span>model {metadata.llm_model || "unknown"}</span>
            <span className="h-1 w-1 rounded-full bg-line" />
            <span>depth {metadata.analysis_depth}</span>
            <span className="h-1 w-1 rounded-full bg-line" />
            <span>{metadata.rag_patterns_used} exploit patterns</span>
            <span className="h-1 w-1 rounded-full bg-line" />
            <span>{formatDuration(metadata.processing_time_ms)}</span>
          </div>

          {metadata.warnings && metadata.warnings.length > 0 ? (
            <div className="mt-4 rounded-lg border border-severity-medium/25 bg-severity-medium/[0.06] px-4 py-3">
              <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-severity-medium">
                Degraded run
              </p>
              <ul className="mt-1.5 space-y-1 text-xs leading-5 text-slate-400">
                {metadata.warnings.map((warning) => (
                  <li key={warning}>{warning}</li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>
      </section>

      <section>
        <div className="flex items-end justify-between gap-4">
          <div>
            <p className="mono-label">Findings</p>
            <h2 className="mt-1 text-xl font-semibold tracking-tight text-white">
              {summary.total_findings === 0
                ? "No vulnerabilities found"
                : `${summary.total_findings} finding${
                    summary.total_findings === 1 ? "" : "s"
                  }`}
            </h2>
          </div>
          {summary.total_findings > 0 ? (
            <SeverityBadge
              severity={
                summary.critical > 0
                  ? "critical"
                  : summary.high > 0
                    ? "high"
                    : "medium"
              }
              className="hidden sm:inline-flex"
            />
          ) : null}
        </div>

        {report.findings.length === 0 ? (
          <div className="panel mt-4 p-8 text-center">
            <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full border border-signal/30 bg-signal/10">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                aria-hidden="true"
                className="h-5 w-5 text-signal"
              >
                <path
                  d="m7.5 12.5 3 3 6-6.5"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </div>
            <p className="mt-4 text-sm font-medium text-white">
              No issues were reported for this contract.
            </p>
            <p className="mx-auto mt-1.5 max-w-md text-xs leading-5 text-slate-500">
              The model was conservative and only reported vulnerabilities
              visible in the submitted source.
            </p>
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            {report.findings.map((finding) => (
              <FindingCard key={finding.id} finding={finding} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}