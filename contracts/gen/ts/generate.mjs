// Generates TypeScript types from each OpenAPI contract into generated/<spec>.ts
import { execSync } from "node:child_process";
import { mkdirSync } from "node:fs";

const specs = ["discover", "prompts", "dlp", "agents", "compliance", "graph", "copilot"];
mkdirSync("generated", { recursive: true });

for (const s of specs) {
  const out = `generated/${s}.ts`;
  execSync(`npx openapi-typescript ../../openapi/${s}.yaml -o ${out}`, { stdio: "inherit" });
  console.log(`generated ${out}`);
}
