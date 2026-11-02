"use client";

import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { useState } from "react";

import type { Monaco } from "@monaco-editor/react";

import type { AnalysisDepth } from "@/lib/types";
import { useAuditStore } from "@/store/useAuditStore";

const MonacoEditor = dynamic(() => import("@monaco-editor/react"), {
  ssr: false,
  loading: () => (
    <div className="flex h-[380px] items-center justify-center bg-ink-950 font-mono text-xs text-slate-600">
      Loading editor…
    </div>
  ),
});

const MAX_SOURCE_CHARS = 100_000;

const depths: Array<{ value: AnalysisDepth; label: string }> = [
  { value: "quick", label: "Quick" },
  { value: "standard", label: "Standard" },
  { value: "deep", label: "Deep" },
];

function configureEditor(monaco: Monaco) {
  monaco.languages.register({ id: "solidity" });
  monaco.languages.setMonarchTokensProvider("solidity", {
    keywords: [
      "pragma", "solidity", "import", "contract", "interface", "library",
      "abstract", "is", "using", "for", "function", "constructor", "fallback",
      "receive", "modifier", "event", "error", "struct", "enum", "mapping",
      "returns", "return", "if", "else", "for", "while", "do", "break",
      "continue", "throw", "emit", "try", "catch", "revert", "require",
      "assert", "new", "delete", "unchecked", "assembly", "public", "private",
      "internal", "external", "pure", "view", "payable", "constant",
      "immutable", "virtual", "override", "memory", "storage", "calldata",
      "indexed", "anonymous", "true", "false",
    ],
    typeKeywords: [
      "address", "bool", "string", "bytes", "byte", "uint", "int", "uint8",
      "uint16", "uint32", "uint64", "uint128", "uint256", "int8", "int16",
      "int32", "int64", "int128", "int256", "bytes32", "bytes4", "wei", "gwei",
      "ether", "seconds", "minutes", "hours", "days", "weeks",
    ],
    operators: [
      "=", ">", "<", "!", "~", "?", ":", "==", "<=", ">=", "!=", "&&", "||",
      "++", "--", "+", "-", "*", "/", "&", "|", "^", "%", "<<", ">>", "+=",
      "-=", "*=", "/=", "&=", "|=", "^=", "%=", "<<=", ">>=", "=>",
    ],
    symbols: /[=><!~?:&|+\-*/^%]+/,
    tokenizer: {
      root: [
        [/[a-zA-Z_$][\w$]*/, { cases: { "@keywords": "keyword", "@typeKeywords": "type", "@default": "identifier" } }],
        { include: "@whitespace" },
        [/[{}()[\]]/, "@brackets"],
        [/[<>](?!@symbols)/, "@brackets"],
        [/@symbols/, { cases: { "@operators": "operator", "@default": "" } }],
        [/\d*\.\d+([eE][-+]?\d+)?/, "number.float"],
        [/0[xX][0-9a-fA-F]+/, "number.hex"],
        [/\d+/, "number"],
        [/[;,.]/, "delimiter"],
        [/"([^"\\]|\\.)*$/, "string.invalid"],
        [/"/, { token: "string.quote", bracket: "@open", next: "@string" }],
        [/'([^'\\]|\\.)*$/, "string.invalid"],
        [/'/, { token: "string.quote", bracket: "@open", next: "@stringSingle" }],
      ],
      comment: [
        [/[^/*]+/, "comment"],
        [/\/\*/, "comment", "@push"],
        [/\*\//, "comment", "@pop"],
        [/[/*]/, "comment"],
      ],
      string: [
        [/[^\\"]+/, "string"],
        [/\\./, "string.escape"],
        [/"/, { token: "string.quote", bracket: "@close", next: "@pop" }],
      ],
      stringSingle: [
        [/[^\\']+/, "string"],
        [/\\./, "string.escape"],
        [/'/, { token: "string.quote", bracket: "@close", next: "@pop" }],
      ],
      whitespace: [
        [/[ \t\r\n]+/, ""],
        [/\/\*/, "comment", "@comment"],
        [/\/\/.*$/, "comment"],
      ],
    },
  });

  monaco.editor.defineTheme("chainmind", {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "comment", foreground: "4B5563", fontStyle: "italic" },
      { token: "keyword", foreground: "34D399" },
      { token: "type", foreground: "6EE7B7" },
      { token: "string", foreground: "FBBF24" },
      { token: "number", foreground: "93C5FD" },
      { token: "identifier", foreground: "CBD5E1" },
      { token: "operator", foreground: "94A3B8" },
    ],
    colors: {
      "editor.background": "#070A0F",
      "editor.foreground": "#CBD5E1",
      "editorLineNumber.foreground": "#334155",
      "editorLineNumber.activeForeground": "#64748B",
      "editor.selectionBackground": "#34D39933",
      "editor.inactiveSelectionBackground": "#34D3991A",
      "editor.lineHighlightBackground": "#0B0F16",
      "editorCursor.foreground": "#34D399",
      "editorIndentGuide.background1": "#1C2530",
      "editorWidget.background": "#0B0F16",
      "editorWidget.border": "#1C2530",
    },
  });
}

export function ContractInput() {
  const router = useRouter();
  const submitAudit = useAuditStore((state) => state.submitAudit);
  const isSubmitting = useAuditStore((state) => state.isSubmitting);
  const submitError = useAuditStore((state) => state.submitError);

  const [contractName, setContractName] = useState("");
  const [source, setSource] = useState("");
  const [depth, setDepth] = useState<AnalysisDepth>("standard");
  const [localError, setLocalError] = useState<string | null>(null);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLocalError(null);

    if (!contractName.trim()) {
      setLocalError("Contract name is required.");
      return;
    }
    if (!source.trim()) {
      setLocalError("Paste the Solidity source code first.");
      return;
    }
    if (source.length > MAX_SOURCE_CHARS) {
      setLocalError("Source code exceeds the 100,000 character limit.");
      return;
    }

    try {
      const auditId = await submitAudit({
        contract_name: contractName.trim(),
        source_code: source,
        analysis_depth: depth,
      });
      router.push(`/audit/${auditId}`);
    } catch {
      // The store exposes the API message via submitError.
    }
  }

  const shownError = localError ?? submitError;

  return (
    <form onSubmit={handleSubmit} className="panel overflow-hidden">
      <div className="flex items-center justify-between border-b border-line/70 bg-ink-850/70 px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full bg-signal/70" />
          <span className="font-mono text-[11px] uppercase tracking-[0.16em] text-slate-400">
            Solidity source
          </span>
        </div>
        <span className="font-mono text-[10px] text-slate-600">
          {source.length.toLocaleString()} / 100,000
        </span>
      </div>

      <MonacoEditor
        height="380px"
        language="solidity"
        theme="chainmind"
        value={source}
        beforeMount={configureEditor}
        onChange={(value) => setSource(value ?? "")}
        options={{
          minimap: { enabled: false },
          fontSize: 13,
          lineHeight: 22,
          fontFamily:
            "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
          scrollBeyondLastLine: false,
          padding: { top: 16, bottom: 16 },
          renderLineHighlight: "none",
          overviewRulerBorder: false,
          wordWrap: "on",
          tabSize: 4,
          automaticLayout: true,
          scrollbar: { verticalScrollbarSize: 8, horizontalScrollbarSize: 8 },
        }}
      />

      <div className="border-t border-line/70 bg-ink-900/60 p-4 sm:p-5">
        <div className="grid gap-4 lg:grid-cols-[1.15fr_auto_auto] lg:items-end">
          <label className="block">
            <span className="mono-label">Contract name</span>
            <input
              value={contractName}
              onChange={(event) => setContractName(event.target.value)}
              placeholder="VulnerableVault"
              maxLength={255}
              className="mt-2 w-full rounded-lg border border-line bg-ink-950 px-3 py-2.5 text-sm text-white placeholder:text-slate-600 focus:border-signal/50 focus:outline-none focus:ring-1 focus:ring-signal/30"
            />
          </label>

          <div>
            <span className="mono-label">Depth</span>
            <div className="mt-2 flex rounded-lg border border-line bg-ink-950 p-1">
              {depths.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setDepth(option.value)}
                  className={`rounded-md px-3 py-1.5 font-mono text-[11px] transition-colors ${
                    depth === option.value
                      ? "bg-signal/15 text-signal"
                      : "text-slate-500 hover:text-slate-300"
                  }`}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="inline-flex h-[42px] items-center justify-center gap-2 rounded-lg bg-signal px-5 text-sm font-medium text-ink-950 transition-colors hover:bg-signal-soft disabled:cursor-not-allowed disabled:opacity-60"
          >
            {isSubmitting ? (
              <>
                <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-ink-950/30 border-t-ink-950" />
                Submitting
              </>
            ) : (
              <>
                Start audit
                <svg viewBox="0 0 16 16" fill="none" aria-hidden="true" className="h-4 w-4">
                  <path
                    d="M3 8h9m0 0L8.5 4.5M12 8l-3.5 3.5"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </>
            )}
          </button>
        </div>

        {shownError ? (
          <p className="mt-4 rounded-lg border border-severity-critical/30 bg-severity-critical/10 px-3 py-2 text-xs text-severity-critical">
            {shownError}
          </p>
        ) : (
          <p className="mt-4 text-xs text-slate-600">
            Solidity only. The contract is parsed, chunked and analyzed
            asynchronously; you will be redirected to the live report.
          </p>
        )}
      </div>
    </form>
  );
}