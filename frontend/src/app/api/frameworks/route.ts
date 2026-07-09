import { NextResponse } from "next/server";
import { COMPLIANCE_SVC_URL, fetchUpstream } from "@/lib/services";
import type { FrameworkScore } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies compliance-svc GET /compliance/frameworks.
export async function GET() {
  const result = await fetchUpstream<FrameworkScore[]>(
    `${COMPLIANCE_SVC_URL}/compliance/frameworks`
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
