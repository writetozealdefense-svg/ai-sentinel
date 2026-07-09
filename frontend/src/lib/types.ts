// Types mirror contracts/openapi/{prompts,discover,compliance}.yaml exactly.
// Kept local (rather than importing the generated paths/components indirection)
// so the BFF and UI stay self-contained.

export type Decision = "ALLOW" | "WARN" | "REDACT" | "BLOCK";
export type Category = "PII" | "PHI" | "PCI" | "SECRET" | "CODE" | "OTHER";
export type Engine = "presidio" | "yara" | "regex";

// --- prompts.yaml ---
export interface Finding {
  type: string;
  category: Category;
  engine: Engine;
  score: number;
  count?: number;
}

export interface InspectRequest {
  userEmail: string;
  provider: string;
  content: string;
  artifactName?: string;
  contentType?: "text" | "csv" | "code" | "json";
  scenario?: string;
}

export interface InspectResult {
  promptId: string;
  userEmail?: string;
  provider?: string;
  artifactName?: string;
  decision: Decision;
  riskScore: number;
  findings: Finding[];
  redactedPreview?: string;
  violationId?: string;
  policyTrace?: string;
  timestamp: string;
}

// --- discover.yaml ---
export interface AiInventoryItem {
  provider: string;
  usageCount: number;
  uniqueUsers?: number;
  sanctioned: boolean;
  firstSeen?: string;
  lastSeen?: string;
}

// --- compliance.yaml ---
export type Framework =
  | "GDPR"
  | "HIPAA"
  | "ISO27001"
  | "NIST_AI_RMF"
  | "SAMA_CSF"
  | "NCA_ECC"
  | "PDPL"
  | "DPDPA";

export interface FrameworkScore {
  framework: Framework | string;
  displayName?: string;
  score: number;
  controlCount: number;
  openViolations: number;
  reviewRequired?: boolean;
}

// PromptSummary — prompts.yaml listPrompts (SOC live monitoring).
export interface PromptSummary {
  promptId: string;
  userEmail: string;
  provider: string;
  artifactName?: string;
  decision: Decision;
  riskScore: number;
  categories?: Category[] | string[];
  scenario?: string;
  timestamp: string;
}

// ControlMapping — compliance.yaml.
export interface ControlMapping {
  framework: string;
  controlId: string;
  title: string;
  description?: string;
  mappedCategories?: Array<Category | string>;
  reviewMarker: string;
}

export type Severity = "low" | "medium" | "high" | "critical";

// Violation — compliance.yaml.
export interface Violation {
  violationId: string;
  promptId?: string;
  userEmail?: string;
  framework: string;
  controlId: string;
  category?: Category | string;
  severity: Severity;
  description?: string;
  timestamp: string;
}

// --- runtime.yaml (SIMULATED telemetry-generator feed) ---
export type RuntimeSource = "falco" | "tetragon";
export type RuntimePriority =
  | "info"
  | "notice"
  | "warning"
  | "error"
  | "critical";

export interface RuntimeEvent {
  eventId: string;
  source: RuntimeSource;
  rule: string;
  priority: RuntimePriority;
  message: string;
  namespace?: string;
  pod?: string;
  container?: string;
  simulated: true;
  timestamp: string;
}

// --- agents.yaml (Phase 3 — SIMULATED agent telemetry, Brief R4) ---
export type AgentFramework =
  | "langgraph"
  | "crewai"
  | "autogen"
  | "openai-agents-sdk"
  | "mcp";

export interface Agent {
  agentId: string;
  name?: string;
  framework: AgentFramework | string;
  owner?: string;
  tools?: string[];
  riskScore: number;
  riskFactors?: string[];
  simulated: boolean;
  lastSeen?: string;
}

// --- graph.yaml (Phase 3 — Neo4j risk graph) ---
export type GraphNodeLabel =
  | "User"
  | "Prompt"
  | "Model"
  | "Agent"
  | "Tool"
  | "Dataset"
  | "Violation";

export type GraphEdgeType = "USED" | "ACCESSED" | "GENERATED" | "EXPOSED";

export interface NamedQuery {
  name: string;
  description: string;
  cypher: string;
  parameters?: string[];
}

export interface GraphNode {
  id: string;
  label: GraphNodeLabel | string;
  properties?: Record<string, unknown>;
}

export interface GraphEdge {
  source: string;
  target: string;
  type: GraphEdgeType | string;
}

export interface GraphResult {
  rows: Array<Record<string, unknown>>;
  nodes: GraphNode[];
  edges: GraphEdge[];
  cypher?: string;
}

// BFF error envelope (never throws to the browser).
export interface BffError {
  error: string;
  detail?: string;
}
