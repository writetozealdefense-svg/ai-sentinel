import type { Decision } from "@/lib/types";

// Color + style maps for decision and risk.
export const decisionStyle: Record<
  Decision,
  { label: string; className: string; dot: string }
> = {
  ALLOW: {
    label: "ALLOW",
    className: "bg-emerald-500/15 text-emerald-300 border border-emerald-500/40",
    dot: "bg-emerald-400",
  },
  WARN: {
    label: "WARN",
    className: "bg-amber-500/15 text-amber-300 border border-amber-500/40",
    dot: "bg-amber-400",
  },
  REDACT: {
    label: "REDACT",
    className: "bg-purple-500/15 text-purple-300 border border-purple-500/40",
    dot: "bg-purple-400",
  },
  BLOCK: {
    label: "BLOCK",
    className: "bg-red-500/15 text-red-300 border border-red-500/50",
    dot: "bg-red-400",
  },
};

export function riskColor(score: number): string {
  if (score >= 75) return "text-red-400";
  if (score >= 50) return "text-amber-400";
  if (score >= 25) return "text-yellow-300";
  return "text-emerald-400";
}

export function riskStroke(score: number): string {
  if (score >= 75) return "hsl(0 84% 60%)";
  if (score >= 50) return "hsl(38 92% 55%)";
  if (score >= 25) return "hsl(50 90% 55%)";
  return "hsl(160 84% 45%)";
}

// Category chip styling (SOC live monitoring + compliance controls).
export function categoryClass(category: string): string {
  switch (category.toUpperCase()) {
    case "PHI":
      return "border-rose-500/40 bg-rose-500/10 text-rose-300";
    case "PCI":
      return "border-orange-500/40 bg-orange-500/10 text-orange-300";
    case "PII":
      return "border-sky-500/40 bg-sky-500/10 text-sky-300";
    case "SECRET":
      return "border-red-500/40 bg-red-500/10 text-red-300";
    case "CODE":
      return "border-violet-500/40 bg-violet-500/10 text-violet-300";
    default:
      return "border-border bg-muted/40 text-muted-foreground";
  }
}

// DLP severity badge styling (compliance violations).
export function severityClass(severity: string): string {
  switch (severity.toLowerCase()) {
    case "critical":
      return "border-red-500/50 bg-red-500/15 text-red-300";
    case "high":
      return "border-orange-500/50 bg-orange-500/15 text-orange-300";
    case "medium":
      return "border-amber-500/40 bg-amber-500/15 text-amber-300";
    case "low":
      return "border-sky-500/40 bg-sky-500/15 text-sky-300";
    default:
      return "border-border bg-muted/40 text-muted-foreground";
  }
}

// Runtime event priority color (Falco/Tetragon).
// critical/error = red, warning = amber, notice/info = blue.
export function priorityClass(priority: string): string {
  switch (priority.toLowerCase()) {
    case "critical":
    case "error":
      return "border-red-500/50 bg-red-500/15 text-red-300";
    case "warning":
      return "border-amber-500/40 bg-amber-500/15 text-amber-300";
    case "notice":
    case "info":
    default:
      return "border-sky-500/40 bg-sky-500/15 text-sky-300";
  }
}

// Short, locale-stable HH:MM:SS for monitoring rows.
export function fmtTime(ts?: string): string {
  if (!ts) return "—";
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return ts;
  return d.toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}
