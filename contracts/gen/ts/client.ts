// Typed API clients for the ZealDefense AI Sentinel services, built from the generated
// contract types via openapi-fetch. The frontend imports these — services conform to the
// contract, not the reverse (Brief §3).
import createClient from "openapi-fetch";
import type { paths as DiscoverPaths } from "./generated/discover.js";
import type { paths as PromptPaths } from "./generated/prompts.js";
import type { paths as DlpPaths } from "./generated/dlp.js";
import type { paths as AgentPaths } from "./generated/agents.js";
import type { paths as CompliancePaths } from "./generated/compliance.js";
import type { paths as GraphPaths } from "./generated/graph.js";
import type { paths as CopilotPaths } from "./generated/copilot.js";

export interface Endpoints {
  discovery: string;
  prompts: string;
  dlp: string;
  agents: string;
  compliance: string;
  graph: string;
  copilot: string;
}

export function makeClients(e: Endpoints) {
  return {
    discovery: createClient<DiscoverPaths>({ baseUrl: e.discovery }),
    prompts: createClient<PromptPaths>({ baseUrl: e.prompts }),
    dlp: createClient<DlpPaths>({ baseUrl: e.dlp }),
    agents: createClient<AgentPaths>({ baseUrl: e.agents }),
    compliance: createClient<CompliancePaths>({ baseUrl: e.compliance }),
    graph: createClient<GraphPaths>({ baseUrl: e.graph }),
    copilot: createClient<CopilotPaths>({ baseUrl: e.copilot }),
  };
}

export type { DiscoverPaths, PromptPaths, DlpPaths, AgentPaths, CompliancePaths, GraphPaths, CopilotPaths };
