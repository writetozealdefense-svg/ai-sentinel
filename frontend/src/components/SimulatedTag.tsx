import { cn } from "@/lib/utils";

// Small badge marking telemetry-derived / synthetic UI elements (Brief R4).
export function SimulatedTag({ className }: { className?: string }) {
  return (
    <span
      title="Synthetic / simulated telemetry — no live integrations on the compose path"
      className={cn(
        "inline-flex items-center gap-1 rounded border border-accent-teal/40 bg-accent-teal/10 px-1.5 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-widest text-accent-teal",
        className
      )}
    >
      <span className="h-1.5 w-1.5 animate-pulseGlow rounded-full bg-accent-teal" />
      Simulated
    </span>
  );
}
