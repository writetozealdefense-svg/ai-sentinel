-- ZealDefense AI Sentinel — PostgreSQL schema (Phase 0 contract)
-- Entities: User, Prompt, Model, Agent, Tool, Dataset, Violation (+ supporting tables).
-- This DDL is a committed contract; services conform to it.

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- gen_random_uuid()

-- ---------------------------------------------------------------------------
-- Users (employees observed using AI)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT UNIQUE NOT NULL,
    display_name TEXT,
    department  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Models / AI systems (ChatGPT, Claude, DeepSeek, ...)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS models (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider    TEXT UNIQUE NOT NULL,   -- chatgpt | claude | deepseek | ...
    vendor      TEXT,
    sanctioned  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Discovery events (SIMULATED telemetry — Brief R4)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS discovery_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider      TEXT NOT NULL,
    user_email    TEXT NOT NULL,
    source        TEXT NOT NULL,        -- browser-ext | dns | proxy | endpoint
    action        TEXT,                 -- visit | upload | prompt | download
    artifact_name TEXT,
    bytes         BIGINT,
    simulated     BOOLEAN NOT NULL DEFAULT TRUE,  -- always TRUE on compose path
    occurred_at   TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_discovery_provider ON discovery_events (provider);
CREATE INDEX IF NOT EXISTS idx_discovery_user ON discovery_events (user_email);

-- ---------------------------------------------------------------------------
-- Prompts (inspected AI interactions)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS prompts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_email      TEXT NOT NULL,
    provider        TEXT NOT NULL,
    artifact_name   TEXT,
    content_type    TEXT NOT NULL DEFAULT 'text',
    decision        TEXT NOT NULL CHECK (decision IN ('ALLOW','WARN','REDACT','BLOCK')),
    risk_score      INT NOT NULL DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    redacted_preview TEXT,
    policy_trace    TEXT,
    scenario        TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_prompts_user ON prompts (user_email);
CREATE INDEX IF NOT EXISTS idx_prompts_decision ON prompts (decision);

-- ---------------------------------------------------------------------------
-- Findings (per-prompt detection hits from detection-svc)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS findings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id   UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,          -- US_SSN | EMAIL_ADDRESS | CREDIT_CARD | AWS_KEY | SOURCE_CODE
    category    TEXT NOT NULL CHECK (category IN ('PII','PHI','PCI','SECRET','CODE','OTHER')),
    engine      TEXT NOT NULL CHECK (engine IN ('presidio','yara','regex')),
    score       REAL NOT NULL,
    rule_name   TEXT,
    occurrences INT NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_findings_prompt ON findings (prompt_id);

-- ---------------------------------------------------------------------------
-- Datasets (artifacts/files referenced by prompts)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS datasets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    classification TEXT,                -- public | internal | confidential | restricted
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Agents (agentic AI — Phase 3) and the tools they touch
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agents (
    id          TEXT PRIMARY KEY,       -- agent identifier from telemetry
    name        TEXT,
    framework   TEXT NOT NULL,          -- langgraph | crewai | autogen | openai-agents-sdk | mcp
    owner_email TEXT,
    risk_score  INT NOT NULL DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    simulated   BOOLEAN NOT NULL DEFAULT TRUE,
    last_seen   TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS tools (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL    -- jira | github | salesforce | ...
);

CREATE TABLE IF NOT EXISTS agent_activity (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id    TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    action      TEXT NOT NULL,          -- tool_call | data_access | privilege_escalation | external_call
    tool        TEXT,
    target      TEXT,
    sensitive   BOOLEAN NOT NULL DEFAULT FALSE,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agent_activity_agent ON agent_activity (agent_id);

-- ---------------------------------------------------------------------------
-- Violations (compliance — control IDs sourced from fixtures only, Brief R5)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS violations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id   UUID REFERENCES prompts(id) ON DELETE SET NULL,
    agent_id    TEXT REFERENCES agents(id) ON DELETE SET NULL,
    user_email  TEXT,
    framework   TEXT NOT NULL,
    control_id  TEXT NOT NULL,          -- from fixtures/compliance/*.yaml
    category    TEXT NOT NULL CHECK (category IN ('PII','PHI','PCI','SECRET','CODE','OTHER')),
    severity    TEXT NOT NULL CHECK (severity IN ('low','medium','high','critical')),
    description TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_violations_framework ON violations (framework);
CREATE INDEX IF NOT EXISTS idx_violations_prompt ON violations (prompt_id);
