"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect } from "react";

import { AuditProgress } from "@/components/AuditProgress";
import { ReportViewer } from "@/components/ReportViewer";
import { formatTimestamp, shortId, statusLabel } from "@/lib/utils";
import { useAuditStore } from "@/store/useAuditStore";

export default function AuditPage() {
  const params = useParams<{ id: string }>();
  const auditId = params?.id;

  const status = useAuditStore((state) => state.status);
  const report = useAuditStore((state) => state.report);
  const isLoading = useAuditStore((state) => state.isLoading);
  const error = useAuditStore((state) => state.error);
  const startPolling = useAuditStore((state) => state.startPolling);
  const stopPolling = useAuditStore((state) => state.stopPolling);

  useEffect(() => {
    if (!auditId) return;
    startPolling(auditId);
    return () => stopPolling();
  }, [auditId, startPolling, stopPolling]);

  const title = report?.contract_name
    ? report.contract_name
    : status
      ? `Audit ${shortId(status.audit_id)}`
      : "Audit";

  return (
    <div className="relative min-h-screen overflow-x-hidden">
      <div
        aria-hidden="true"
        className="pointer-events-none fixed inset-0 -z-10 bg-grid-fade bg-grid"
      />

      <header className="sticky top-0 z-40 border-b border-line/60 bg-ink-950/75 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-6">
          <Link href="/" className="inline-flex items-center gap-2.5">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-signal/25 bg-signal/10">
              <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="h-4 w-4 text-signal">
                <path
                  d="M12 3 19 6v6c0 4.4-2.9 7.9-7 9-4.1-1.1-7-4.6-7-9V6l7-3Z"
                  stroke="currentColor"
                  strokeWidth="1.6"
                  strokeLinejoin="round"
                />
                <path
                  d="m8.8 11.8 2.2 2.2 4.2-4.4"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </span>
            <span className="text-[15px] font-semibold tracking-tight text-white">
              ChainMind
            </span>
          </Link>

          <Link
            href="/#console"
            className="rounded-lg border border-line bg-ink-900 px-3.5 py-1.5 text-sm text-slate-200 transition-colors hover:border-signal/40 hover:text-white"
          >
            New audit
          </Link>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 pb-24 pt-10">
        <Link
          href="/#console"
          className="inline-flex items-center gap-1.5 font-mono text-[11px] uppercase tracking-[0.16em] text-slate-500 transition-colors hover:text-signal"
        >
          <svg viewBox="0 0 16 16" fill="none" aria-hidden="true" className="h-3.5 w-3.5">
            <path
              d="M13 8H4m0 0 3.5-3.5M4 8l3.5 3.5"
              stroke="currentColor"
              strokeWidth="1.4"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          Audit console
        </Link>

        <div className="mt-5 flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="mono-label">
              {status?.status === "COMPLETED" ? "Audit result" : "Live audit"}
            </p>
            <h1 className="mt-1 text-3xl font-semibold tracking-tight text-white">
              {title}
            </h1>
            {status ? (
              <p className="mt-1.5 font-mono text-[11px] text-slate-600">
                started {formatTimestamp(status.created_at)}
              </p>
            ) : null}
          </div>
          {status ? (
            <span className="rounded-full border border-line bg-ink-900 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.14em] text-slate-400">
              {statusLabel(status.status)}
            </span>
          ) : null}
        </div>

        <div className="mt-8 space-y-6">
          {error ? (
            <section className="panel p-6">
              <p className="text-sm text-severity-critical">{error}</p>
              <div className="mt-4 flex flex-wrap items-center gap-3">
                <button
                  type="button"
                  onClick={() => auditId && startPolling(auditId)}
                  className="rounded-lg border border-line bg-ink-850 px-3.5 py-2 text-sm text-slate-200 transition-colors hover:border-signal/40 hover:text-white"
                >
                  Retry
                </button>
                <Link
                  href="/#console"
                  className="text-sm text-slate-500 transition-colors hover:text-signal"
                >
                  Start a new audit
                </Link>
              </div>
            </section>
          ) : null}

          {!error && !status && (isLoading || auditId) ? (
            <>
              <div className="panel animate-pulse p-6">
                <div className="h-4 w-32 rounded bg-ink-700" />
                <div className="mt-4 h-2 w-full rounded bg-ink-800" />
                <div className="mt-6 grid grid-cols-2 gap-2 sm:grid-cols-4">
                  {[0, 1, 2, 3].map((item) => (
                    <div key={item} className="h-9 rounded-lg bg-ink-800" />
                  ))}
                </div>
              </div>
              <p className="text-center font-mono text-[11px] uppercase tracking-[0.16em] text-slate-600">
                Loading audit status…
              </p>
            </>
          ) : null}

          {status ? <AuditProgress status={status} /> : null}

          {status?.status === "COMPLETED" && !report ? (
            <div className="panel animate-pulse p-6">
              <div className="h-4 w-40 rounded bg-ink-700" />
              <div className="mt-4 h-24 rounded bg-ink-800" />
            </div>
          ) : null}

          {status?.status === "FAILED" ? (
            <section className="panel p-6">
              <p className="text-sm font-medium text-white">
                This audit did not produce a report.
              </p>
              <p className="mt-1.5 text-sm leading-6 text-slate-500">
                Fix the provider configuration or submit the contract again.
                Failed audits keep their error message above.
              </p>
              <Link
                href="/#console"
                className="mt-4 inline-flex items-center gap-2 rounded-lg bg-signal px-4 py-2 text-sm font-medium text-ink-950 transition-colors hover:bg-signal-soft"
              >
                Start a new audit
              </Link>
            </section>
          ) : null}

          {report ? <ReportViewer report={report} /> : null}
        </div>
      </main>
    </div>
  );
}