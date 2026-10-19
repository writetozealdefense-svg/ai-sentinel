# Compliance Fixtures (Brief R5)

Control mappings are read **exclusively** from these YAML files. Services never invent
control IDs or clause numbers.

## Rules
1. Every mapping carries a `review: "# REVIEW: ..."` marker until a human who knows the
   framework has confirmed it. The compliance service surfaces `reviewRequired:true` on any
   framework that still has unreviewed mappings.
2. **Well-documented frameworks** (GDPR, HIPAA, ISO 27001, NIST AI RMF) are seeded with a
   small, conservative starter set of *publicly documented* article/section references — still
   marked for review.
3. **Saudi/GCC frameworks** (NCA ECC-2:2024, SAMA CSF v1.0, PDPL, SDAIA AI) were populated
   on 2026-08-10 from research against official sources — every control cites its source
   document in the file header and carries a per-control review marker recording *how* it was
   verified (official PDF fetched / official mirror / secondary reproduction). See
   [`SOURCES.md`](SOURCES.md) for the full audit trail and known caveats. Markers stay until
   the product owner signs off; **do not** remove a marker without checking the cited source.
4. DPDPA (India) remains a `TODO-VERIFY` placeholder — out of scope for the Saudi-core round.

## File = one framework
`<framework>.yaml` → `framework`, `displayName`, and a `controls:` list. Each control:
`controlId`, `title`, `description`, `mappedCategories` (PII/PHI/PCI/SECRET/CODE/OTHER),
`severity` (low|medium|high|critical), `review`.

Frameworks are auto-discovered (`*.yaml` glob) — adding a file adds a framework.
The mapping rule (PHASE1-INTEGRATION.md Interface 4) selects the single highest-severity
matching control per framework; list order breaks ties (first wins), so flagship controls
are ordered first in each Saudi pack.
