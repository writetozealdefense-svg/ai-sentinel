import { NextResponse } from "next/server";
import { GRAPH_SVC_URL, fetchUpstream } from "@/lib/services";
import type { NamedQuery } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies graph-svc GET /graph/queries (Phase 3 — named Cypher queries).
export async function GET() {
  const result = await fetchUpstream<NamedQuery[]>(
    `${GRAPH_SVC_URL}/graph/queries`
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
