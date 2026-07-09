// Server-side only. Resolves upstream service URLs with compose-DNS defaults.

export const PROMPT_SVC_URL =
  process.env.PROMPT_SVC_URL ?? "http://prompt-svc:8082";
export const DISCOVERY_SVC_URL =
  process.env.DISCOVERY_SVC_URL ?? "http://discovery-svc:8081";
export const COMPLIANCE_SVC_URL =
  process.env.COMPLIANCE_SVC_URL ?? "http://compliance-svc:8084";
export const RUNTIME_SVC_URL =
  process.env.RUNTIME_SVC_URL ?? "http://telemetry-generator:8087";
export const AGENT_SVC_URL =
  process.env.AGENT_SVC_URL ?? "http://agent-svc:8083";
export const GRAPH_SVC_URL =
  process.env.GRAPH_SVC_URL ?? "http://graph-svc:8085";
export const COPILOT_SVC_URL =
  process.env.COPILOT_SVC_URL ?? "http://copilot-svc:8086";

const DEFAULT_TIMEOUT_MS = 15000;

/**
 * Fetch JSON from an upstream service. Never throws on network failure or
 * non-2xx; instead returns a typed envelope the route handler can forward.
 */
export async function fetchUpstream<T>(
  url: string,
  init?: RequestInit & { timeoutMs?: number }
): Promise<
  | { ok: true; data: T }
  | { ok: false; status: number; error: string; detail?: string }
> {
  const controller = new AbortController();
  const timeout = setTimeout(
    () => controller.abort(),
    init?.timeoutMs ?? DEFAULT_TIMEOUT_MS
  );
  try {
    const res = await fetch(url, {
      ...init,
      signal: controller.signal,
      cache: "no-store",
    });
    // Decode explicitly as UTF-8. Upstream Go services send `application/json`
    // without a charset; relying on charset inference mojibakes non-ASCII (e.g. em
    // dashes in framework display names). TextDecoder('utf-8') is deterministic.
    const text = new TextDecoder("utf-8").decode(await res.arrayBuffer());
    let body: unknown = undefined;
    if (text) {
      try {
        body = JSON.parse(text);
      } catch {
        body = text;
      }
    }
    if (!res.ok) {
      const detail =
        typeof body === "object" && body !== null
          ? JSON.stringify(body)
          : String(body ?? "");
      return {
        ok: false,
        status: res.status,
        error: `Upstream ${res.status}`,
        detail,
      };
    }
    return { ok: true, data: body as T };
  } catch (e) {
    const message = e instanceof Error ? e.message : String(e);
    const unreachable =
      message.includes("abort") ||
      message.includes("fetch failed") ||
      message.includes("ECONNREFUSED") ||
      message.includes("ENOTFOUND");
    return {
      ok: false,
      status: 503,
      error: unreachable ? "Service unreachable" : "Upstream error",
      detail: message,
    };
  } finally {
    clearTimeout(timeout);
  }
}
