import type { AuditStatusResponse } from "@/lib/types";
import { formatTimestamp, shortId, statusLabel } from "@/lib/utils";

const stages = [
  { key: "PARSING", label: "Parse" },
  { key: "ANALYZING", label: "Analyze" },
  { key: "GENERATING_REPORT", label: "Report" },
  { key: "COMPLETED", label: "Done" },
];

function stageIndex(status: AuditStatusResponse["status"]): number {
  switch (status) {
    case "PARSING":
      return 0;
    case "ANALYZING":
      return 1;
    case "GENERATING_REPORT":
      return 2;
    case "COMPLETED":
      return 3;
    default:
      return -1;
  }
}

export function AuditProgress({ status }: { status: AuditStatusResponse }) {
  const currentStage = stageIndex(status.status);
  const failed = status.status === "FAILED";
  const completed = status.status === "COMPLETED";
  const running = !failed && !completed;

  const total = status.progress.total_chunks;
  const analyzed = status.progress.analyzed_chunks;
  const percent = completed
    ? 100
    : total > 0
      ? Math.min(100, Math.round((analyzed / total) * 100))
      : 0;

  return (
    <section className="panel overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line/70 bg-ink-850/60 px-5 py-3">
        <div className="flex items-center gap-2.5">
          <span
            className={`h-1.5 w-1.5 rounded-full ${
              failed
                ? "bg-severity-critical"
                : completed
                  ? "bg-signal"
                  : "animate-pulse bg-signal"
            }`}
          />
          <span className="font-mono text-[11px] uppercase tracking-[0.16em] text-slate-400">
            {statusLabel(status.status)}
          </span>
        </div>
        <div className="flex items-center gap-4 font-mono text-[10px] text-slate-600">
          <span>audit {shortId(status.audit_id)}</span>
          <span className="hidden sm:inline">
            updated {formatTimestamp(status.updated_at)}
          </span>
        </div>
      </div>

      <div className="p-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="text-lg font-medium text-white">
              {failed
                ? "Audit failed"
                : completed
                  ? "Report ready"
                  : running
                    ? "Analyzing contract"
                    : "Audit queued"}
            </h2>
            <p className="mt-1 text-sm text-slate-500">
              {status.status === "ANALYZING" && total > 0
                ? `${analyzed} of ${total} functions analyzed`
                : failed
                  ? "The pipeline stopped before generating a report."
                  : completed
                    ? `Processed ${total} function chunk${total === 1 ? "" : "s"}.`
                    : "Waiting for the analysis worker."}
            </p>
          </div>
          <span className="font-mono text-[11px] text-slate-500">
            {percent}%
          </span>
        </div>

        <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-ink-700">
          <div
            className={`h-full rounded-full transition-all duration-700 ${
              failed ? "bg-severity-critical" : "bg-signal"
            }`}
            style={{ width: `${percent}%` }}
          />
        </div>

        <ol className="mt-5 grid grid-cols-2 gap-2 sm:grid-cols-4">
          {stages.map((stage, index) => {
            const done = completed || (!failed && index < currentStage);
            const active =
              running && index === Math.max(0, currentStage) && !done;

            return (
              <li
                key={stage.key}
                className={`flex items-center gap-2 rounded-lg border px-3 py-2 ${
                  done
                    ? "border-signal/25 bg-signal/[0.06] text-signal"
                    : active
                      ? "border-signal/40 bg-signal/10 text-signal"
                      : "border-line/70 bg-ink-950/40 text-slate-600"
                }`}
              >
                <span
                  className={`flex h-4 w-4 items-center justify-center rounded-full border font-mono text-[9px] ${
                    done
                      ? "border-signal/40 text-signal"
                      : active
                        ? "border-signal/50 text-signal"
                        : "border-line text-slate-600"
                  }`}
                >
                  {done ? "✓" : index + 1}
                </span>
                <span className="font-mono text-[10px] uppercase tracking-[0.14em]">
                  {stage.label}
                </span>
                {active ? (
                  <span className="ml-auto h-1 w-1 animate-pulse rounded-full bg-signal" />
                ) : null}
              </li>
            );
          })}
        </ol>

        {failed && status.error_message ? (
          <p className="mt-4 rounded-lg border border-severity-critical/30 bg-severity-critical/10 px-3 py-2 text-xs leading-5 text-severity-critical">
            {status.error_message}
          </p>
        ) : null}
      </div>
    </section>
  );
}