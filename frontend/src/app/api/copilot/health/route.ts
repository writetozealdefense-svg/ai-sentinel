import { NextResponse } from "next/server";
import { COPILOT_SVC_URL, fetchUpstream } from "@/lib/services";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// CopilotHealth — contracts/openapi/copilot.yaml /healthz.
interface CopilotHealth {
  status: "ok";
  mode: "offline" | "live";
}

// Proxies copilot-svc GET /healthz (Phase 4 — mode badge: offline | live).
export async function GET() {
  const result = await fetchUpstream<CopilotHealth>(
    `${COPILOT_SVC_URL}/healthz`,
    { timeoutMs: 5000 }
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
