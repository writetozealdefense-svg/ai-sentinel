# Phase 5 Integration Contract (Kubernetes Hardening Path)

The architecture/hardening story (Brief R1): real runtime security on a local cluster. This is
the `helm install` path — distinct from the compose demo path.

## Cluster
- **kind** on Docker Desktop (WSL2 backend). Install kind via `go install sigs.k8s.io/kind@v0.23.0`
  (Go 1.26 present). Cluster name `ai-sentinel`, 1 control-plane node.
- WSL2 kernel (5.15+, BTF enabled) supports Falco `modern_ebpf` and Tetragon CO-RE. If eBPF
  loading fails on this machine, record the observed error honestly in VERIFICATION.md and
  demonstrate Kyverno (userspace) fully — do NOT fake runtime events.

## Namespace & baseline hardening
- Namespace `ai-sentinel` labeled for **Pod Security Standards: restricted** (enforce).
- All workload pods: `runAsNonRoot`, no privilege escalation, seccomp RuntimeDefault, read-only
  root FS where possible, resource requests/limits.
- **NetworkPolicies**: default-deny ingress in `ai-sentinel`; explicit allows (frontend→services,
  services→datastores, all→NATS/postgres/neo4j/redis as needed).
- **RBAC**: a dedicated ServiceAccount per service, no cluster-wide permissions.

## Helm chart — `deploy/helm/ai-sentinel/`
One umbrella chart:
- `Chart.yaml`, `values.yaml` (images default to the locally built `ai-sentinel-*:latest`,
  `imagePullPolicy: Never` — images are `kind load`-ed).
- Templates: Deployment+Service per component (postgres, redis, nats, neo4j, detection-svc,
  opa, discovery-svc, prompt-svc, compliance-svc, agent-svc, graph-svc, telemetry-generator,
  copilot-svc, frontend), ConfigMap for compliance fixtures + OPA policy (mounted), Secret for
  DB/Neo4j passwords, NetworkPolicies, ServiceAccounts.
- Datastores as simple single-replica Deployments with emptyDir (demo cluster, not production
  HA — say so in comments).
- `helm lint` must pass. `helm install ai-sentinel deploy/helm/ai-sentinel -n ai-sentinel
  --create-namespace` must bring pods Ready.

## Runtime security (real, on this path only)
- **Falco** via official helm chart (`falcosecurity/falco`), `driver.kind=modern_ebpf`, plus
  **falcosidekick** with an HTTP webhook output → telemetry-generator.
- **Tetragon** via `cilium/tetragon` chart with a TracingPolicy (e.g. sensitive file access
  `/etc/shadow`).
- **Kyverno** via official chart + `ClusterPolicy` (enforce): disallow privileged containers +
  require runAsNonRoot. Demo: apply `deploy/k8s-tests/bad-pod.yaml` (privileged) → admission
  REJECTED; paste the error.
- Falco trigger demo: `kubectl exec` a shell into a workload pod → Falco "Terminal shell in
  container" alert; paste the alert JSON.

## Wiring real Falco → SOC dashboard (replaces the simulated feed on this path)
telemetry-generator gains an ingest mode:
- Env `RUNTIME_MODE=ingest` (default `simulate`): in ingest mode the internal generator is OFF
  and `POST /runtime/events` accepts falcosidekick webhook payloads, normalizing them to
  RuntimeEvent with **`simulated: false`**.
- Contract change (contracts/openapi/runtime.yaml): `simulated` becomes a plain boolean with
  description "true on the compose path (synthetic); false only on the Helm path fed by real
  Falco". The SOC UI badge switches to **LIVE (Falco)** when events carry simulated=false.
- Event schema `contracts/events/runtime.observed.schema.json` mirrors this.

## Verification Gate 5 (record)
`kubectl get pods -n ai-sentinel` all Ready; Kyverno rejection output for bad-pod; a real Falco
alert (and its arrival in the SOC feed if sidekick wiring succeeds); `helm lint` output. Honest
notes for anything environment-limited.
