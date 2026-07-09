import { NextResponse } from "next/server";
import { COPILOT_SVC_URL, fetchUpstream } from "@/lib/services";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies copilot-svc GET /copilot/explain/{violationId} (Phase 4 — three
// grounded views + citations per contracts/openapi/copilot.yaml AskResponse).
// The body is forwarded verbatim; the UI owns the shape.
export async function GET(
  _req: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const result = await fetchUpstream<unknown>(
    `${COPILOT_SVC_URL}/copilot/explain/${encodeURIComponent(id)}`
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
