"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  forceCollide,
  forceLink,
  forceManyBody,
  forceSimulation,
  forceX,
  forceY,
  type Simulation,
  type SimulationLinkDatum,
  type SimulationNodeDatum,
} from "d3-force";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { GraphEdge, GraphNode, GraphResult, NamedQuery } from "@/lib/types";
import { Network, Play, Terminal, Table2, Loader2 } from "lucide-react";

// ---------------------------------------------------------------------------
// Simulation datum types (GraphNode/GraphEdge from graph.yaml + d3 kinematics)
// ---------------------------------------------------------------------------

interface SimNode extends SimulationNodeDatum {
  id: string;
  label: string;
  properties?: Record<string, unknown>;
}

interface SimLink extends SimulationLinkDatum<SimNode> {
  type: string;
}

// Node palette per contract label (spec: User=teal, Prompt=blue, Model=purple,
// Agent=orange, Tool=slate, Dataset=cyan, Violation=red).
const NODE_COLORS: Record<string, string> = {
  User: "#2dd4bf",
  Prompt: "#3b82f6",
  Model: "#a855f7",
  Agent: "#fb923c",
  Tool: "#94a3b8",
  Dataset: "#22d3ee",
  Violation: "#ef4444",
};

const NODE_RADIUS: Record<string, number> = {
  User: 9,
  Prompt: 6,
  Model: 9,
  Agent: 10,
  Tool: 6,
  Dataset: 7,
  Violation: 8,
};

const EDGE_TYPES = ["USED", "ACCESSED", "GENERATED", "EXPOSED"] as const;

// alphaTarget kept above alphaMin so the simulation never fully settles —
// this is what produces the gentle continuous ambient motion.
const AMBIENT_ALPHA = 0.02;
const RISK_THRESHOLD = 70;
const DEFAULT_QUERY = "full-graph";

function nodeColor(label: string): string {
  return NODE_COLORS[label] ?? "#94a3b8";
}

function nodeRadius(label: string): number {
  return NODE_RADIUS[label] ?? 7;
}

// Violations and high-risk nodes get the pulsing glow.
function isHot(n: SimNode): boolean {
  if (n.label === "Violation") return true;
  const risk = n.properties?.riskScore;
  return typeof risk === "number" && risk >= RISK_THRESHOLD;
}

function nodeName(n: SimNode): string {
  const p = n.properties ?? {};
  const cand = p.email ?? p.name ?? p.provider ?? p.id ?? n.id;
  const s = String(cand);
  return s.length > 22 ? `${s.slice(0, 21)}…` : s;
}

export default function GraphPage() {
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const simRef = useRef<Simulation<SimNode, SimLink> | null>(null);
  const nodesRef = useRef<SimNode[]>([]);
  const linksRef = useRef<SimLink[]>([]);
  const sizeRef = useRef({ w: 800, h: 520 });
  const dragRef = useRef<SimNode | null>(null);

  const [queries, setQueries] = useState<NamedQuery[]>([]);
  const [queriesError, setQueriesError] = useState<string | null>(null);
  const [selected, setSelected] = useState(DEFAULT_QUERY);
  const [result, setResult] = useState<GraphResult | null>(null);
  const [activeQuery, setActiveQuery] = useState<string | null>(null);
  const [runError, setRunError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);

  // ---- simulation rebuild (position-preserving, so re-queries animate) ----
  const rebuildSimulation = useCallback(
    (nodes: GraphNode[], edges: GraphEdge[]) => {
      const { w, h } = sizeRef.current;
      const prev = new Map(nodesRef.current.map((n) => [n.id, n]));
      const simNodes: SimNode[] = nodes.map((n) => {
        const old = prev.get(n.id);
        return {
          id: n.id,
          label: n.label,
          properties: n.properties,
          // Existing nodes keep their position; new nodes spawn near center
          // and the forces animate them outward.
          x: old?.x ?? w / 2 + (Math.random() - 0.5) * 60,
          y: old?.y ?? h / 2 + (Math.random() - 0.5) * 60,
          vx: old?.vx ?? 0,
          vy: old?.vy ?? 0,
        };
      });
      const ids = new Set(simNodes.map((n) => n.id));
      const simLinks: SimLink[] = edges
        .filter((e) => ids.has(e.source) && ids.has(e.target))
        .map((e) => ({ source: e.source, target: e.target, type: e.type }));

      nodesRef.current = simNodes;
      linksRef.current = simLinks;

      simRef.current?.stop();
      simRef.current = forceSimulation<SimNode>(simNodes)
        .force(
          "link",
          forceLink<SimNode, SimLink>(simLinks)
            .id((d) => d.id)
            .distance(88)
            .strength(0.5)
        )
        .force("charge", forceManyBody().strength(-230))
        .force("collide", forceCollide(24))
        .force("x", forceX(w / 2).strength(0.05))
        .force("y", forceY(h / 2).strength(0.06))
        .alpha(1)
        .alphaMin(0.0001)
        .alphaDecay(0.022)
        .alphaTarget(AMBIENT_ALPHA)
        .restart();
    },
    []
  );

  // ---- run a named query through the BFF ----
  const runQuery = useCallback(
    async (name: string) => {
      setRunning(true);
      setRunError(null);
      try {
        const res = await fetch(`/api/graph/query/${encodeURIComponent(name)}`, {
          method: "POST",
          headers: { "content-type": "application/json" },
          body: "{}",
          cache: "no-store",
        });
        const body = (await res.json().catch(() => null)) as
          | GraphResult
          | { error?: string; detail?: string }
          | null;
        if (!res.ok || !body || !("nodes" in body)) {
          const msg =
            (body && "error" in body && body.error) ||
            `query failed (${res.status})`;
          setRunError(String(msg));
          return;
        }
        setResult(body);
        setActiveQuery(name);
        rebuildSimulation(body.nodes ?? [], body.edges ?? []);
      } catch (e) {
        setRunError(e instanceof Error ? e.message : "network error");
      } finally {
        setRunning(false);
      }
    },
    [rebuildSimulation]
  );

  // ---- initial load: named queries list + default full-graph view ----
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch("/api/graph/queries", { cache: "no-store" });
        const body = await res.json().catch(() => null);
        if (cancelled) return;
        if (!res.ok) {
          setQueriesError(
            (body && (body.error as string)) || `request failed (${res.status})`
          );
        } else if (Array.isArray(body)) {
          setQueries(body as NamedQuery[]);
          setQueriesError(null);
        }
      } catch (e) {
        if (!cancelled)
          setQueriesError(e instanceof Error ? e.message : "network error");
      }
    })();
    // ?q=<name> deep-links a named query (demo runbook); default = full-graph.
    const requested = new URLSearchParams(window.location.search).get("q");
    void runQuery(requested || DEFAULT_QUERY);
    return () => {
      cancelled = true;
    };
  }, [runQuery]);

  // ---- canvas: resize handling + render loop + drag ----
  useEffect(() => {
    const container = containerRef.current;
    const canvas = canvasRef.current;
    if (!container || !canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const resize = () => {
      const rect = container.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;
      sizeRef.current = { w: rect.width, h: rect.height };
      canvas.width = Math.max(1, Math.round(rect.width * dpr));
      canvas.height = Math.max(1, Math.round(rect.height * dpr));
      canvas.style.width = `${rect.width}px`;
      canvas.style.height = `${rect.height}px`;
      const sim = simRef.current;
      if (sim) {
        sim.force("x", forceX<SimNode>(rect.width / 2).strength(0.05));
        sim.force("y", forceY<SimNode>(rect.height / 2).strength(0.06));
        sim.alpha(Math.max(sim.alpha(), 0.2)).restart();
      }
    };
    resize();
    const ro = new ResizeObserver(resize);
    ro.observe(container);

    let raf = 0;
    const draw = () => {
      const dpr = window.devicePixelRatio || 1;
      const { w, h } = sizeRef.current;
      const now = performance.now();
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);

      const links = linksRef.current;
      const nodes = nodesRef.current;

      // ---- edges + arrowheads + type labels ----
      ctx.font = "8px ui-monospace, monospace";
      for (const l of links) {
        const s = l.source as SimNode;
        const t = l.target as SimNode;
        if (
          typeof s !== "object" ||
          typeof t !== "object" ||
          s.x == null ||
          s.y == null ||
          t.x == null ||
          t.y == null
        )
          continue;
        const dx = t.x - s.x;
        const dy = t.y - s.y;
        const dist = Math.hypot(dx, dy) || 1;
        const ux = dx / dist;
        const uy = dy / dist;
        const rT = nodeRadius(t.label) + 3;
        const ex = t.x - ux * rT;
        const ey = t.y - uy * rT;

        ctx.strokeStyle = "rgba(100,116,139,0.32)";
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(s.x, s.y);
        ctx.lineTo(ex, ey);
        ctx.stroke();

        // subtle arrowhead
        const ah = 5;
        ctx.fillStyle = "rgba(148,163,184,0.5)";
        ctx.beginPath();
        ctx.moveTo(ex, ey);
        ctx.lineTo(ex - ux * ah - uy * (ah * 0.55), ey - uy * ah + ux * (ah * 0.55));
        ctx.lineTo(ex - ux * ah + uy * (ah * 0.55), ey - uy * ah - ux * (ah * 0.55));
        ctx.closePath();
        ctx.fill();

        // edge type label at midpoint
        ctx.fillStyle = "rgba(148,163,184,0.55)";
        ctx.textAlign = "center";
        ctx.fillText(l.type, (s.x + t.x) / 2, (s.y + t.y) / 2 - 3);
      }

      // ---- nodes ----
      for (let i = 0; i < nodes.length; i++) {
        const n = nodes[i];
        if (n.x == null || n.y == null) continue;
        const r = nodeRadius(n.label);
        const color = nodeColor(n.label);
        const hot = isHot(n);

        ctx.save();
        if (hot) {
          // pulsing glow for Violation / high-risk nodes
          const pulse = 0.5 + 0.5 * Math.sin(now / 420 + i * 1.7);
          ctx.shadowColor = color;
          ctx.shadowBlur = 10 + 14 * pulse;
          ctx.strokeStyle = color;
          ctx.globalAlpha = 0.28 + 0.3 * pulse;
          ctx.lineWidth = 1.5;
          ctx.beginPath();
          ctx.arc(n.x, n.y, r + 4 + 3 * pulse, 0, Math.PI * 2);
          ctx.stroke();
          ctx.globalAlpha = 1;
        } else {
          ctx.shadowColor = color;
          ctx.shadowBlur = 6;
        }
        ctx.fillStyle = color;
        ctx.beginPath();
        ctx.arc(n.x, n.y, r, 0, Math.PI * 2);
        ctx.fill();
        ctx.restore();

        // label
        ctx.font = "9px ui-monospace, monospace";
        ctx.fillStyle = "rgba(178,190,205,0.85)";
        ctx.textAlign = "center";
        ctx.fillText(nodeName(n), n.x, n.y + r + 11);
      }

      raf = requestAnimationFrame(draw);
    };
    raf = requestAnimationFrame(draw);

    // ---- drag (optional interaction) ----
    const findNode = (x: number, y: number): SimNode | null => {
      for (const n of nodesRef.current) {
        if (n.x == null || n.y == null) continue;
        if (Math.hypot(n.x - x, n.y - y) <= nodeRadius(n.label) + 6) return n;
      }
      return null;
    };
    const toLocal = (e: PointerEvent) => {
      const rect = canvas.getBoundingClientRect();
      return { x: e.clientX - rect.left, y: e.clientY - rect.top };
    };
    const onDown = (e: PointerEvent) => {
      const { x, y } = toLocal(e);
      const n = findNode(x, y);
      if (!n) return;
      dragRef.current = n;
      n.fx = x;
      n.fy = y;
      canvas.setPointerCapture(e.pointerId);
      simRef.current?.alphaTarget(0.25).restart();
    };
    const onMove = (e: PointerEvent) => {
      const n = dragRef.current;
      const { x, y } = toLocal(e);
      canvas.style.cursor = n || findNode(x, y) ? "grab" : "default";
      if (!n) return;
      n.fx = x;
      n.fy = y;
    };
    const onUp = (e: PointerEvent) => {
      const n = dragRef.current;
      if (!n) return;
      n.fx = null;
      n.fy = null;
      dragRef.current = null;
      try {
        canvas.releasePointerCapture(e.pointerId);
      } catch {
        /* already released */
      }
      simRef.current?.alphaTarget(AMBIENT_ALPHA);
    };
    canvas.addEventListener("pointerdown", onDown);
    canvas.addEventListener("pointermove", onMove);
    canvas.addEventListener("pointerup", onUp);
    canvas.addEventListener("pointercancel", onUp);

    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
      canvas.removeEventListener("pointerdown", onDown);
      canvas.removeEventListener("pointermove", onMove);
      canvas.removeEventListener("pointerup", onUp);
      canvas.removeEventListener("pointercancel", onUp);
      simRef.current?.stop();
    };
  }, []);

  const hasGraph = (result?.nodes.length ?? 0) > 0;
  const rows = result?.rows ?? [];
  const columns =
    rows.length > 0
      ? Array.from(
          rows.reduce((set, r) => {
            Object.keys(r).forEach((k) => set.add(k));
            return set;
          }, new Set<string>())
        )
      : [];

  return (
    <main className="relative min-h-screen">
      <div className="pointer-events-none fixed inset-0 grid-bg opacity-30" />
      <div className="relative mx-auto max-w-7xl px-4 py-6 md:px-8 md:py-10">
        <header className="mb-6 flex flex-col gap-2">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-accent-purple/40 bg-accent-purple/10 shadow-glow-purple">
              <Network className="h-6 w-6 text-accent-purple" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight text-glow-blue md:text-2xl">
                Risk <span className="text-accent-purple text-glow-purple">Graph</span>
              </h1>
              <p className="font-mono text-[11px] uppercase tracking-[0.3em] text-muted-foreground">
                Neo4j exposure paths · live force-directed view
              </p>
            </div>
          </div>
        </header>

        {/* query controls */}
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <select
            value={selected}
            onChange={(e) => setSelected(e.target.value)}
            className="h-8 rounded-md border border-border bg-card px-2 font-mono text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            aria-label="Named graph query"
          >
            {queries.length === 0 && (
              <option value={DEFAULT_QUERY}>{DEFAULT_QUERY}</option>
            )}
            {queries.map((q) => (
              <option key={q.name} value={q.name} title={q.description}>
                {q.name}
              </option>
            ))}
          </select>
          <Button
            size="sm"
            disabled={running}
            onClick={() => void runQuery(selected)}
          >
            {running ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Play className="h-3.5 w-3.5" />
            )}
            Run
          </Button>
          {activeQuery && (
            <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
              showing: <span className="text-accent-teal">{activeQuery}</span>
            </span>
          )}
          {queriesError && (
            <span className="font-mono text-[10px] text-amber-300/90">
              query list unavailable — {queriesError}
            </span>
          )}
        </div>

        {runError && (
          <div className="mb-4 rounded-lg border border-red-500/40 bg-red-500/10 px-3 py-2 font-mono text-xs text-red-300">
            graph-svc error — {runError}
          </div>
        )}

        {/* animated canvas */}
        <Card className="glass overflow-hidden">
          <CardContent className="relative p-0">
            <div ref={containerRef} className="relative h-[520px] w-full">
              <canvas ref={canvasRef} className="block h-full w-full" />
              {!hasGraph && !running && (
                <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
                  <p className="rounded-lg border border-dashed border-border bg-background/60 px-4 py-3 text-center font-mono text-xs text-muted-foreground">
                    {runError
                      ? "Graph service unreachable. Start graph-svc and retry."
                      : "No graph data yet. Run scenarios to populate the risk graph."}
                  </p>
                </div>
              )}
              {/* legend */}
              <div className="pointer-events-none absolute bottom-3 left-3 rounded-lg border border-border bg-background/70 px-3 py-2 backdrop-blur">
                <div className="flex flex-wrap gap-x-3 gap-y-1">
                  {Object.entries(NODE_COLORS).map(([label, color]) => (
                    <span
                      key={label}
                      className="flex items-center gap-1.5 font-mono text-[9px] uppercase tracking-wider text-muted-foreground"
                    >
                      <span
                        className="h-2 w-2 rounded-full"
                        style={{ backgroundColor: color, boxShadow: `0 0 5px ${color}` }}
                      />
                      {label}
                    </span>
                  ))}
                </div>
                <div className="mt-1 font-mono text-[9px] uppercase tracking-wider text-muted-foreground/70">
                  edges: {EDGE_TYPES.join(" · ")} · glow = violation / risk ≥{" "}
                  {RISK_THRESHOLD}
                </div>
              </div>
              {hasGraph && (
                <div className="pointer-events-none absolute right-3 top-3 rounded border border-border bg-background/70 px-2 py-1 font-mono text-[9px] uppercase tracking-widest text-muted-foreground">
                  {result?.nodes.length} nodes · {result?.edges.length} edges
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* executed cypher (verbatim, for authenticity) */}
        <Card className="glass mt-4 overflow-hidden">
          <CardHeader className="pb-2">
            <CardTitle>
              <Terminal className="h-4 w-4 text-accent-teal" />
              Executed Cypher
            </CardTitle>
          </CardHeader>
          <CardContent>
            {result?.cypher ? (
              <pre className="overflow-x-auto rounded-lg border border-border bg-background/60 p-3 font-mono text-[11px] leading-relaxed text-accent-teal">
                {result.cypher}
              </pre>
            ) : (
              <p className="font-mono text-xs text-muted-foreground">
                No query executed yet.
              </p>
            )}
          </CardContent>
        </Card>

        {/* tabular rows */}
        <Card className="glass mt-4 overflow-hidden">
          <CardHeader className="pb-2">
            <CardTitle className="justify-between">
              <span className="flex items-center gap-2">
                <Table2 className="h-4 w-4 text-accent-blue" />
                Result Rows
              </span>
              <span className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                {rows.length} row{rows.length === 1 ? "" : "s"}
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {rows.length === 0 ? (
              <p className="rounded-lg border border-dashed border-border px-3 py-6 text-center font-mono text-xs text-muted-foreground">
                Query returned no rows.
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full border-collapse text-left">
                  <thead>
                    <tr className="border-b border-border">
                      {columns.map((c) => (
                        <th
                          key={c}
                          className="px-3 py-2 font-mono text-[10px] font-bold uppercase tracking-widest text-muted-foreground"
                        >
                          {c}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((r, i) => (
                      <tr
                        key={i}
                        className="border-b border-border/50 hover:bg-muted/20"
                      >
                        {columns.map((c) => {
                          const v = r[c];
                          return (
                            <td
                              key={c}
                              className="px-3 py-1.5 font-mono text-[11px] text-foreground/90"
                            >
                              {v == null
                                ? "—"
                                : typeof v === "object"
                                  ? JSON.stringify(v)
                                  : String(v)}
                            </td>
                          );
                        })}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        <footer className="mt-8 text-center font-mono text-[10px] uppercase tracking-[0.25em] text-muted-foreground/70">
          Graph data live from graph-svc (Neo4j) via BFF · drag nodes to explore
        </footer>
      </div>
    </main>
  );
}
