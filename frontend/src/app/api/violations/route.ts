import { NextResponse } from "next/server";
import { COMPLIANCE_SVC_URL, fetchUpstream } from "@/lib/services";
import type { Violation } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Proxies compliance-svc GET /compliance/violations.
export async function GET() {
  const result = await fetchUpstream<Violation[]>(
    `${COMPLIANCE_SVC_URL}/compliance/violations?limit=100`
  );
  if (!result.ok) {
    return NextResponse.json(
      { error: result.error, detail: result.detail },
      { status: result.status }
    );
  }
  return NextResponse.json(result.data);
}
