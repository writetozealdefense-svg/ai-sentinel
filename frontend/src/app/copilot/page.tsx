"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { usePolling } from "@/lib/usePolling";
import { categoryClass, severityClass, fmtTime } from "@/lib/decision";
import { cn } from "@/lib/utils";
import type { Violation } from "@/lib/types";
import {
  Sparkles,
  Radio,
  Send,
  Loader2,
  ScrollText,
  Quote,
  CircleAlert,
  MessageSquareText,
} from "lucide-react";

const VIOLATIONS_POLL_MS = 5000;
const HEALTH_POLL_MS = 10000;

// --- contracts/openapi/copilot.yaml (AskResponse / Citation / CopilotHealth) ---
type CopilotMode = "offline" | "live";

interface CopilotHealth {
  status: "ok";
  mode: CopilotMode;
}

interface Citation {
  kind: "violation" | "control" | "prompt" | "finding" | string;
  ref: string;
  excerpt?: string;
}

interface AskResponse {
  mode: CopilotMode;
  views: {
    executive?: string;
    technical?: string;
    compliance?: string;
  };
  citations: Citation[];
}

const VIEW_TABS = ["executive", "technical", "compliance"] as const;
type ViewTab = (typeof VIEW_TABS)[number];

const SUGGESTED_QUESTIONS = [
  "Why was the last prompt blocked?",
  "Who is the riskiest user?",
  "What is our compliance posture?",
  "Which agent is the riskiest?",
] as const;

// Citation chip accent per kind (mirrors the platform's category-chip idiom).
function citationClass(kind: string): string {
  switch (kind.toLowerCase()) {
    case "violation":
      return "border-red-500/40 bg-red-500/10 text-red-300";
    case "control":
      return "border-purple-500/40 bg-purple-500/10 text-purple-300";
    case "prompt":
      return "border-sky-500/40 bg-sky-500/10 text-sky-300";
    case "finding":
      return "border-amber-500/40 bg-amber-500/10 text-amber-300";
    default:
      return "border-border bg-muted/40 text-muted-foreground";
  }
}

// Polls the copilot healthz via the BFF; drives the header mode badge.
function useCopilotHealth(): {
  health: CopilotHealth | null;
  error: string | null;
  ready: boolean;
} {
  const [health, setHealth] = useState<CopilotHealth | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const mounted = useRef(true);

  const load = useCallback(async () => {
    try {
      const res = await fetch("/api/copilot/health", { cache: "no-store" });
      const body = await res.json().catch(() => null);
      if (!mounted.current) return;
      if (!res.ok) {
        setHealth(null);
        setError(
          (body && (body.error as string)) || `request failed (${res.status})`
        );
      } else {
        setHealth(body as CopilotHealth);
        setError(null);
      }
    } catch (e) {
      if (!mounted.current) return;
      setHealth(null);
      setError(e instanceof Error ? e.message : "network error");
    } finally {
      if (mounted.current) setReady(true);
    }
  }, []);

  useEffect(() => {
    mounted.current = true;
    void load();
    const id = setInterval(() => void load(), HEALTH_POLL_MS);
    return () => {
      mounted.current = false;
      clearInterval(id);
    };
  }, [load]);

  return { health, error, ready };
}

function ModeBadge({
  health,
  error,
  ready,
}: {
  health: CopilotHealth | null;
  error: string | null;
  ready: boolean;
}) {
  if (!ready) {
    return (
      <span className="inline-flex items-center gap-1.5 rounded border border-border bg-muted/40 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">
        <Loader2 className="h-3 w-3 animate-spin" />
        Checking
      </span>
    );
  }
  if (error || !health) {
    return (
      <span
        className="inline-flex items-center gap-1.5 rounded border border-red-500/50 bg-red-500/15 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] text-red-300"
        title={error ?? undefined}
      >
        <CircleAlert className="h-3 w-3" />
        copilot-svc unreachable
      </span>
    );
  }
  if (health.mode === "live") {
    return (
      <span className="inline-flex items-center gap-1.5 rounded border border-emerald-500/50 bg-emerald-500/15 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] text-emerald-300">
        <Radio className="h-3 w-3 animate-pulseGlow" />
        Live
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1.5 rounded border border-amber-500/50 bg-amber-500/15 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] text-amber-300">
      <ScrollText className="h-3 w-3" />
      Offline · Deterministic
    </span>
  );
}

function ViolationRow({
  violation,
  selected,
  disabled,
  onExplain,
}: {
  violation: Violation;
  selected: boolean;
  disabled: boolean;
  onExplain: (v: Violation) => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onExplain(violation)}
      aria-pressed={selected}
      className={cn(
        "w-full rounded-lg border px-3 py-2.5 text-left transition-colors",
        selected
          ? "border-accent-blue/60 bg-accent-blue/10 shadow-glow"
          : "border-border bg-muted/20 hover:border-accent-blue/40 hover:bg-muted/40",
        disabled && "cursor-wait opacity-60"
      )}
    >
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="font-mono text-[10px] font-bold uppercase tracking-wider text-foreground">
          {violation.framework}
        </span>
        <span className="font-mono text-[10px] text-muted-foreground">
          {violation.controlId}
        </span>
        {violation.category && (
          <span
            className={cn(
              "rounded border px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wider",
              categoryClass(String(violation.category))
            )}
          >
            {violation.category}
          </span>
        )}
        <span
          className={cn(
            "rounded border px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wider",
            severityClass(violation.severity)
          )}
        >
          {violation.severity}
        </span>
      </div>
      {violation.description && (
        <p className="mt-1.5 line-clamp-2 text-[11px] leading-snug text-muted-foreground">
          {violation.description}
        </p>
      )}
      <div className="mt-1.5 flex items-center justify-between font-mono text-[10px] text-muted-foreground/80">
        <span className="truncate">{violation.userEmail ?? "—"}</span>
        <span>{fmtTime(violation.timestamp)}</span>
      </div>
    </button>
  );
}

function CitationChips({ citations }: { citations: Citation[] }) {
  const [expanded, setExpanded] = useState<number | null>(null);

  if (citations.length === 0) return null;
  const open = expanded !== null ? citations[expanded] : null;

  return (
    <div className="border-t border-border pt-3">
      <p className="mb-2 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground">
        <Quote className="h-3 w-3" />
        Grounded in
      </p>
      <div className="flex flex-wrap gap-1.5">
        {citations.map((c, i) => (
          <button
            key={`${c.kind}-${c.ref}-${i}`}
            type="button"
            onClick={() => setExpanded(expanded === i ? null : i)}
            title={c.excerpt}
            className={cn(
              "inline-flex items-center gap-1.5 rounded border px-2 py-0.5 font-mono text-[10px] transition-colors",
              citationClass(c.kind),
              expanded === i && "ring-1 ring-current"
            )}
          >
            <span className="font-bold uppercase tracking-wider">{c.kind}</span>
            <span className="max-w-[16rem] truncate opacity-90">{c.ref}</span>
          </button>
        ))}
      </div>
      {open?.excerpt && (
        <blockquote className="mt-2 whitespace-pre-wrap rounded border border-border bg-muted/30 px-3 py-2 font-mono text-[11px] leading-relaxed text-muted-foreground">
          {open.excerpt}
        </blockquote>
      )}
    </div>
  );
}

function AnswerPanel({
  answer,
  contextLabel,
  initialTab,
}: {
  answer: AskResponse;
  contextLabel: string;
  initialTab?: ViewTab;
}) {
  const available = VIEW_TABS.filter((t) => Boolean(answer.views[t]));
  const [tab, setTab] = useState<ViewTab>(
    initialTab && available.includes(initialTab)
      ? initialTab
      : (available[0] ?? "executive")
  );
  const active: ViewTab = available.includes(tab) ? tab : (available[0] ?? tab);
  const text = answer.views[active];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="min-w-0 truncate font-mono text-[11px] text-muted-foreground">
          {contextLabel}
        </p>
        <span
          className={cn(
            "rounded border px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-[0.2em]",
            answer.mode === "live"
              ? "border-emerald-500/50 bg-emerald-500/15 text-emerald-300"
              : "border-amber-500/50 bg-amber-500/15 text-amber-300"
          )}
        >
          {answer.mode}
        </span>
      </div>

      {available.length > 0 ? (
        <>
          <div
            role="tablist"
            className="flex items-center gap-1 rounded-lg border border-border bg-card/50 p-1"
          >
            {available.map((t) => (
              <button
                key={t}
                role="tab"
                aria-selected={active === t}
                onClick={() => setTab(t)}
                className={cn(
                  "flex-1 rounded-md px-3 py-1.5 font-mono text-xs font-semibold uppercase tracking-wider transition-colors",
                  active === t
                    ? "bg-accent-blue/15 text-accent-blue shadow-glow"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                {t}
              </button>
            ))}
          </div>
          <div
            role="tabpanel"
            className="whitespace-pre-wrap rounded-lg border border-border bg-muted/20 px-4 py-3 text-sm leading-relaxed text-foreground"
          >
            {text}
          </div>
        </>
      ) : (
        <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
          The copilot returned no views for this request.
        </div>
      )}

      <CitationChips citations={answer.citations ?? []} />
    </div>
  );
}

export default function CopilotPage() {
  const health = useCopilotHealth();
  const violations = usePolling<Violation>(
    "/api/violations",
    VIOLATIONS_POLL_MS
  );
  const violationsLoading = violations.loading && !violations.ready;

  const [question, setQuestion] = useState("");
  const [busy, setBusy] = useState(false);
  const [answer, setAnswer] = useState<AskResponse | null>(null);
  const [answerError, setAnswerError] = useState<string | null>(null);
  const [contextLabel, setContextLabel] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  // Remounts AnswerPanel per answer so the tab state resets to the first view.
  const [answerKey, setAnswerKey] = useState(0);

  const applyResult = useCallback(
    (body: unknown, ok: boolean, status: number, label: string) => {
      if (!ok) {
        const envelope = (body ?? {}) as { error?: string; detail?: string };
        const msg = envelope.error ?? `request failed (${status})`;
        setAnswer(null);
        setAnswerError(
          envelope.detail ? `${msg} — ${envelope.detail}` : msg
        );
        return;
      }
      setAnswer(body as AskResponse);
      setAnswerError(null);
      setContextLabel(label);
      setAnswerKey((k) => k + 1);
    },
    []
  );

  const runExplainById = useCallback(
    async (violationId: string, label: string) => {
      if (busy) return;
      setBusy(true);
      setSelectedId(violationId);
      setContextLabel(label);
      try {
        const res = await fetch(
          `/api/copilot/explain/${encodeURIComponent(violationId)}`,
          { cache: "no-store" }
        );
        const body = await res.json().catch(() => null);
        applyResult(body, res.ok, res.status, label);
      } catch (e) {
        setAnswer(null);
        setAnswerError(e instanceof Error ? e.message : "network error");
      } finally {
        setBusy(false);
      }
    },
    [busy, applyResult]
  );

  const runExplain = useCallback(
    (v: Violation) => {
      void runExplainById(
        v.violationId,
        `Explain · ${v.framework} ${v.controlId} · ${v.violationId}`
      );
    },
    [runExplainById]
  );

  // ?explain=<violationId> deep-links an Explain (gate evidence / demo
  // runbook) — same pattern as /graph's ?q= initial-load effect. Guarded by a
  // ref so it fires exactly once, even as callback identities change.
  // Optional &tab=executive|technical|compliance pre-selects the rendered
  // view (headless screenshot capture can't click tabs).
  const [deepLinkTab, setDeepLinkTab] = useState<ViewTab | null>(null);
  const deepLinked = useRef(false);
  useEffect(() => {
    if (deepLinked.current) return;
    deepLinked.current = true;
    const params = new URLSearchParams(window.location.search);
    const requested = params.get("explain");
    if (!requested) return;
    const tab = params.get("tab");
    if (tab && (VIEW_TABS as readonly string[]).includes(tab)) {
      setDeepLinkTab(tab as ViewTab);
    }
    void runExplainById(requested, `Explain · ${requested}`);
  }, [runExplainById]);

  const runAsk = useCallback(
    async (q: string) => {
      const trimmed = q.trim();
      if (!trimmed || busy) return;
      setBusy(true);
      setSelectedId(null);
      try {
        const res = await fetch("/api/copilot/ask", {
          method: "POST",
          headers: { "content-type": "application/json" },
          body: JSON.stringify({ question: trimmed }),
          cache: "no-store",
        });
        const body = await res.json().catch(() => null);
        applyResult(body, res.ok, res.status, `Ask · “${trimmed}”`);
      } catch (e) {
        setAnswer(null);
        setAnswerError(e instanceof Error ? e.message : "network error");
      } finally {
        setBusy(false);
      }
    },
    [busy, applyResult]
  );

  return (
    <main className="relative min-h-screen">
      <div className="pointer-events-none fixed inset-0 grid-bg opacity-30" />
      <div className="relative mx-auto max-w-7xl px-4 py-6 md:px-8 md:py-10">
        <header className="mb-6 flex flex-col gap-2">
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-accent-purple/40 bg-accent-purple/10 shadow-[0_0_20px_-2px_hsl(270_80%_60%/0.5)]">
              <Sparkles className="h-6 w-6 text-accent-purple" />
            </div>
            <div>
              <h1 className="flex flex-wrap items-center gap-3 text-xl font-bold tracking-tight text-glow-blue md:text-2xl">
                AI Security <span className="text-accent-purple">Copilot</span>
                <ModeBadge {...health} />
              </h1>
              <p className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.3em] text-muted-foreground">
                <Radio className="h-3 w-3 animate-pulseGlow text-accent-teal" />
                Grounded RAG over platform incidents · citations required
              </p>
            </div>
          </div>
        </header>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]">
          {/* Left: recent violations (click → explain) */}
          <Card className="glass self-start">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">
                <ScrollText className="h-4 w-4 text-accent-blue" />
                Recent Violations
                <span className="ml-auto font-mono text-[10px] font-normal uppercase tracking-wider text-muted-foreground">
                  click to explain
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {violationsLoading && (
                <div className="space-y-2">
                  {[0, 1, 2, 3].map((i) => (
                    <div
                      key={i}
                      className="h-20 animate-pulse rounded-lg border border-border bg-muted/30"
                    />
                  ))}
                </div>
              )}
              {!violationsLoading && violations.error && (
                <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
                  No violations available — {violations.error}
                </div>
              )}
              {!violationsLoading &&
                !violations.error &&
                violations.data.length === 0 && (
                  <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
                    No violations recorded yet. Run a demo scenario.
                  </div>
                )}
              {!violationsLoading &&
                !violations.error &&
                violations.data.length > 0 && (
                  <div className="max-h-[34rem] space-y-2 overflow-y-auto pr-1">
                    {violations.data.map((v) => (
                      <ViolationRow
                        key={v.violationId}
                        violation={v}
                        selected={selectedId === v.violationId}
                        disabled={busy}
                        onExplain={runExplain}
                      />
                    ))}
                  </div>
                )}
            </CardContent>
          </Card>

          {/* Main: ask box + grounded answer */}
          <Card className="glass self-start">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">
                <MessageSquareText className="h-4 w-4 text-accent-purple" />
                Ask the Copilot
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  void runAsk(question);
                }}
                className="flex items-center gap-2"
              >
                <input
                  type="text"
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                  placeholder="Ask about a violation, user risk, or compliance posture…"
                  className="h-9 min-w-0 flex-1 rounded-md border border-border bg-muted/30 px-3 font-mono text-xs text-foreground placeholder:text-muted-foreground/60 focus:border-accent-purple/60 focus:outline-none focus:ring-1 focus:ring-accent-purple/40"
                />
                <button
                  type="submit"
                  disabled={busy || !question.trim()}
                  className="inline-flex h-9 items-center gap-1.5 rounded-md border border-accent-purple/40 bg-accent-purple/15 px-3 font-mono text-xs font-semibold uppercase tracking-wider text-accent-purple transition-colors hover:bg-accent-purple/25 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {busy ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Send className="h-3.5 w-3.5" />
                  )}
                  Ask
                </button>
              </form>

              <div className="flex flex-wrap gap-1.5">
                {SUGGESTED_QUESTIONS.map((q) => (
                  <button
                    key={q}
                    type="button"
                    disabled={busy}
                    onClick={() => {
                      setQuestion(q);
                      void runAsk(q);
                    }}
                    className="rounded-full border border-border bg-muted/30 px-3 py-1 font-mono text-[10px] text-muted-foreground transition-colors hover:border-accent-purple/50 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {q}
                  </button>
                ))}
              </div>

              {busy && (
                <div className="flex items-center justify-center gap-2 rounded-lg border border-dashed border-border px-3 py-10 font-mono text-xs text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin text-accent-purple" />
                  Retrieving evidence…
                </div>
              )}

              {!busy && answerError && (
                <div className="flex items-start gap-2 rounded-lg border border-red-500/40 bg-red-500/10 px-3 py-4 font-mono text-xs text-red-300">
                  <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <span className="break-all">
                    Copilot request failed — {answerError}
                  </span>
                </div>
              )}

              {!busy && !answerError && !answer && (
                <div className="rounded-lg border border-dashed border-border px-3 py-10 text-center font-mono text-xs text-muted-foreground">
                  Select a violation to explain, or ask a question above.
                </div>
              )}

              {!busy && !answerError && answer && (
                <AnswerPanel
                  key={answerKey}
                  answer={answer}
                  contextLabel={contextLabel}
                  initialTab={deepLinkTab ?? undefined}
                />
              )}
            </CardContent>
          </Card>
        </div>

        <footer className="mt-8 text-center font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground/70">
          Answers are grounded in stored violations + fixture controls · offline
          deterministic mode is the default (Brief R3)
        </footer>
      </div>
    </main>
  );
}
