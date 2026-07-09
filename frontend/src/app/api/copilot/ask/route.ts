import { NextResponse } from "next/server";
import { COPILOT_SVC_URL, fetchUpstream } from "@/lib/services";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies copilot-svc POST /copilot/ask (Phase 4 — grounded Q&A per
// contracts/openapi/copilot.yaml AskRequest/AskResponse). Forwards the JSON
// body verbatim; upstream validates it (400 on bad request passes through).
export async function POST(req: Request) {
  let body: string | undefined;
  try {
    const text = await req.text();
    if (text) body = text;
  } catch {
    body = undefined;
  }

  const result = await fetchUpstream<unknown>(`${COPILOT_SVC_URL}/copilot/ask`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: body ?? "{}",
  });
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
