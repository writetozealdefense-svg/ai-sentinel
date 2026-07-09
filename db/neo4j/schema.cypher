// ZealDefense AI Sentinel — Neo4j risk graph model (Phase 0 contract)
//
// Nodes:         User, Prompt, Model, Agent, Tool, Dataset, Violation
// Relationships: USED, ACCESSED, GENERATED, EXPOSED
//
//   (User)-[:USED]->(Model)
//   (User)-[:ACCESSED]->(Dataset)
//   (Prompt)-[:EXPOSED]->(Dataset)        // prompt leaked data from a dataset
//   (User)-[:GENERATED]->(Prompt)
//   (Agent)-[:USED]->(Tool)
//   (Agent)-[:ACCESSED]->(Dataset)
//   (Prompt)-[:EXPOSED]->(Violation)      // a prompt raised a violation
//
// Run once on first boot to create constraints (idempotent).

CREATE CONSTRAINT user_email IF NOT EXISTS
  FOR (u:User) REQUIRE u.email IS UNIQUE;

CREATE CONSTRAINT model_provider IF NOT EXISTS
  FOR (m:Model) REQUIRE m.provider IS UNIQUE;

CREATE CONSTRAINT prompt_id IF NOT EXISTS
  FOR (p:Prompt) REQUIRE p.id IS UNIQUE;

CREATE CONSTRAINT agent_id IF NOT EXISTS
  FOR (a:Agent) REQUIRE a.id IS UNIQUE;

CREATE CONSTRAINT tool_name IF NOT EXISTS
  FOR (t:Tool) REQUIRE t.name IS UNIQUE;

CREATE CONSTRAINT dataset_name IF NOT EXISTS
  FOR (d:Dataset) REQUIRE d.name IS UNIQUE;

CREATE CONSTRAINT violation_id IF NOT EXISTS
  FOR (v:Violation) REQUIRE v.id IS UNIQUE;
