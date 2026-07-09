import { riskStroke, riskColor } from "@/lib/decision";
import { cn } from "@/lib/utils";

// Compact circular risk gauge (0-100). Pure SVG, no deps.
export function RiskGauge({ score, size = 96 }: { score: number; size?: number }) {
  const clamped = Math.max(0, Math.min(100, score));
  const stroke = 8;
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const offset = c - (clamped / 100) * c;
  const color = riskStroke(clamped);

  return (
    <div className="relative inline-flex items-center justify-center">
      <svg width={size} height={size} className="-rotate-90">
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
          style={{ filter: `drop-shadow(0 0 6px ${color})` }}
        />
      </svg>
      <div className="absolute flex flex-col items-center">
        <span className={cn("font-mono text-2xl font-bold", riskColor(clamped))}>
          {clamped}
        </span>
        <span className="font-mono text-[9px] uppercase tracking-widest text-muted-foreground">
          risk
        </span>
      </div>
    </div>
  );
}
