import { NextResponse } from "next/server";
import { AGENT_SVC_URL, fetchUpstream } from "@/lib/services";
import type { Agent } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies agent-svc GET /agents (Phase 3 — discovered agents + risk scores).
export async function GET() {
  const result = await fetchUpstream<Agent[]>(`${AGENT_SVC_URL}/agents`);
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
