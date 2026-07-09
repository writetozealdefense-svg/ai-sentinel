import { NextResponse } from "next/server";
import { RUNTIME_SVC_URL, fetchUpstream } from "@/lib/services";
import type { RuntimeEvent } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies telemetry-generator GET /runtime/events (SIMULATED Falco/Tetragon feed).
export async function GET() {
  const result = await fetchUpstream<RuntimeEvent[]>(
    `${RUNTIME_SVC_URL}/runtime/events?limit=50`
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
