import { NextResponse } from "next/server";
import { GRAPH_SVC_URL, fetchUpstream } from "@/lib/services";
import type { GraphResult } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies graph-svc POST /graph/query/{name} (Phase 3 — run a named graph
// query). Forwards the JSON body (query parameters) when one is provided.
export async function POST(
  req: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  const { name } = await params;

  // Forward the request body verbatim if present (named-query parameters).
  let body: string | undefined;
  try {
    const text = await req.text();
    if (text) body = text;
  } catch {
    body = undefined;
  }

  const result = await fetchUpstream<GraphResult>(
    `${GRAPH_SVC_URL}/graph/query/${encodeURIComponent(name)}`,
    {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: body ?? "{}",
    }
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
