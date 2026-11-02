import type { Severity } from "@/lib/types";
import { severityClasses, severityLabel } from "@/lib/utils";

export function SeverityBadge({
  severity,
  className = "",
}: {
  severity: Severity | string;
  className?: string;
}) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-[0.14em] ${severityClasses(
        severity,
      )} ${className}`}
    >
      <span className="h-1 w-1 rounded-full bg-current" />
      {severityLabel(severity)}
    </span>
  );
}