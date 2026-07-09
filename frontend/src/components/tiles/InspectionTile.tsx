"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RiskGauge } from "@/components/RiskGauge";
import { decisionStyle } from "@/lib/decision";
import { cn } from "@/lib/utils";
import type { InspectResult, Category } from "@/lib/types";
import { ScanSearch, FileWarning } from "lucide-react";

function summarizeByCategory(result: InspectResult): Record<string, number> {
  const acc: Record<string, number> = {};
  for (const f of result.findings ?? []) {
    const inc = typeof f.count === "number" ? f.count : 1;
    acc[f.category] = (acc[f.category] ?? 0) + inc;
  }
  return acc;
}

const catColor: Record<Category, string> = {
  PII: "text-accent-blue border-accent-blue/40 bg-accent-blue/10",
  PHI: "text-pink-300 border-pink-500/40 bg-pink-500/10",
  PCI: "text-amber-300 border-amber-500/40 bg-amber-500/10",
  SECRET: "text-red-300 border-red-500/40 bg-red-500/10",
  CODE: "text-accent-teal border-accent-teal/40 bg-accent-teal/10",
  OTHER: "text-muted-foreground border-border bg-muted/30",
};

export function InspectionTile({
  result,
  error,
}: {
  result: InspectResult | null;
  error?: string | null;
}) {
  const style = result ? decisionStyle[result.decision] : null;
  const byCat = result ? summarizeByCategory(result) : {};

  return (
    <Card className="glass overflow-hidden">
      <CardHeader>
        <CardTitle>
          <ScanSearch className="h-4 w-4 text-accent-purple" />
          Latest Inspection
        </CardTitle>
      </CardHeader>
      <CardContent>
        {!result && (
          <div className="rounded-lg border border-dashed border-border px-3 py-10 text-center font-mono text-xs text-muted-foreground">
            {error
              ? `No inspection yet — ${error}`
              : "No inspection yet. Press “Run Scenario 1”."}
          </div>
        )}

        {result && style && (
          <div className="space-y-4">
            <div className="flex items-center justify-between gap-4">
              <div className="flex flex-col gap-2">
                <span
                  className={cn(
                    "inline-flex w-fit items-center gap-2 rounded-md px-3 py-1 font-mono text-sm font-bold tracking-widest",
                    style.className
                  )}
                >
                  <span
                    className={cn(
                      "h-2 w-2 animate-pulseGlow rounded-full",
                      style.dot
                    )}
                  />
                  {style.label}
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {result.provider ?? "—"} · {result.artifactName ?? "—"}
                </span>
              </div>
              <RiskGauge score={result.riskScore} />
            </div>

            <div>
              <div className="mb-1.5 flex items-center gap-1.5 font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
                <FileWarning className="h-3 w-3" />
                Findings by category
              </div>
              <div className="flex flex-wrap gap-1.5">
                {Object.keys(byCat).length === 0 && (
                  <span className="font-mono text-xs text-muted-foreground">
                    No findings.
                  </span>
                )}
                {Object.entries(byCat).map(([cat, n]) => (
                  <span
                    key={cat}
                    className={cn(
                      "rounded-md border px-2 py-0.5 font-mono text-xs font-semibold",
                      catColor[(cat as Category) in catColor ? (cat as Category) : "OTHER"]
                    )}
                  >
                    {cat} ×{n}
                  </span>
                ))}
              </div>
            </div>

            {result.redactedPreview && (
              <div>
                <div className="mb-1.5 font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
                  Redacted preview
                </div>
                <pre className="max-h-40 overflow-auto rounded-lg border border-border bg-background/60 p-3 font-mono text-[11px] leading-relaxed text-foreground/90 whitespace-pre-wrap break-words">
                  {result.redactedPreview}
                </pre>
              </div>
            )}

            {result.policyTrace && (
              <div className="rounded-lg border border-border bg-background/40 px-3 py-2 font-mono text-[11px] text-muted-foreground">
                <span className="text-accent-purple">OPA:</span>{" "}
                {result.policyTrace}
              </div>
            )}

            <div className="flex flex-wrap gap-x-4 gap-y-1 font-mono text-[10px] text-muted-foreground">
              <span>id: {result.promptId}</span>
              {result.violationId && (
                <span className="text-red-300">
                  violation: {result.violationId}
                </span>
              )}
              <span>{new Date(result.timestamp).toLocaleString()}</span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
