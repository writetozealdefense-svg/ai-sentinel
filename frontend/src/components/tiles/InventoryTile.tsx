"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SimulatedTag } from "@/components/SimulatedTag";
import type { AiInventoryItem } from "@/lib/types";
import { Boxes, ShieldAlert, ShieldCheck, Users } from "lucide-react";

export function InventoryTile({
  items,
  error,
  loading,
}: {
  items: AiInventoryItem[];
  error?: string | null;
  loading?: boolean;
}) {
  return (
    <Card className="glass overflow-hidden">
      <CardHeader>
        <CardTitle className="justify-between">
          <span className="flex items-center gap-2">
            <Boxes className="h-4 w-4 text-accent-blue" />
            AI Inventory
          </span>
          <SimulatedTag />
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {loading && <SkeletonRows />}
        {!loading && error && (
          <EmptyState text={`No data yet — ${error}`} />
        )}
        {!loading && !error && items.length === 0 && (
          <EmptyState text="No AI usage discovered yet. Run Scenario 1." />
        )}
        {!loading &&
          !error &&
          items.map((it) => (
            <div
              key={it.provider}
              className="flex items-center justify-between rounded-lg border border-border bg-background/40 px-3 py-2.5"
            >
              <div className="flex flex-col">
                <span className="font-mono text-sm font-semibold capitalize">
                  {it.provider}
                </span>
                <span className="flex items-center gap-3 font-mono text-[11px] text-muted-foreground">
                  <span>{it.usageCount} uses</span>
                  {typeof it.uniqueUsers === "number" && (
                    <span className="flex items-center gap-1">
                      <Users className="h-3 w-3" />
                      {it.uniqueUsers}
                    </span>
                  )}
                </span>
              </div>
              {it.sanctioned ? (
                <Badge className="gap-1 border-emerald-500/40 bg-emerald-500/15 text-emerald-300">
                  <ShieldCheck className="h-3 w-3" />
                  SANCTIONED
                </Badge>
              ) : (
                <Badge className="animate-pulseGlow gap-1 border-red-500/50 bg-red-500/15 text-red-300">
                  <ShieldAlert className="h-3 w-3" />
                  SHADOW AI
                </Badge>
              )}
            </div>
          ))}
      </CardContent>
    </Card>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <div className="rounded-lg border border-dashed border-border px-3 py-6 text-center font-mono text-xs text-muted-foreground">
      {text}
    </div>
  );
}

function SkeletonRows() {
  return (
    <>
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="h-12 animate-pulse rounded-lg border border-border bg-muted/30"
        />
      ))}
    </>
  );
}
