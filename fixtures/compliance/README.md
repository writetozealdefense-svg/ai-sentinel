# Compliance Fixtures (Brief R5)

Control mappings are read **exclusively** from these YAML files. Services never invent
control IDs or clause numbers.

## Rules
1. Every mapping carries a `review: "# REVIEW: verify control ID"` marker until a human who
   knows the framework has confirmed it. The compliance service surfaces `reviewRequired:true`
   on any framework that still has unreviewed mappings.
2. **Well-documented frameworks** (GDPR, HIPAA, ISO 27001, NIST AI RMF) are seeded with a
   small, conservative starter set of *publicly documented* article/section references — still
   marked for review.
3. **Regional frameworks the product owner will personally verify** (NCA ECC, SAMA CSF, PDPL,
   DPDPA) are intentionally **NOT** populated with clause numbers from memory. Their files
   contain `controlId: "TODO-VERIFY"` placeholders only. Do not guess these — they will be
   reviewed by people who wrote the controls.

## File = one framework
`<framework>.yaml` → `framework`, `displayName`, and a `controls:` list. Each control:
`controlId`, `title`, `description`, `mappedCategories` (PII/PHI/PCI/SECRET/CODE), `severity`,
`review`.
