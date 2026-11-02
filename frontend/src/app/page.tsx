import Link from "next/link";

import { ContractInput } from "@/components/ContractInput";

const pipeline = [
  {
    step: "01",
    title: "Parse",
    detail:
      "Extract the contract name, functions, state variables, imports and inheritance from Solidity source.",
  },
  {
    step: "02",
    title: "Chunk",
    detail:
      "Split the contract at function boundaries and attach a contract-level context summary to every chunk.",
  },
  {
    step: "03",
    title: "Retrieve",
    detail:
      "Embed each chunk and pull the closest real-world exploit patterns from a pgvector index.",
  },
  {
    step: "04",
    title: "Analyze",
    detail:
      "Claude or GPT audits each chunk against the retrieved evidence and answers with strict JSON.",
  },
  {
    step: "05",
    title: "Report",
    detail:
      "Deduplicate, score and persist findings into a structured report you can poll and render.",
  },
];

const vulnerabilityClasses = [
  {
    tag: "RE",
    name: "Reentrancy",
    severity: "critical",
    detail:
      "External calls that run before the contract updates its own state, enabling recursive withdrawals.",
    swc: "SWC-107",
  },
  {
    tag: "OF",
    name: "Integer overflow / underflow",
    severity: "high",
    detail:
      "Arithmetic that wraps around and silently corrupts balances, supply counters or accounting.",
    swc: "SWC-101",
  },
  {
    tag: "AC",
    name: "Access control",
    severity: "critical",
    detail:
      "Privileged functions such as mint, withdraw or selfdestruct that anyone can call.",
    swc: "SWC-105",
  },
  {
    tag: "FL",
    name: "Flash loan vector",
    severity: "high",
    detail:
      "Single-transaction liquidity that can distort spot prices or drain undercollateralized pools.",
    swc: "DeFi",
  },
  {
    tag: "OR",
    name: "Oracle manipulation",
    severity: "high",
    detail:
      "Pricing logic that trusts one feed, a spot pool or a stale answer without sanity checks.",
    swc: "SWC-116",
  },
  {
    tag: "UC",
    name: "Unchecked external call",
    severity: "medium",
    detail:
      "Return values from low-level calls that are ignored, letting failed transfers look successful.",
    swc: "SWC-104",
  },
];

const severityStyles: Record<string, string> = {
  critical:
    "border-severity-critical/30 bg-severity-critical/10 text-severity-critical",
  high: "border-severity-high/30 bg-severity-high/10 text-severity-high",
  medium: "border-severity-medium/30 bg-severity-medium/10 text-severity-medium",
};

function SeverityChip({ severity }: { severity: string }) {
  const style = severityStyles[severity] ?? severityStyles.medium;
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-[0.14em] ${style}`}
    >
      <span className="h-1 w-1 rounded-full bg-current" />
      {severity}
    </span>
  );
}

function Logo() {
  return (
    <Link href="/" className="group inline-flex items-center gap-2.5">
      <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-signal/25 bg-signal/10">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          aria-hidden="true"
          className="h-4.5 w-4.5 text-signal"
        >
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
  );
}

function CodeWindow() {
  const plainLine = (number: number, code: string) => (
    <span className="block">
      <span className="inline-block w-7 select-none text-slate-600">{number}</span>
      <span className="text-slate-300">{code}</span>
    </span>
  );

  return (
    <div className="relative mb-0 min-w-0 sm:mb-10">
      <div
        aria-hidden="true"
        className="absolute -inset-6 rounded-[2rem] bg-signal/[0.06] blur-2xl"
      />

      <div className="panel relative overflow-hidden shadow-2xl shadow-black/60">
        <div className="flex items-center justify-between border-b border-line/70 bg-ink-850/80 px-4 py-3">
          <div className="flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-full bg-[#F87171]/70" />
            <span className="h-2.5 w-2.5 rounded-full bg-[#FBBF24]/70" />
            <span className="h-2.5 w-2.5 rounded-full bg-[#34D399]/70" />
          </div>
          <span className="font-mono text-[11px] text-slate-500">
            VulnerableVault.sol
          </span>
          <span className="rounded border border-line px-1.5 py-0.5 font-mono text-[10px] text-slate-500">
            ^0.8.0
          </span>
        </div>

        <div className="code-scroll overflow-x-auto">
          <pre className="min-w-[30rem] px-4 py-4 font-mono text-[12.5px] leading-6">
            <code>
              {plainLine(1, "pragma solidity ^0.8.0;")}
              <span className="block h-3" />
              {plainLine(3, "contract VulnerableVault {")}
              {plainLine(
                4,
                "    mapping(address => uint256) public balances;",
              )}
              <span className="block h-3" />
              {plainLine(6, "    function withdraw() external {")}
              {plainLine(
                7,
                "        uint256 amount = balances[msg.sender];",
              )}
              <span className="block h-3" />
              <span className="my-0.5 -mx-4 block border-y border-severity-critical/20 bg-severity-critical/[0.07] px-4 py-0.5">
                <span className="inline-block w-7 select-none text-severity-critical/70">
                  9
                </span>
                <span className="text-slate-200">
                  {'        (bool success, ) = msg.sender.call{value: amount}("");'}
                </span>
              </span>
              {plainLine(10, '        require(success, "Transfer failed");')}
              <span className="block h-3" />
              <span className="my-0.5 -mx-4 block border-y border-signal/20 bg-signal/[0.06] px-4 py-0.5">
                <span className="inline-block w-7 select-none text-signal/70">
                  12
                </span>
                <span className="text-slate-200">
                  {"        balances[msg.sender] = 0;"}
                </span>
              </span>
              {plainLine(13, "    }")}
              {plainLine(14, "}")}
            </code>
          </pre>
        </div>

        <div className="flex items-center justify-between border-t border-line/70 bg-ink-850/60 px-4 py-2.5">
          <span className="font-mono text-[11px] text-slate-500">
            withdraw()
          </span>
          <span className="font-mono text-[11px] text-severity-critical/80">
            external call before state update
          </span>
        </div>
      </div>

      <div className="panel mt-3 w-full border-severity-critical/30 bg-ink-900/95 p-4 shadow-xl shadow-black/50 sm:absolute sm:-bottom-8 sm:right-0 sm:mt-0 sm:w-[17.5rem] sm:-right-6">
        <div className="flex items-center justify-between">
          <SeverityChip severity="critical" />
          <span className="font-mono text-[10px] text-slate-500">SWC-107</span>
        </div>
        <p className="mt-3 text-sm font-medium text-white">
          Reentrancy in withdraw()
        </p>
        <p className="mt-1 text-xs leading-5 text-slate-400">
          Balance reset happens after the external call.
        </p>
        <div className="mt-3 flex items-center justify-between border-t border-line/70 pt-2.5 font-mono text-[10px] text-slate-500">
          <span>confidence 0.95</span>
          <span className="text-signal/80">matched exploit pattern</span>
        </div>
      </div>
    </div>
  );
}

export default function HomePage() {
  return (
    <div className="relative min-h-screen overflow-x-hidden">
      <div
        aria-hidden="true"
        className="pointer-events-none fixed inset-0 -z-10 bg-grid-fade bg-grid"
      />

      <header className="sticky top-0 z-40 border-b border-line/60 bg-ink-950/70 backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
          <div className="flex items-center gap-8">
            <Logo />
            <nav className="hidden items-center gap-6 md:flex">
              <a href="#console" className="nav-link">
                Console
              </a>
              <a href="#pipeline" className="nav-link">
                Pipeline
              </a>
              <a href="#coverage" className="nav-link">
                Coverage
              </a>
            </nav>
          </div>

          <div className="flex items-center gap-3">
            <span className="hidden items-center gap-2 rounded-full border border-line bg-ink-900 px-3 py-1 font-mono text-[10px] uppercase tracking-[0.14em] text-slate-400 sm:inline-flex">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-signal" />
              MVP v0.1
            </span>
            <a
              href="#console"
              className="rounded-lg border border-line bg-ink-900 px-3.5 py-1.5 text-sm text-slate-200 transition-colors hover:border-signal/40 hover:text-white"
            >
              Open console
            </a>
          </div>
        </div>
      </header>

      <main>
        <section className="relative mx-auto max-w-7xl px-6 pb-24 pt-16 lg:pb-32 lg:pt-24">
          <div className="grid grid-cols-1 items-center gap-16 lg:grid-cols-[1.02fr_0.98fr]">
            <div className="min-w-0">
              <div className="flex items-center gap-3">
                <span className="h-px w-8 bg-signal/60" />
                <span className="eyebrow">AI-powered Solidity auditor</span>
              </div>

              <h1 className="text-balance mt-6 text-4xl font-semibold leading-[1.05] tracking-tight text-white sm:text-5xl lg:text-6xl">
                Find the exploit
                <br />
                <span className="text-signal">before it ships.</span>
              </h1>

              <p className="mt-6 max-w-xl text-base leading-7 text-slate-400">
                ChainMind parses your contract, retrieves real-world exploit
                patterns with vector search, and returns a structured,
                severity-ranked audit report. Built for Solidity teams that need
                a first pass before an external audit.
              </p>

              <div className="mt-8 flex flex-wrap items-center gap-3">
                <a
                  href="#console"
                  className="inline-flex items-center gap-2 rounded-lg bg-signal px-4 py-2.5 text-sm font-medium text-ink-950 transition-colors hover:bg-signal-soft"
                >
                  Audit a contract
                  <svg
                    viewBox="0 0 16 16"
                    fill="none"
                    aria-hidden="true"
                    className="h-4 w-4"
                  >
                    <path
                      d="M3 8h9m0 0L8.5 4.5M12 8l-3.5 3.5"
                      stroke="currentColor"
                      strokeWidth="1.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                </a>
                <a
                  href="#coverage"
                  className="inline-flex items-center rounded-lg border border-line bg-ink-900/60 px-4 py-2.5 text-sm text-slate-200 transition-colors hover:border-signal/40 hover:text-white"
                >
                  Vulnerability coverage
                </a>
              </div>

              <dl className="mt-12 grid max-w-lg grid-cols-3 gap-6 border-t border-line/70 pt-6">
                <div>
                  <dt className="mono-label">Classes</dt>
                  <dd className="mt-1 text-2xl font-semibold text-white">06</dd>
                </div>
                <div>
                  <dt className="mono-label">Patterns</dt>
                  <dd className="mt-1 text-2xl font-semibold text-white">
                    21+
                  </dd>
                </div>
                <div>
                  <dt className="mono-label">Output</dt>
                  <dd className="mt-1 text-2xl font-semibold text-white">
                    JSON
                  </dd>
                </div>
              </dl>
            </div>

            <CodeWindow />
          </div>
        </section>

        <section
          id="console"
          className="scroll-mt-24 border-t border-line/60 bg-ink-900/30"
        >
          <div className="mx-auto max-w-7xl px-6 py-20">
            <div className="flex flex-col justify-between gap-6 lg:flex-row lg:items-end">
              <div className="max-w-2xl">
                <span className="eyebrow">Audit console</span>
                <h2 className="text-balance mt-4 text-3xl font-semibold tracking-tight text-white sm:text-4xl">
                  Paste the contract. Read the report.
                </h2>
                <p className="mt-4 text-base leading-7 text-slate-400">
                  One Solidity contract at a time. ChainMind parses it, retrieves
                  real exploit context and returns severity-ranked findings with
                  fixes you can review immediately.
                </p>
              </div>
              <div className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.14em] text-slate-600">
                <span className="h-1.5 w-1.5 rounded-full bg-signal" />
                Local API · v0.1
              </div>
            </div>

            <div className="mt-10 grid gap-6 lg:grid-cols-[1.45fr_0.55fr]">
              <ContractInput />

              <aside className="space-y-4">
                <div className="panel p-5">
                  <span className="mono-label">Analysis depth</span>
                  <ul className="mt-4 space-y-4">
                    <li>
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium text-white">
                          Quick
                        </span>
                        <span className="font-mono text-[10px] text-slate-600">
                          ~15s
                        </span>
                      </div>
                      <p className="mt-1 text-xs leading-5 text-slate-500">
                        High-confidence issues only. Best for a fast sanity check.
                      </p>
                    </li>
                    <li>
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium text-white">
                          Standard
                        </span>
                        <span className="font-mono text-[10px] text-slate-600">
                          ~30s
                        </span>
                      </div>
                      <p className="mt-1 text-xs leading-5 text-slate-500">
                        Balanced pass across all six vulnerability classes.
                      </p>
                    </li>
                    <li>
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium text-white">
                          Deep
                        </span>
                        <span className="font-mono text-[10px] text-slate-600">
                          ~45s
                        </span>
                      </div>
                      <p className="mt-1 text-xs leading-5 text-slate-500">
                        More retrieved context for functions with complex state.
                      </p>
                    </li>
                  </ul>
                </div>

                <div className="panel p-5">
                  <span className="mono-label">What happens next</span>
                  <ol className="mt-4 space-y-3">
                    {[
                      "Parse functions and attach contract context.",
                      "Retrieve similar exploit patterns from pgvector.",
                      "Send each chunk to the configured LLM provider.",
                      "Deduplicate, score and persist the findings.",
                    ].map((step, index) => (
                      <li key={step} className="flex gap-3">
                        <span className="mt-0.5 font-mono text-[10px] text-signal/70">
                          0{index + 1}
                        </span>
                        <span className="text-xs leading-5 text-slate-500">
                          {step}
                        </span>
                      </li>
                    ))}
                  </ol>
                </div>

                <div className="rounded-xl border border-line/70 px-5 py-4 font-mono text-[10px] uppercase tracking-[0.14em] text-slate-600">
                  Solidity only · 100k character limit
                </div>
              </aside>
            </div>
          </div>
        </section>

        <section
          id="pipeline"
          className="scroll-mt-24 border-t border-line/60 bg-ink-900/30"
        >
          <div className="mx-auto max-w-7xl px-6 py-20">
            <div className="max-w-2xl">
              <span className="eyebrow">The pipeline</span>
              <h2 className="text-balance mt-4 text-3xl font-semibold tracking-tight text-white sm:text-4xl">
                From raw Solidity to a ranked report.
              </h2>
              <p className="mt-4 text-base leading-7 text-slate-400">
                Five deterministic stages, one audit job. Every stage is
                observable through the status endpoint, so you always know where
                the analysis stands.
              </p>
            </div>

            <ol className="mt-12 grid gap-px overflow-hidden rounded-2xl border border-line/80 bg-line/60 sm:grid-cols-2 lg:grid-cols-5">
              {pipeline.map((item) => (
                <li
                  key={item.step}
                  className="group bg-ink-950/95 p-5 transition-colors hover:bg-ink-900"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-[11px] text-signal/80">
                      {item.step}
                    </span>
                    <span className="h-1 w-1 rounded-full bg-line transition-colors group-hover:bg-signal/60" />
                  </div>
                  <h3 className="mt-6 text-sm font-medium text-white">
                    {item.title}
                  </h3>
                  <p className="mt-2 text-xs leading-5 text-slate-500">
                    {item.detail}
                  </p>
                </li>
              ))}
            </ol>
          </div>
        </section>

        <section id="coverage" className="scroll-mt-24">
          <div className="mx-auto max-w-7xl px-6 py-20">
            <div className="flex flex-col justify-between gap-6 sm:flex-row sm:items-end">
              <div className="max-w-2xl">
                <span className="eyebrow">Coverage</span>
                <h2 className="text-balance mt-4 text-3xl font-semibold tracking-tight text-white sm:text-4xl">
                  The six classes that drain contracts.
                </h2>
              </div>
              <p className="max-w-sm text-sm leading-6 text-slate-500">
                Each finding maps back to a known exploit pattern with a real
                world case, an SWC reference and a concrete fix.
              </p>
            </div>

            <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {vulnerabilityClasses.map((item) => (
                <article
                  key={item.name}
                  className="panel group p-5 transition-colors hover:border-signal/30 hover:bg-ink-900"
                >
                  <div className="flex items-center justify-between">
                    <span className="flex h-7 w-7 items-center justify-center rounded border border-line bg-ink-850 font-mono text-[10px] text-slate-400 transition-colors group-hover:border-signal/30 group-hover:text-signal">
                      {item.tag}
                    </span>
                    <SeverityChip severity={item.severity} />
                  </div>
                  <h3 className="mt-5 text-[15px] font-medium text-white">
                    {item.name}
                  </h3>
                  <p className="mt-2 text-sm leading-6 text-slate-400">
                    {item.detail}
                  </p>
                  <div className="mt-5 border-t border-line/60 pt-3 font-mono text-[10px] uppercase tracking-[0.16em] text-slate-600">
                    {item.swc}
                  </div>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className="mx-auto max-w-7xl px-6 pb-24">
          <div className="panel relative overflow-hidden border-signal/20 bg-gradient-to-br from-ink-900 via-ink-900 to-signal/[0.06] px-6 py-12 sm:px-10">
            <div
              aria-hidden="true"
              className="pointer-events-none absolute -right-24 -top-24 h-64 w-64 rounded-full bg-signal/10 blur-3xl"
            />
            <div className="relative flex flex-col items-start justify-between gap-8 lg:flex-row lg:items-center">
              <div className="max-w-2xl">
                <span className="eyebrow">MVP scope</span>
                <h2 className="text-balance mt-4 text-2xl font-semibold tracking-tight text-white sm:text-3xl">
                  One contract in. One auditable report out.
                </h2>
                <p className="mt-3 text-sm leading-6 text-slate-400">
                  Solidity only, no accounts, no billing, no code rewrites. Paste
                  a contract, follow the job status, read the findings.
                </p>
              </div>
              <a
                href="#console"
                className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-signal/40 bg-signal/10 px-4 py-2.5 text-sm font-medium text-signal transition-colors hover:bg-signal/15"
              >
                Start an audit
                <svg
                  viewBox="0 0 16 16"
                  fill="none"
                  aria-hidden="true"
                  className="h-4 w-4"
                >
                  <path
                    d="M3 8h9m0 0L8.5 4.5M12 8l-3.5 3.5"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </a>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-line/60">
        <div className="mx-auto flex max-w-7xl flex-col items-start justify-between gap-4 px-6 py-8 sm:flex-row sm:items-center">
          <div className="flex items-center gap-4">
            <Logo />
            <span className="hidden text-xs text-slate-600 sm:inline">
              AI smart contract auditing
            </span>
          </div>
          <div className="flex items-center gap-5 font-mono text-[11px] text-slate-600">
            <span>Solidity only</span>
            <span className="h-1 w-1 rounded-full bg-line" />
            <span>API v0.1</span>
          </div>
        </div>
      </footer>
    </div>
  );
}