import ReactMarkdown from "react-markdown";

import { SeverityBadge } from "@/components/SeverityBadge";
import type { Finding } from "@/lib/types";
import { confidencePercent, lineRange } from "@/lib/utils";

const accents: Record<string, string> = {
  critical: "border-l-severity-critical",
  high: "border-l-severity-high",
  medium: "border-l-severity-medium",
  low: "border-l-severity-low",
  informational: "border-l-severity-informational",
};

export function FindingCard({ finding }: { finding: Finding }) {
  const accent = accents[finding.severity] ?? "border-l-line";

  return (
    <article
      className={`panel overflow-hidden border-l-2 ${accent}`}
    >
      <div className="p-5 sm:p-6">
        <div className="flex flex-wrap items-center gap-2.5">
          <SeverityBadge severity={finding.severity} />
          <span className="rounded border border-line bg-ink-850 px-2 py-0.5 font-mono text-[10px] lowercase tracking-wide text-slate-400">
            {finding.vulnerability_type}
          </span>
          <span className="ml-auto font-mono text-[10px] uppercase tracking-[0.14em] text-slate-500">
            confidence {confidencePercent(finding.confidence)}
          </span>
        </div>

        <h3 className="mt-4 text-lg font-medium leading-6 text-white">
          {finding.title}
        </h3>
        <p className="mt-1.5 font-mono text-[11px] text-slate-500">
          {finding.affected_function}() ·{" "}
          {lineRange(finding.line_start, finding.line_end)}
        </p>

        <div className="markdown mt-4">
          <ReactMarkdown>{finding.description}</ReactMarkdown>
        </div>

        {finding.vulnerable_code ? (
          <div className="mt-5 overflow-hidden rounded-lg border border-line/80 bg-ink-950">
            <div className="flex items-center justify-between border-b border-line/60 px-3 py-2">
              <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-severity-critical/80">
                Vulnerable code
              </span>
              <span className="font-mono text-[10px] text-slate-600">
                {finding.affected_function}
              </span>
            </div>
            <pre className="code-scroll overflow-x-auto p-4 font-mono text-[12px] leading-6 text-slate-300">
              <code>{finding.vulnerable_code}</code>
            </pre>
          </div>
        ) : null}

        {finding.recommended_fix ? (
          <div className="mt-4 rounded-lg border border-signal/20 bg-signal/[0.04] p-4">
            <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-signal/80">
              Recommended fix
            </span>
            <div className="markdown mt-2">
              <ReactMarkdown>{finding.recommended_fix}</ReactMarkdown>
            </div>
            {finding.fixed_code ? (
              <pre className="code-scroll mt-3 overflow-x-auto rounded-md border border-signal/15 bg-ink-950/70 p-3 font-mono text-[12px] leading-6 text-signal-soft">
                <code>{finding.fixed_code}</code>
              </pre>
            ) : null}
          </div>
        ) : null}

        {finding.matched_exploit_pattern ? (
          <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border border-line/70 bg-ink-850/60 px-4 py-3">
            <svg
              viewBox="0 0 16 16"
              fill="none"
              aria-hidden="true"
              className="h-4 w-4 shrink-0 text-signal/70"
            >
              <path
                d="M8 1.5 14 4v4.2c0 3.1-2.4 5.7-6 6.3-3.6-.6-6-3.2-6-6.3V4l6-2.5Z"
                stroke="currentColor"
                strokeWidth="1.2"
                strokeLinejoin="round"
              />
            </svg>
            <div className="min-w-0">
              <p className="truncate text-sm text-slate-200">
                {finding.matched_exploit_pattern.name}
              </p>
              <p className="mt-0.5 text-xs text-slate-500">
                {finding.matched_exploit_pattern.real_world_case ||
                  "Known exploit pattern"}
              </p>
            </div>
            <span className="ml-auto shrink-0 rounded-full border border-signal/25 bg-signal/10 px-2 py-0.5 font-mono text-[10px] text-signal">
              {Math.round(
                finding.matched_exploit_pattern.similarity_score * 100,
              )}
              % match
            </span>
          </div>
        ) : null}

        {finding.references.length > 0 ? (
          <div className="mt-4 flex flex-wrap gap-2">
            {finding.references.map((reference) => (
              <a
                key={reference}
                href={reference}
                target="_blank"
                rel="noreferrer"
                className="max-w-full truncate rounded border border-line bg-ink-850/60 px-2.5 py-1 font-mono text-[10px] text-slate-400 transition-colors hover:border-signal/30 hover:text-signal"
              >
                {reference.replace(/^https?:\/\//, "")}
              </a>
            ))}
          </div>
        ) : null}
      </div>
    </article>
  );
}