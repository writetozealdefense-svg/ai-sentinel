"use client";

import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { InventoryTile } from "@/components/tiles/InventoryTile";
import { InspectionTile } from "@/components/tiles/InspectionTile";
import { ViolationsTile } from "@/components/tiles/ViolationsTile";
import type {
  AiInventoryItem,
  FrameworkScore,
  InspectResult,
} from "@/lib/types";
import { Play, Loader2, Activity } from "lucide-react";

async function getJson<T>(
  url: string,
  init?: RequestInit
): Promise<{ data?: T; error?: string }> {
  try {
    const res = await fetch(url, { ...init, cache: "no-store" });
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

export function Dashboard() {
  const [inventory, setInventory] = useState<AiInventoryItem[]>([]);
  const [inventoryError, setInventoryError] = useState<string | null>(null);
  const [inventoryLoading, setInventoryLoading] = useState(true);

  const [frameworks, setFrameworks] = useState<FrameworkScore[]>([]);
  const [frameworksError, setFrameworksError] = useState<string | null>(null);
  const [frameworksLoading, setFrameworksLoading] = useState(true);

  const [inspection, setInspection] = useState<InspectResult | null>(null);
  const [inspectionError, setInspectionError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);

  const loadInventory = useCallback(async () => {
    setInventoryLoading(true);
    const { data, error } = await getJson<AiInventoryItem[]>("/api/inventory");
    setInventory(Array.isArray(data) ? data : []);
    setInventoryError(error ?? null);
    setInventoryLoading(false);
  }, []);

  const loadFrameworks = useCallback(async () => {
    setFrameworksLoading(true);
    const { data, error } = await getJson<FrameworkScore[]>("/api/frameworks");
    setFrameworks(Array.isArray(data) ? data : []);
    setFrameworksError(error ?? null);
    setFrameworksLoading(false);
  }, []);

  useEffect(() => {
    void loadInventory();
    void loadFrameworks();
  }, [loadInventory, loadFrameworks]);

  const runScenario = useCallback(async () => {
    setRunning(true);
    setInspectionError(null);
    const { data, error } = await getJson<InspectResult>("/api/scenario1", {
      method: "POST",
    });
    if (error || !data) {
      setInspectionError(error ?? "no result returned");
      setInspection(null);
    } else {
      setInspection(data);
      setInspectionError(null);
    }
    setRunning(false);
    // Refresh telemetry-derived tiles after the run.
    void loadInventory();
    void loadFrameworks();
  }, [loadInventory, loadFrameworks]);

  return (
    <main className="relative min-h-screen">
      <div className="pointer-events-none fixed inset-0 grid-bg opacity-30" />
      <div className="relative mx-auto max-w-7xl px-4 py-6 md:px-8 md:py-10">
        {/* Header */}
        <header className="mb-8 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-glow-blue md:text-2xl">
              Executive Dashboard
            </h1>
            <p className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.3em] text-muted-foreground">
              <Activity className="h-3 w-3 text-accent-teal" />
              Posture Overview · Phase 1
            </p>
          </div>

          <Button
            size="lg"
            onClick={runScenario}
            disabled={running}
            className="group relative font-mono uppercase tracking-wider"
          >
            {running ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Inspecting…
              </>
            ) : (
              <>
                <Play className="h-4 w-4 fill-current" />
                Run Scenario 1
              </>
            )}
          </Button>
        </header>

        <p className="mb-6 rounded-lg border border-border bg-card/40 px-4 py-2.5 font-mono text-xs text-muted-foreground">
          <span className="text-accent-blue">SCENARIO 1 ·</span> Employee{" "}
          <span className="text-foreground">james.wilson@example.com</span>{" "}
          uploads{" "}
          <span className="text-foreground">customer_database.csv</span> to{" "}
          <span className="text-red-300">ChatGPT</span> (unsanctioned).
        </p>

        {/* Tiles */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <div className="lg:col-span-1">
            <InventoryTile
              items={inventory}
              error={inventoryError}
              loading={inventoryLoading}
            />
          </div>
          <div className="lg:col-span-2">
            <InspectionTile result={inspection} error={inspectionError} />
          </div>
          <div className="lg:col-span-3">
            <ViolationsTile
              frameworks={frameworks}
              error={frameworksError}
              loading={frameworksLoading}
            />
          </div>
        </div>

        <footer className="mt-8 text-center font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground/70">
          Discovery telemetry is SIMULATED · live data via BFF at /api/*
        </footer>
      </div>
    </main>
  );
}
