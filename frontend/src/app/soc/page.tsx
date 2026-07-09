"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { usePolling } from "@/lib/usePolling";
import {
  decisionStyle,
  riskColor,
  categoryClass,
  severityClass,
  priorityClass,
  fmtTime,
} from "@/lib/decision";
import { cn } from "@/lib/utils";
import type {
  PromptSummary,
  Violation,
  RuntimeEvent,
  Decision,
} from "@/lib/types";
import {
  Radar,
  ScanLine,
  ShieldAlert,
  Server,
  AlertTriangle,
  Radio,
} from "lucide-react";

const POLL_MS = 4000;

export default function SocPage() {
  const prompts = usePolling<PromptSummary>("/api/prompts", POLL_MS);
  const violations = usePolling<Violation>("/api/violations", POLL_MS);
  const runtime = usePolling<RuntimeEvent>("/api/runtime", POLL_MS);

  return (
    <main className="relative min-h-screen">
      <div className="pointer-events-none fixed inset-0 grid-bg opacity-30" />
      <div className="relative mx-auto max-w-7xl px-4 py-6 md:px-8 md:py-10">
        <header className="mb-6 flex flex-col gap-2">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-red-500/40 bg-red-500/10 shadow-[0_0_20px_-2px_hsl(0_84%_60%/0.5)]">
              <Radar className="h-6 w-6 text-red-400" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight text-glow-blue md:text-2xl">
                SOC{" "}
                <span className="text-red-400">Threat Monitor</span>
              </h1>
              <p className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.3em] text-muted-foreground">
                <Radio className="h-3 w-3 animate-pulseGlow text-accent-teal" />
                Live · auto-refresh {POLL_MS / 1000}s
              </p>
            </div>
          </div>
        </header>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <PromptMonitorPanel
              rows={prompts.data}
              error={prompts.error}
              loading={prompts.loading && !prompts.ready}
            />
          </div>
          <div className="lg:col-span-1">
            <DlpAlertsPanel
              rows={violations.data}
              error={violations.error}
              loading={violations.loading && !violations.ready}
            />
          </div>
          <div className="lg:col-span-3">
            <RuntimePanel
              rows={runtime.data}
              error={runtime.error}
              loading={runtime.loading && !runtime.ready}
            />
          </div>
        </div>

        <footer className="mt-8 text-center font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground/70">
          Prompt + DLP data live via BFF · runtime feed is SIMULATED
        </footer>
      </div>
    </main>
  );
}

function PanelEmpty({ text }: { text: string }) {
  return (
    <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
      {text}
    </div>
  );
}

function Skeleton() {
  return (
    <div className="space-y-2">
      {[0, 1, 2, 3].map((i) => (
        <div
          key={i}
          className="h-11 animate-pulse rounded-lg border border-border bg-muted/30"
        />
      ))}
    </div>
  );
}

function DecisionBadge({ decision }: { decision: Decision }) {
  const s = decisionStyle[decision] ?? decisionStyle.ALLOW;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-md px-2 py-0.5 font-mono text-[10px] font-bold tracking-wider",
        s.className
      )}
    >
      <span className={cn("h-1.5 w-1.5 rounded-full", s.dot)} />
      {s.label}
    </span>
  );
}

function PromptMonitorPanel({
  rows,
  error,
  loading,
}: {
  rows: PromptSummary[];
  error: string | null;
  loading: boolean;
}) {
  return (
    <Card className="glass overflow-hidden">
      <CardHeader>
        <CardTitle className="justify-between">
          <span className="flex items-center gap-2">
            <ScanLine className="h-4 w-4 text-accent-blue" />
            Live Prompt Monitoring
          </span>
          <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
            {rows.length} recent
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading && <Skeleton />}
        {!loading && error && (
          <PanelEmpty text={`No data yet — ${error}`} />
        )}
        {!loading && !error && rows.length === 0 && (
          <PanelEmpty text="No inspected prompts yet. Trigger a scenario." />
        )}
        {!loading && !error && rows.length > 0 && (
          <div className="space-y-1.5">
            {rows.map((p, i) => (
              <div
                key={p.promptId}
                style={{ animationDelay: `${Math.min(i, 8) * 40}ms` }}
                className="animate-streamIn rounded-lg border border-border bg-background/40 px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="flex min-w-0 items-center gap-2">
                    <span className="font-mono text-[11px] text-muted-foreground">
                      {fmtTime(p.timestamp)}
                    </span>
                    <span className="truncate font-mono text-xs text-foreground">
                      {p.userEmail}
                    </span>
                    <span className="rounded border border-border bg-muted/40 px-1.5 py-0.5 font-mono text-[10px] capitalize text-muted-foreground">
                      {p.provider}
                    </span>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <span
                      className={cn(
                        "font-mono text-xs font-bold",
                        riskColor(p.riskScore)
                      )}
                    >
                      {p.riskScore}
                    </span>
                    <DecisionBadge decision={p.decision} />
                  </div>
                </div>
                {p.categories && p.categories.length > 0 && (
                  <div className="mt-1.5 flex flex-wrap gap-1">
                    {p.categories.map((c) => (
                      <span
                        key={String(c)}
                        className={cn(
                          "rounded border px-1.5 py-0.5 font-mono text-[9px] font-semibold tracking-wide",
                          categoryClass(String(c))
                        )}
                      >
                        {String(c)}
                      </span>
                    ))}
                    {p.artifactName && (
                      <span className="rounded border border-border bg-muted/30 px-1.5 py-0.5 font-mono text-[9px] text-muted-foreground">
                        {p.artifactName}
                      </span>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function DlpAlertsPanel({
  rows,
  error,
  loading,
}: {
  rows: Violation[];
  error: string | null;
  loading: boolean;
}) {
  return (
    <Card className="glass overflow-hidden">
      <CardHeader>
        <CardTitle className="justify-between">
          <span className="flex items-center gap-2">
            <ShieldAlert className="h-4 w-4 text-amber-400" />
            DLP Alerts
          </span>
          <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
            {rows.length}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading && <Skeleton />}
        {!loading && error && <PanelEmpty text={`No data yet — ${error}`} />}
        {!loading && !error && rows.length === 0 && (
          <PanelEmpty text="No violations yet." />
        )}
        {!loading && !error && rows.length > 0 && (
          <div className="space-y-1.5">
            {rows.map((v, i) => (
              <div
                key={v.violationId}
                style={{ animationDelay: `${Math.min(i, 8) * 40}ms` }}
                className="animate-streamIn rounded-lg border border-border bg-background/40 px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-mono text-xs font-semibold text-foreground">
                    {v.framework}
                  </span>
                  <span
                    className={cn(
                      "rounded border px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wider",
                      severityClass(v.severity)
                    )}
                  >
                    {v.severity}
                  </span>
                </div>
                <div className="mt-1 flex items-center justify-between gap-2">
                  <span className="truncate font-mono text-[11px] text-accent-teal">
                    {v.controlId}
                  </span>
                  <span className="shrink-0 font-mono text-[10px] text-muted-foreground">
                    {fmtTime(v.timestamp)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function RuntimePanel({
  rows,
  error,
  loading,
}: {
  rows: RuntimeEvent[];
  error: string | null;
  loading: boolean;
}) {
  // LIVE only when every fetched event explicitly carries simulated:false
  // (real Falco via falcosidekick, Helm hardening path). Mixed or absent
  // events keep the SIMULATED badge.
  // (cast: the compose-era type still narrows `simulated` to literal true)
  const live =
    rows.length > 0 && rows.every((e) => (e.simulated as boolean) === false);
  return (
    <Card className="glass overflow-hidden border-amber-500/30">
      <CardHeader>
        <CardTitle className="justify-between">
          <span className="flex items-center gap-2">
            <Server className="h-4 w-4 text-accent-purple" />
            Runtime Events
            <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
              (Falco / Tetragon)
            </span>
          </span>
          {live ? (
            <span
              title="Real Falco events via falcosidekick (Helm hardening path)."
              className="inline-flex items-center gap-1 rounded border border-emerald-500/50 bg-emerald-500/15 px-2 py-0.5 font-mono text-[10px] font-bold tracking-[0.2em] text-emerald-300"
            >
              <Radio className="h-3 w-3" />
              LIVE (Falco)
            </span>
          ) : (
            <span
              title="No live kernel telemetry on the compose path (Brief R1)."
              className="inline-flex items-center gap-1 rounded border border-amber-500/50 bg-amber-500/15 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] text-amber-300"
            >
              <AlertTriangle className="h-3 w-3" />
              Simulated
            </span>
          )}
        </CardTitle>
        <p className="font-mono text-[10px] text-muted-foreground">
          No live kernel telemetry on the compose path (Brief R1). Events are
          synthetic.
        </p>
      </CardHeader>
      <CardContent>
        {loading && <Skeleton />}
        {!loading && error && <PanelEmpty text={`No data yet — ${error}`} />}
        {!loading && !error && rows.length === 0 && (
          <PanelEmpty text="No runtime events yet." />
        )}
        {!loading && !error && rows.length > 0 && (
          <div className="grid grid-cols-1 gap-1.5 md:grid-cols-2">
            {rows.map((e, i) => (
              <div
                key={e.eventId}
                style={{ animationDelay: `${Math.min(i, 8) * 30}ms` }}
                className="animate-streamIn rounded-lg border border-border bg-background/40 px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate font-mono text-xs font-semibold text-foreground">
                    {e.rule}
                  </span>
                  <div className="flex shrink-0 items-center gap-1.5">
                    <span className="rounded border border-border bg-muted/40 px-1.5 py-0.5 font-mono text-[9px] uppercase text-muted-foreground">
                      {e.source}
                    </span>
                    <span
                      className={cn(
                        "rounded border px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-wider",
                        priorityClass(e.priority)
                      )}
                    >
                      {e.priority}
                    </span>
                  </div>
                </div>
                <p className="mt-1 truncate font-mono text-[11px] text-muted-foreground">
                  {e.message}
                </p>
                <div className="mt-1 flex items-center justify-between gap-2 font-mono text-[10px] text-muted-foreground/80">
                  <span className="truncate">
                    {e.pod ?? e.namespace ?? "—"}
                    {e.container ? ` · ${e.container}` : ""}
                  </span>
                  <span className="shrink-0">{fmtTime(e.timestamp)}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
