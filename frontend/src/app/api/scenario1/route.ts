import { NextResponse } from "next/server";
import { promises as fs } from "fs";
import path from "path";
import { PROMPT_SVC_URL, fetchUpstream } from "@/lib/services";
import type { InspectRequest, InspectResult } from "@/lib/types";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

// Read the bundled demo CSV at request time. Lives at src/data and is copied
// into the standalone runtime image (see Dockerfile).
async function readDemoCsv(): Promise<string> {
  const filePath = path.join(process.cwd(), "src", "data", "customer_database.csv");
  return fs.readFile(filePath, "utf-8");
}

// Scenario 1: employee uploads customer_database.csv to ChatGPT.
// POSTs the bundled CSV to prompt-svc /prompts/inspect and returns InspectResult.
export async function POST() {
  let content: string;
  try {
    content = await readDemoCsv();
  } catch (e) {
    const message = e instanceof Error ? e.message : String(e);
    return NextResponse.json(
      { error: "Bundled demo CSV not found", detail: message },
      { status: 500 }
    );
  }

  const body: InspectRequest = {
    userEmail: "james.wilson@example.com",
    provider: "chatgpt",
    content,
    artifactName: "customer_database.csv",
    contentType: "csv",
    scenario: "scenario-1",
  };

  const result = await fetchUpstream<InspectResult>(
    `${PROMPT_SVC_URL}/prompts/inspect`,
    {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(body),
      timeoutMs: 30000,
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
