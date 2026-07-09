"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { FrameworkScore } from "@/lib/types";
import { Scale, AlertTriangle } from "lucide-react";

interface Row {
  framework: string;
  openViolations: number;
  score: number;
  reviewRequired: boolean;
}

function CustomTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: Array<{ payload: Row }>;
}) {
  if (!active || !payload?.length) return null;
  const r = payload[0].payload;
  return (
    <div className="rounded-md border border-border bg-card/95 px-3 py-2 font-mono text-xs shadow-glow">
      <div className="font-semibold">{r.framework}</div>
      <div className="text-red-300">open violations: {r.openViolations}</div>
      <div className="text-accent-teal">score: {r.score}</div>
      {r.reviewRequired && (
        <div className="text-amber-300">REVIEW PENDING</div>
      )}
    </div>
  );
}

export function ViolationsTile({
  frameworks,
  error,
  loading,
}: {
  frameworks: FrameworkScore[];
  error?: string | null;
  loading?: boolean;
}) {
  const data: Row[] = (frameworks ?? []).map((f) => ({
    framework: f.displayName ?? String(f.framework),
    openViolations: f.openViolations ?? 0,
    score: f.score ?? 0,
    reviewRequired: Boolean(f.reviewRequired),
  }));

  const barColor = (v: number) =>
    v >= 3 ? "hsl(0 84% 60%)" : v >= 1 ? "hsl(38 92% 55%)" : "hsl(217 91% 60%)";

  return (
    <Card className="glass overflow-hidden">
      <CardHeader>
        <CardTitle className="justify-between">
          <span className="flex items-center gap-2">
            <Scale className="h-4 w-4 text-accent-teal" />
            Open Violations by Framework
          </span>
          {data.some((d) => d.reviewRequired) && (
            <span className="inline-flex items-center gap-1 rounded border border-amber-500/40 bg-amber-500/10 px-1.5 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-widest text-amber-300">
              <AlertTriangle className="h-3 w-3" />
              Review Pending
            </span>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {loading && (
          <div className="h-[260px] animate-pulse rounded-lg border border-border bg-muted/20" />
        )}
        {!loading && error && (
          <div className="flex h-[260px] items-center justify-center rounded-lg border border-dashed border-border font-mono text-xs text-muted-foreground">
            No data yet — {error}
          </div>
        )}
        {!loading && !error && data.length === 0 && (
          <div className="flex h-[260px] items-center justify-center rounded-lg border border-dashed border-border font-mono text-xs text-muted-foreground">
            No framework posture yet.
          </div>
        )}
        {!loading && !error && data.length > 0 && (
          <ResponsiveContainer width="100%" height={260}>
            <BarChart
              data={data}
              margin={{ top: 8, right: 8, bottom: 8, left: -16 }}
            >
              <CartesianGrid
                strokeDasharray="3 3"
                stroke="hsl(var(--border))"
                vertical={false}
              />
              <XAxis
                dataKey="framework"
                tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 10 }}
                interval={0}
                angle={-25}
                textAnchor="end"
                height={56}
                stroke="hsl(var(--border))"
              />
              <YAxis
                allowDecimals={false}
                tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 10 }}
                stroke="hsl(var(--border))"
              />
              <Tooltip
                content={<CustomTooltip />}
                cursor={{ fill: "hsl(var(--muted) / 0.3)" }}
              />
              <Bar dataKey="openViolations" radius={[4, 4, 0, 0]}>
                {data.map((d, i) => (
                  <Cell
                    key={i}
                    fill={barColor(d.openViolations)}
                    stroke={d.reviewRequired ? "hsl(38 92% 55%)" : "none"}
                    strokeWidth={d.reviewRequired ? 2 : 0}
                    strokeDasharray={d.reviewRequired ? "4 2" : undefined}
                  />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}
