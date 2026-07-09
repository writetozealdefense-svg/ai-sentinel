"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SimulatedTag } from "@/components/SimulatedTag";
import { usePolling } from "@/lib/usePolling";
import { fmtTime } from "@/lib/decision";
import { cn } from "@/lib/utils";
import type { Agent } from "@/lib/types";
import {
  Bot,
  Radio,
  Wrench,
  AlertTriangle,
  UserRound,
  Clock3,
} from "lucide-react";

const POLL_MS = 4000;
const RISK_THRESHOLD = 70;

// Framework badge styling — one accent per supported agent framework.
function frameworkClass(framework: string): string {
  switch (framework.toLowerCase()) {
    case "crewai":
      return "border-orange-500/40 bg-orange-500/10 text-orange-300";
    case "langgraph":
      return "border-emerald-500/40 bg-emerald-500/10 text-emerald-300";
    case "autogen":
      return "border-sky-500/40 bg-sky-500/10 text-sky-300";
    case "openai-agents-sdk":
      return "border-violet-500/40 bg-violet-500/10 text-violet-300";
    case "mcp":
      return "border-teal-500/40 bg-teal-500/10 text-teal-300";
    default:
      return "border-border bg-muted/40 text-muted-foreground";
  }
}

// Circular 0-100 gauge with a hard THRESHOLD (70) rule: red + glow when
// breached (spec), green/amber below it.
function AgentRiskGauge({ score }: { score: number }) {
  const clamped = Math.max(0, Math.min(100, Math.round(score)));
  const breached = clamped >= RISK_THRESHOLD;
  const size = 92;
  const stroke = 8;
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const offset = c - (clamped / 100) * c;
  const color = breached
    ? "hsl(0 84% 60%)"
    : clamped >= 40
      ? "hsl(38 92% 55%)"
      : "hsl(160 84% 45%)";

  return (
    <div className="relative inline-flex items-center justify-center">
      <svg
        width={size}
        height={size}
        className={cn("-rotate-90", breached && "animate-pulseGlow")}
      >
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="hsl(var(--muted))"
          strokeWidth={stroke}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeDasharray={c}
          strokeDashoffset={offset}
          strokeLinecap="round"
          style={{
            filter: `drop-shadow(0 0 ${breached ? 10 : 5}px ${color})`,
            transition: "stroke-dashoffset 0.8s ease, stroke 0.4s ease",
          }}
        />
      </svg>
      <div className="absolute flex flex-col items-center">
        <span
          className={cn(
            "font-mono text-2xl font-bold",
            breached ? "text-red-400" : clamped >= 40 ? "text-amber-400" : "text-emerald-400"
          )}
        >
          {clamped}
        </span>
        <span className="font-mono text-[9px] uppercase tracking-widest text-muted-foreground">
          risk
        </span>
      </div>
    </div>
  );
}

function AgentCard({ agent }: { agent: Agent }) {
  const breached = agent.riskScore >= RISK_THRESHOLD;
  return (
    <Card
      className={cn(
        "glass overflow-hidden transition-shadow",
        breached &&
          "border-red-500/50 shadow-[0_0_28px_-4px_hsl(0_84%_60%/0.45)]"
      )}
    >
      <CardHeader className="pb-3">
        <CardTitle className="justify-between">
          <span className="flex min-w-0 items-center gap-2">
            <Bot
              className={cn(
                "h-4 w-4 shrink-0",
                breached ? "text-red-400" : "text-accent-blue"
              )}
            />
            <span className="truncate text-sm">
              {agent.name || agent.agentId}
            </span>
          </span>
          <SimulatedTag />
        </CardTitle>
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <span
            className={cn(
              "rounded border px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider",
              frameworkClass(String(agent.framework))
            )}
          >
            {agent.framework}
          </span>
          <span className="font-mono text-[10px] text-muted-foreground">
            {agent.agentId}
          </span>
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1 space-y-2.5">
            {agent.owner && (
              <div className="flex items-center gap-1.5 font-mono text-[11px] text-muted-foreground">
                <UserRound className="h-3 w-3 shrink-0" />
                <span className="truncate">{agent.owner}</span>
              </div>
            )}
            {agent.tools && agent.tools.length > 0 && (
              <div className="flex flex-wrap items-center gap-1">
                <Wrench className="h-3 w-3 text-muted-foreground" />
                {agent.tools.map((t) => (
                  <span
                    key={t}
                    className="rounded border border-accent-blue/30 bg-accent-blue/10 px-1.5 py-0.5 font-mono text-[10px] text-sky-300"
                  >
                    {t}
                  </span>
                ))}
              </div>
            )}
            {agent.riskFactors && agent.riskFactors.length > 0 && (
              <ul className="space-y-1">
                {agent.riskFactors.map((f) => (
                  <li
                    key={f}
                    className="flex items-start gap-1.5 font-mono text-[11px] text-amber-300/90"
                  >
                    <span className="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-amber-400" />
                    {f}
                  </li>
                ))}
              </ul>
            )}
            <div className="flex items-center gap-1.5 font-mono text-[10px] text-muted-foreground/80">
              <Clock3 className="h-3 w-3 shrink-0" />
              last seen {fmtTime(agent.lastSeen)}
            </div>
          </div>
          <div className="flex shrink-0 flex-col items-center gap-2">
            <AgentRiskGauge score={agent.riskScore} />
            {breached && (
              <span className="inline-flex animate-pulseGlow items-center gap-1 rounded border border-red-500/60 bg-red-500/15 px-2 py-0.5 font-mono text-[9px] font-bold uppercase tracking-[0.2em] text-red-300">
                <AlertTriangle className="h-3 w-3" />
                Threshold breach
              </span>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export default function AgentsPage() {
  const agents = usePolling<Agent>("/api/agents", POLL_MS);
  const loading = agents.loading && !agents.ready;

  return (
    <main className="relative min-h-screen">
      <div className="pointer-events-none fixed inset-0 grid-bg opacity-30" />
      <div className="relative mx-auto max-w-7xl px-4 py-6 md:px-8 md:py-10">
        <header className="mb-6 flex flex-col gap-2">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-orange-500/40 bg-orange-500/10 shadow-[0_0_20px_-2px_hsl(25_95%_55%/0.5)]">
              <Bot className="h-6 w-6 text-orange-400" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight text-glow-blue md:text-2xl">
                Agent <span className="text-orange-400">Security</span>
              </h1>
              <p className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.3em] text-muted-foreground">
                <Radio className="h-3 w-3 animate-pulseGlow text-accent-teal" />
                Live · auto-refresh {POLL_MS / 1000}s · telemetry is SIMULATED
              </p>
            </div>
          </div>
        </header>

        {loading && (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {[0, 1, 2].map((i) => (
              <div
                key={i}
                className="h-52 animate-pulse rounded-xl border border-border bg-muted/30"
              />
            ))}
          </div>
        )}

        {!loading && agents.error && (
          <div className="rounded-lg border border-dashed border-border px-3 py-10 text-center font-mono text-xs text-muted-foreground">
            No data yet — {agents.error}
          </div>
        )}

        {!loading && !agents.error && agents.data.length === 0 && (
          <div className="rounded-lg border border-dashed border-border px-3 py-10 text-center font-mono text-xs text-muted-foreground">
            No agents discovered yet. Run Scenario 5.
          </div>
        )}

        {!loading && !agents.error && agents.data.length > 0 && (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {agents.data.map((a) => (
              <AgentCard key={a.agentId} agent={a} />
            ))}
          </div>
        )}

        <footer className="mt-8 text-center font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground/70">
          Agent telemetry is SIMULATED (Brief R4) · risk threshold ={" "}
          {RISK_THRESHOLD}
        </footer>
      </div>
    </main>
  );
}
