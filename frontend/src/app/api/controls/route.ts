import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { COMPLIANCE_SVC_URL, fetchUpstream } from "@/lib/services";
import type { ControlMapping } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies compliance-svc GET /compliance/frameworks/{framework}/controls.
export async function GET(req: NextRequest) {
  const framework = req.nextUrl.searchParams.get("framework");
  if (!framework) {
    return NextResponse.json(
      { error: "Missing required query parameter: framework" },
      { status: 400 }
    );
  }

  const result = await fetchUpstream<ControlMapping[]>(
    `${COMPLIANCE_SVC_URL}/compliance/frameworks/${encodeURIComponent(
      framework
    )}/controls`
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
