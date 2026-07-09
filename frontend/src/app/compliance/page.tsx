"use client";

import { useCallback, useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { categoryClass, severityClass, fmtTime } from "@/lib/decision";
import { cn } from "@/lib/utils";
import type {
  FrameworkScore,
  ControlMapping,
  Violation,
} from "@/lib/types";
import {
  Scale,
  AlertTriangle,
  Info,
  ListChecks,
  Loader2,
  ShieldQuestion,
} from "lucide-react";

async function getJson<T>(
  url: string
): Promise<{ data?: T; error?: string }> {
  try {
    const res = await fetch(url, { cache: "no-store" });
    const body = await res.json().catch(() => null);
    if (!res.ok) {
      const msg =
        (body && (body.error as string)) || `request failed (${res.status})`;
      return { error: msg };
    }
    return { data: body as T };
  } catch (e) {
    return { error: e instanceof Error ? e.message : "network error" };
  }
}

function scoreColor(score: number): string {
  if (score >= 80) return "text-emerald-300";
  if (score >= 60) return "text-amber-300";
  if (score >= 40) return "text-orange-300";
  return "text-red-300";
}

function scoreBar(score: number): string {
  if (score >= 80) return "bg-emerald-500";
  if (score >= 60) return "bg-amber-500";
  if (score >= 40) return "bg-orange-500";
  return "bg-red-500";
}

export default function CompliancePage() {
  const [frameworks, setFrameworks] = useState<FrameworkScore[]>([]);
  const [frameworksError, setFrameworksError] = useState<string | null>(null);
  const [frameworksLoading, setFrameworksLoading] = useState(true);

  const [selected, setSelected] = useState<string | null>(null);
  const [controls, setControls] = useState<ControlMapping[]>([]);
  const [controlsError, setControlsError] = useState<string | null>(null);
  const [controlsLoading, setControlsLoading] = useState(false);

  const [violations, setViolations] = useState<Violation[]>([]);
  const [violationsError, setViolationsError] = useState<string | null>(null);
  const [violationsLoading, setViolationsLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      const { data, error } = await getJson<FrameworkScore[]>("/api/frameworks");
      setFrameworks(Array.isArray(data) ? data : []);
      setFrameworksError(error ?? null);
      setFrameworksLoading(false);
    })();
    void (async () => {
      const { data, error } = await getJson<Violation[]>("/api/violations");
      setViolations(Array.isArray(data) ? data : []);
      setViolationsError(error ?? null);
      setViolationsLoading(false);
    })();
  }, []);

  const selectFramework = useCallback(async (framework: string) => {
    setSelected(framework);
    setControlsLoading(true);
    setControlsError(null);
    const { data, error } = await getJson<ControlMapping[]>(
      `/api/controls?framework=${encodeURIComponent(framework)}`
    );
    setControls(Array.isArray(data) ? data : []);
    setControlsError(error ?? null);
    setControlsLoading(false);
  }, []);

  return (
    <main className="relative min-h-screen">
      <div className="pointer-events-none fixed inset-0 grid-bg opacity-30" />
      <div className="relative mx-auto max-w-7xl px-4 py-6 md:px-8 md:py-10">
        <header className="mb-5 flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-accent-teal/40 bg-accent-teal/10 shadow-glow-teal">
            <Scale className="h-6 w-6 text-accent-teal" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight text-glow-blue md:text-2xl">
              Compliance{" "}
              <span className="text-accent-teal">Posture</span>
            </h1>
            <p className="font-mono text-[11px] uppercase tracking-[0.3em] text-muted-foreground">
              Framework scores · controls · violations
            </p>
          </div>
        </header>

        {/* R5 banner */}
        <div className="mb-6 flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-3">
          <Info className="mt-0.5 h-4 w-4 shrink-0 text-amber-300" />
          <p className="font-mono text-xs leading-relaxed text-amber-200">
            <span className="font-bold uppercase tracking-wider">R5 ·</span>{" "}
            Regional framework control IDs (NCA ECC, SAMA CSF, PDPL, DPDPA) are
            placeholders pending product-owner verification.
          </p>
        </div>

        {/* Framework cards */}
        <section className="mb-6">
          {frameworksLoading && (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
              {Array.from({ length: 8 }).map((_, i) => (
                <div
                  key={i}
                  className="h-32 animate-pulse rounded-xl border border-border bg-muted/20"
                />
              ))}
            </div>
          )}
          {!frameworksLoading && frameworksError && (
            <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
              No framework posture yet — {frameworksError}
            </div>
          )}
          {!frameworksLoading && !frameworksError && (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
              {frameworks.map((f) => {
                const key = String(f.framework);
                const active = selected === key;
                return (
                  <button
                    key={key}
                    onClick={() => void selectFramework(key)}
                    className={cn(
                      "group relative flex flex-col gap-2 rounded-xl border bg-card/60 p-3 text-left transition-all hover:bg-card",
                      active
                        ? "border-accent-blue/60 shadow-glow"
                        : "border-border hover:border-accent-blue/40"
                    )}
                  >
                    <div className="flex items-start justify-between gap-1">
                      <span className="font-mono text-[11px] font-semibold leading-tight text-foreground">
                        {f.displayName ?? key}
                      </span>
                      {f.reviewRequired && (
                        <span className="inline-flex shrink-0 items-center gap-0.5 rounded border border-amber-500/40 bg-amber-500/10 px-1 py-0.5 font-mono text-[8px] font-bold uppercase tracking-wider text-amber-300">
                          <AlertTriangle className="h-2.5 w-2.5" />
                          Review
                        </span>
                      )}
                    </div>
                    <div
                      className={cn(
                        "font-mono text-3xl font-bold",
                        scoreColor(f.score)
                      )}
                    >
                      {f.score}
                      <span className="text-sm text-muted-foreground">/100</span>
                    </div>
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted/40">
                      <div
                        className={cn("h-full rounded-full", scoreBar(f.score))}
                        style={{ width: `${Math.max(0, Math.min(100, f.score))}%` }}
                      />
                    </div>
                    <div className="flex items-center justify-between font-mono text-[10px] text-muted-foreground">
                      <span>{f.controlCount} controls</span>
                      <span
                        className={cn(
                          f.openViolations > 0 && "text-red-300"
                        )}
                      >
                        {f.openViolations} open
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </section>

        {/* Controls drill-down */}
        <section className="mb-6">
          <Card className="glass overflow-hidden">
            <CardHeader>
              <CardTitle>
                <ListChecks className="h-4 w-4 text-accent-blue" />
                {selected ? `Controls · ${selected}` : "Controls"}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {!selected && (
                <div className="flex items-center justify-center gap-2 rounded-lg border border-dashed border-border px-3 py-8 font-mono text-xs text-muted-foreground">
                  <ShieldQuestion className="h-4 w-4" />
                  Select a framework above to view its control mappings.
                </div>
              )}
              {selected && controlsLoading && (
                <div className="flex items-center justify-center gap-2 py-8 font-mono text-xs text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading controls…
                </div>
              )}
              {selected && !controlsLoading && controlsError && (
                <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
                  No controls — {controlsError}
                </div>
              )}
              {selected &&
                !controlsLoading &&
                !controlsError &&
                controls.length === 0 && (
                  <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
                    No control mappings for {selected}.
                  </div>
                )}
              {selected &&
                !controlsLoading &&
                !controlsError &&
                controls.length > 0 && (
                  <div className="space-y-2">
                    {controls.map((c, i) => (
                      <div
                        key={`${c.framework}-${c.controlId}-${i}`}
                        className="rounded-lg border border-border bg-background/40 px-3 py-2.5"
                      >
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <span className="font-mono text-xs font-bold text-accent-teal">
                            {c.controlId}
                          </span>
                          {c.reviewMarker && (
                            <span
                              title="Verbatim fixture review marker (Brief R5)"
                              className="inline-flex items-center gap-1 rounded border border-amber-500/40 bg-amber-500/10 px-1.5 py-0.5 font-mono text-[9px] font-semibold uppercase tracking-wider text-amber-300"
                            >
                              <AlertTriangle className="h-2.5 w-2.5" />
                              {c.reviewMarker}
                            </span>
                          )}
                        </div>
                        <p className="mt-1 text-sm text-foreground">{c.title}</p>
                        {c.description && (
                          <p className="mt-0.5 font-mono text-[11px] text-muted-foreground">
                            {c.description}
                          </p>
                        )}
                        {c.mappedCategories &&
                          c.mappedCategories.length > 0 && (
                            <div className="mt-1.5 flex flex-wrap gap-1">
                              {c.mappedCategories.map((cat) => (
                                <span
                                  key={String(cat)}
                                  className={cn(
                                    "rounded border px-1.5 py-0.5 font-mono text-[9px] font-semibold tracking-wide",
                                    categoryClass(String(cat))
                                  )}
                                >
                                  {String(cat)}
                                </span>
                              ))}
                            </div>
                          )}
                      </div>
                    ))}
                  </div>
                )}
            </CardContent>
          </Card>
        </section>

        {/* Violations table */}
        <section>
          <Card className="glass overflow-hidden">
            <CardHeader>
              <CardTitle className="justify-between">
                <span className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-red-400" />
                  Compliance Violations
                </span>
                <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                  {violations.length}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {violationsLoading && (
                <div className="h-40 animate-pulse rounded-lg border border-border bg-muted/20" />
              )}
              {!violationsLoading && violationsError && (
                <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
                  No violations yet — {violationsError}
                </div>
              )}
              {!violationsLoading &&
                !violationsError &&
                violations.length === 0 && (
                  <div className="rounded-lg border border-dashed border-border px-3 py-8 text-center font-mono text-xs text-muted-foreground">
                    No violations recorded.
                  </div>
                )}
              {!violationsLoading &&
                !violationsError &&
                violations.length > 0 && (
                  <div className="overflow-x-auto">
                    <table className="w-full border-collapse text-left">
                      <thead>
                        <tr className="border-b border-border font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                          <th className="px-2 py-2 font-semibold">Framework</th>
                          <th className="px-2 py-2 font-semibold">Control</th>
                          <th className="px-2 py-2 font-semibold">Category</th>
                          <th className="px-2 py-2 font-semibold">Severity</th>
                          <th className="px-2 py-2 font-semibold">User</th>
                          <th className="px-2 py-2 font-semibold">Time</th>
                        </tr>
                      </thead>
                      <tbody>
                        {violations.map((v) => (
                          <tr
                            key={v.violationId}
                            className="border-b border-border/60 font-mono text-xs hover:bg-muted/20"
                          >
                            <td className="px-2 py-2 font-semibold text-foreground">
                              {v.framework}
                            </td>
                            <td className="px-2 py-2 text-accent-teal">
                              {v.controlId}
                            </td>
                            <td className="px-2 py-2">
                              {v.category ? (
                                <span
                                  className={cn(
                                    "rounded border px-1.5 py-0.5 text-[9px] font-semibold",
                                    categoryClass(String(v.category))
                                  )}
                                >
                                  {String(v.category)}
                                </span>
                              ) : (
                                <span className="text-muted-foreground">—</span>
                              )}
                            </td>
                            <td className="px-2 py-2">
                              <span
                                className={cn(
                                  "rounded border px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider",
                                  severityClass(v.severity)
                                )}
                              >
                                {v.severity}
                              </span>
                            </td>
                            <td className="px-2 py-2 text-muted-foreground">
                              {v.userEmail ?? "—"}
                            </td>
                            <td className="px-2 py-2 text-muted-foreground">
                              {fmtTime(v.timestamp)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
            </CardContent>
          </Card>
        </section>

        <footer className="mt-8 text-center font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground/70">
          Live data via BFF · regional control IDs pending verification (R5)
        </footer>
      </div>
    </main>
  );
}
