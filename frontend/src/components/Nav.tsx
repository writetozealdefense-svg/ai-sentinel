"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import Image from "next/image";
import { cn } from "@/lib/utils";
import {
  Activity,
  Radar,
  Scale,
  Bot,
  Network,
  Sparkles,
} from "lucide-react";

const LINKS = [
  { href: "/", label: "Executive", icon: Activity },
  { href: "/soc", label: "SOC", icon: Radar },
  { href: "/compliance", label: "Compliance", icon: Scale },
  { href: "/agents", label: "Agents", icon: Bot },
  { href: "/graph", label: "Risk Graph", icon: Network },
  { href: "/copilot", label: "Copilot", icon: Sparkles },
] as const;

export function Nav() {
  const pathname = usePathname();

  return (
    <header className="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur-md">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 md:px-8">
        <Link href="/" className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-accent-blue/40 bg-accent-blue/10 shadow-glow">
            <Image
              src="/zeal-defense-mark.svg"
              alt="AI Sentinel"
              width={30}
              height={30}
              priority
            />
          </div>
          <div className="leading-tight">
            <span className="text-sm font-bold tracking-tight text-glow-blue">
              <span className="text-accent-purple text-glow-purple">
                AI Sentinel
              </span>
            </span>
            <p className="hidden font-mono text-[9px] uppercase tracking-[0.3em] text-muted-foreground sm:block">
              AI Security Posture Management
            </p>
          </div>
        </Link>

        <nav className="flex items-center gap-1 rounded-lg border border-border bg-card/50 p-1">
          {LINKS.map(({ href, label, icon: Icon }) => {
            const active =
              href === "/" ? pathname === "/" : pathname.startsWith(href);
            return (
              <Link
                key={href}
                href={href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex items-center gap-1.5 rounded-md px-3 py-1.5 font-mono text-xs font-semibold uppercase tracking-wider transition-colors",
                  active
                    ? "bg-accent-blue/15 text-accent-blue shadow-glow"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                <Icon className="h-3.5 w-3.5" />
                {label}
              </Link>
            );
          })}
        </nav>
      </div>
    </header>
  );
}
