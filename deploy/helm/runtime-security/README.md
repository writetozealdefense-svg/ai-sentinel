# Runtime Security Add-ons (Helm path)

Real runtime security for the AI Sentinel kind cluster: **Kyverno** (admission
control), **Falco + falcosidekick** (runtime detection, wired to the SOC feed),
and **Tetragon** (eBPF observability with a TracingPolicy for `/etc/shadow`
reads). Contract: `docs/PHASE5-INTEGRATION.md` §"Runtime security" and
§"Verification Gate 5".

Prerequisites: the `ai-sentinel` umbrella chart is installed in namespace
`ai-sentinel` (telemetry-generator must be reachable at
`http://telemetry-generator.ai-sentinel.svc:8087` with `RUNTIME_MODE=ingest`
for real Falco events to reach the SOC dashboard).

All commands assume repo root `D:\AISPM - Sentinel` as the working directory.

## 1. Helm repos

```powershell
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo add cilium https://helm.cilium.io
helm repo update
```

## 2. Install Kyverno (admission control first — it protects everything after)

```powershell
helm install kyverno kyverno/kyverno -n kyverno --create-namespace
kubectl wait --for=condition=Ready pods -n kyverno --all --timeout=180s
kubectl apply -f deploy/helm/runtime-security/kyverno-policies.yaml
kubectl get clusterpolicy
```

Expected: `disallow-privileged-containers` and `require-run-as-nonroot` both
show `ADMISSION: true`, `READY: True`, `VALIDATE ACTION: Enforce`.

Note: the policies match namespace `ai-sentinel` only, so installing
Falco/Tetragon (which need privileged DaemonSets in other namespaces) is not
blocked by them.

## 3. Install Falco + falcosidekick

```powershell
helm install falco falcosecurity/falco -n falco --create-namespace `
  -f deploy/helm/runtime-security/falco-values.yaml
kubectl wait --for=condition=Ready pods -n falco --all --timeout=300s
kubectl logs -n falco -l app.kubernetes.io/name=falco -c falco --tail=20
```

Expected in the falco container logs (success case):

```
Falco initialized with configuration files: ...
Loaded event sources: syscall
Enabled event sources: syscall
Opening 'syscall' source with modern BPF probe.
```

falcosidekick logs should show the webhook output enabled:

```powershell
kubectl logs -n falco -l app.kubernetes.io/name=falcosidekick --tail=20
# expect: "Webhook. Address=http://telemetry-generator.ai-sentinel.svc:8087/runtime/events" among enabled outputs
```

## 4. Install Tetragon + TracingPolicy

```powershell
helm install tetragon cilium/tetragon -n kube-system `
  -f deploy/helm/runtime-security/tetragon-values.yaml
kubectl rollout status -n kube-system ds/tetragon --timeout=300s
kubectl apply -f deploy/helm/runtime-security/tetragon-tracingpolicy.yaml
kubectl get tracingpolicy
```

Expected: `file-monitoring-etc-shadow` listed. (The TracingPolicy CRD is
installed by the tetragon-operator; if `kubectl apply` reports
`no matches for kind "TracingPolicy"`, wait for the operator pod and retry.)

## 5. Demo 1 — Kyverno admission rejection (bad-pod)

```powershell
kubectl apply -f deploy/k8s-tests/bad-pod.yaml
```

Expected output (REJECTION — paste this verbatim into VERIFICATION.md):

```
Error from server: error when creating "deploy/k8s-tests/bad-pod.yaml": admission webhook "validate.kyverno.svc-fail" denied the request:

resource Pod/ai-sentinel/bad-pod was blocked due to the following policies

disallow-privileged-containers:
  privileged-containers: 'validation error: Privileged mode is disallowed in the ai-sentinel
    namespace. ...'
require-run-as-nonroot:
  run-as-non-root: 'validation error: Running as root is not allowed in the ai-sentinel
    namespace. ...'
```

(Exact wording varies slightly by Kyverno version; the essentials are
"denied the request" and both policy names.)

## 6. Demo 2 — Falco "Terminal shell in container"

Exec a shell into any workload pod in `ai-sentinel`:

```powershell
kubectl exec -it -n ai-sentinel deploy/telemetry-generator -- /bin/sh
# type a command or two, then: exit
```

Watch the alert:

```powershell
kubectl logs -n falco -l app.kubernetes.io/name=falco -c falco --tail=50 | Select-String "Terminal shell"
```

Expected alert JSON (shape; paste the real one):

```json
{"hostname":"ai-sentinel-control-plane","output":"... Notice A shell was spawned in a container with an attached terminal (user=... container_id=... container_name=telemetry-generator ... shell=sh parent=runc cmdline=sh ...)","priority":"Notice","rule":"Terminal shell in container","source":"syscall","tags":["container","mitre_execution","shell"],"time":"..."}
```

If the sidekick wiring works, the same event arrives at telemetry-generator's
`POST /runtime/events`, is normalized with `simulated: false`, and the SOC UI
badge switches to **LIVE (Falco)**. Verify:

```powershell
kubectl logs -n ai-sentinel deploy/telemetry-generator --tail=20
```

## 7. Demo 3 (optional) — Tetragon sensitive file read

```powershell
kubectl exec -n ai-sentinel deploy/telemetry-generator -- cat /etc/shadow
kubectl logs -n kube-system ds/tetragon -c export-stdout --tail=200 | Select-String "shadow"
```

Expected: a `process_kprobe` JSON event with `function_name:
"security_file_permission"` and the `/etc/shadow` file argument.

## Fallback notes (be honest — never fake runtime events)

- **If Falco's `modern_ebpf` driver fails to load on this WSL2 kernel**
  (symptoms: falco pod CrashLoopBackOff; log lines like
  `Error: can't initialize the libbpf's ring buffer` or
  `Error: unable to load the BPF program` or BTF-related errors):
  1. Capture the error **verbatim**:
     `kubectl logs -n falco -l app.kubernetes.io/name=falco -c falco --previous`
     and paste it into VERIFICATION.md.
  2. Optionally try the legacy eBPF driver (`driver.kind: ebpf`) once; if it
     also fails, capture that too.
  3. Continue with the **Kyverno-only demo** (step 5) — admission control is
     userspace and does not depend on eBPF. State plainly in VERIFICATION.md
     that runtime events were environment-limited. Do **not** switch
     telemetry-generator out of ingest expectations by fabricating events.
- **If Tetragon CO-RE fails similarly**: capture
  `kubectl logs -n kube-system ds/tetragon -c tetragon` verbatim and note it;
  the Kyverno demo still stands.
- **If falcosidekick cannot reach telemetry-generator** (webhook errors in
  sidekick logs): the Falco alert in the falco pod logs still satisfies the
  "real Falco alert" gate; record the sidekick error and check the
  NetworkPolicy in `ai-sentinel` allows ingress to telemetry-generator:8087
  from the `falco` namespace.

## Cleanup

```powershell
kubectl delete -f deploy/helm/runtime-security/tetragon-tracingpolicy.yaml --ignore-not-found
kubectl delete -f deploy/helm/runtime-security/kyverno-policies.yaml --ignore-not-found
helm uninstall tetragon -n kube-system
helm uninstall falco -n falco
helm uninstall kyverno -n kyverno
```
