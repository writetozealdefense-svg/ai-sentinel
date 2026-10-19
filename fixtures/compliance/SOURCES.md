# Saudi Rule Packs — Source & Verification Audit Trail

Research date: **2026-08-10**. This file is the evidence trail for the Saudi/GCC compliance
packs (`nca_ecc.yaml`, `sama_csf.yaml`, `pdpl.yaml`, `sdaia_ai.yaml`) and the checklist the
product owner works through before removing `# REVIEW` markers. Suitable as supporting
evidence for the Gate log in `VERIFICATION.md`.

---

## 1. NCA ECC — `nca_ecc.yaml` — confidence: HIGH

- **Current version confirmed: ECC-2:2024** (supersedes ECC-1:2018). 4 domains, 28
  subdomains, 108 controls. Numbering `<domain>-<subdomain>-<control>` (e.g., 2-7-1).
- Verified from the **fetched official NCA English PDF**:
  - Controls page: <https://nca.gov.sa/en/regulatory-documents/controls-list/ecc/>
  - Official PDF: <https://cdn.nca.gov.sa/api/files/public/upload/86e09090-44e4-481f-bc28-355673607654_ECC--2024-EN.pdf>
- All 10 control IDs in the pack were read from the official PDF.
- **Watch item:** NCA **AI Cybersecurity Guidelines (AICG-1:2026)** is DRAFT — public
  consultation closed 2026-08-05, final pending. Structure mirrors ECC (4 domains), covers
  generative and agentic AI. **Not shipped as a framework.** Re-check for final publication
  before Black Hat MEA (1–3 Dec 2026) — shipping it day-one when final is a differentiator.
  Draft: <https://cdn.nca.gov.sa/api/files/public/upload/ef097c32-075b-4e44-9798-3b5d6b8270f7_AI_Cybersecurity_Guidelines-_EN-.pdf>

**Owner checklist:** spot-check the 10 IDs against the PDF (esp. subcontrol numbering
2-2-3-3, 2-12-3-2/-4, 4-1-2-1, 4-2-3-1) → delete review markers.

## 2. SAMA CSF — `sama_csf.yaml` — confidence: HIGH (via mirrors)

- **Current version: v1.0, May 2017 — no v2.0 exists** (in force as of mid-2026). Issued
  under Circular 381000091275 (24/05/2017). Applies to all Member Organizations.
- Official endpoints were partially unreachable during research (sama.gov.sa PDF errored;
  rulebook.sama.gov.sa blocks robots). Text verified from **two independent mirror copies of
  the official v1.0 PDF**, numbering cross-confirmed on official Rulebook pages:
  - Rulebook: <https://rulebook.sama.gov.sa/en/cyber-security-framework-2> (e.g., 3.3.14, 3.4 pages)
  - Mirrors: kr-labs.com.ua and allegrosolutionsllc.com copies (identical numbering/titles)
- Key verbatim anchors: **3.3.8 consideration 6a** — the only explicit
  "data-leakage prevention [DLP]" mandate; **3.3.6 consideration 5d** — customer-data
  protection tied to classification; **3.4.3** — KSA data residency + SAMA approval for
  locations abroad.
- Discrepancy logged: an extraction hit for "data leakage" at 3.1.3(4f) did NOT verify on
  targeted quote — 3.1.3 is not cited in the pack.
- No SAMA AI-specific circular found (negative finding, secondary-verified). SDAIA
  instruments cover AI; adjacent SAMA instruments noted: IT Governance Framework (2021),
  Counter-Fraud Framework (2022), Cyber Resilience Fundamental Requirements (2022).

**Owner checklist:** open the official CSF PDF from a KSA-reachable network, confirm the 11
subdomain numbers/titles, esp. 3.3.8(6a) and 3.3.6(5d) verbatim text → delete markers.

## 3. PDPL — `pdpl.yaml` — confidence: MEDIUM-HIGH

- Law: Royal Decree **M/19** (16 Sep 2021), amended **M/148** (27 Mar 2023); 43 articles.
  In force 14 Sep 2023; grace period ended **14 Sep 2024 — full enforcement live**.
  Enforcement is real: **48 SDAIA enforcement decisions by mid-Jan 2026**; top violation
  categories include unauthorized disclosure and missing technical/organizational
  safeguards (Clyde & Co, Mar 2026).
- Official SDAIA PDFs **reject automated fetches (WAF)**. Article text verified against a
  verbatim reproduction of the SDAIA official English translation (ksapdpl.com),
  cross-checked with DLA Piper, Clyde & Co, King & Spalding, Multilaw, Akin — no
  contradictions. SDAIA "Guide to the PDPL" (dgp.sdaia.gov.sa) fetched OK and used for
  cross-checks.
- **Numbering hazard:** some sites circulate an alternative numbering ("Art 18 = breach").
  The pack uses the SDAIA-derived numbering (Art 19 = safeguards, Art 20 = breach) that all
  major law-firm sources support. Do not mix sources.
- Secondary-only items (flagged in review markers): **IR Article 24 = 72-hour** SDAIA
  notification; Transfer Regulation **Article 7** risk assessment. Verify both against the
  official IR/Transfer Regulation PDFs.
- M/148 changes reflected: legitimate-interest basis (Art 6(4)); Art 15 disclosure grounds;
  Art 29 rewritten to adequacy/safeguards model; criminal liability narrowed to Art 35
  (Sensitive Data disclosure, ≤2 yrs / ≤SAR 3M); Art 36 admin fines ≤SAR 5M (doubled on
  repeat).
- Definitions used by the pack (Art 1): "Disclosure" = enabling any person other than
  controller/processor to access/collect/use personal data by any means; "Sensitive Data"
  includes health, biometric, genetic, credit/financial, location, criminal, belief data.

**Owner checklist:** spot-check Arts 5, 11, 15, 19, 20, 23, 29, 35, 36 against the SDAIA
PDF; confirm IR Art 24 (72h) and Transfer Reg Art 7 → delete markers.

## 4. SDAIA AI — `sdaia_ai.yaml` — confidence: MEDIUM (constructed IDs, flagged)

- **AI Ethics Principles** (Sept 2023; cover says "Version 1.0" but market cites v2.0):
  verified from the **primary PDF** on dgp.sdaia.gov.sa. 7 principles in order: Fairness;
  Privacy & Security; Humanity; Social & Environmental Benefits; Reliability & Safety;
  Transparency & Explainability; Accountability & Responsibility. 4 risk tiers (compliance
  mandatory for limited/high risk; unacceptable-risk AI prohibited). Roles: Responsible AI
  Officer, AI System Assessor. Official lifecycle control codes exist (PD.n, PID.n, BV.n,
  DM.n) — per-phase counts not re-verified, so the pack uses principle-level constructed
  IDs (`AIEP-Pn`), clearly flagged.
- **Generative AI Guidelines** (Jan 2024, government + public editions; "2025" refreshed
  editions indexed): sdaia.gov.sa WAF-blocked, so requirements are from CMS, HSF Kramer,
  Securiti, Adeptiv summaries — all `GAIG-*` IDs are constructed and flagged. Gov edition
  binding for government entities (incl. compliance checklist); public edition advisory.
- **Not shipped (monitor list):**
  - NCA **AICG-1:2026** — draft (see §1).
  - SDAIA **Responsible AI Policy** — consultation closed 3 May 2026, NOT enacted. Four
    risk tiers, registration, labelling, audits for high-risk. The likely future binding
    regime — highest-value future pack.
  - **National AI Risk Management Framework (NAIRMF)** — final Apr 2026 (Cabinet Res. 292
    basis; 7 risk categories, 5-phase process). Add as a pack after reviewing primary text.
  - No standalone "Deemed-High-Risk AI rules" and no NCA/SDAIA joint AI security rulebook
    exist (verified negative findings).

**Owner checklist:** download the two GenAI guideline PDFs from a KSA-reachable network,
verify the four GAIG requirements' wording, decide whether to switch AIEP IDs to official
lifecycle codes (PD/PID/BV/DM) → delete markers. Track the three monitor-list instruments.

---

## Demo note (Black Hat MEA)

`reviewRequired:true` will show on all four Saudi frameworks until markers are removed —
which is itself an honest demo beat consistent with the platform's authenticity principle:
"every control ID is source-cited and human-verified before the badge clears." The mapping
rule picks the single highest-severity matching control per framework; flagship controls
are ordered first (NCA 2-7-2, SAMA 3.3.8, PDPL Art 35/29, SDAIA GAIG-GOV-ProhibInput) so
demo scenarios surface the strongest story.
