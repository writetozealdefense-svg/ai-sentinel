import { NextResponse } from "next/server";
import { DISCOVERY_SVC_URL, fetchUpstream } from "@/lib/services";
import type { AiInventoryItem } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies discovery-svc GET /discover/inventory.
export async function GET() {
  const result = await fetchUpstream<AiInventoryItem[]>(
    `${DISCOVERY_SVC_URL}/discover/inventory`
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
