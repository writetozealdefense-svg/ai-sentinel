# Singapore compliance packs — source audit trail

**Research date:** 2026-09-29 (brief compiled the same day; packs populated 2026-09-29).
**Scope:** nine Singapore fixture packs mapping AI data-leakage / shadow-AI events to regulatory
and guidance controls. Every control carries a `review:` marker recording *how* it was verified.
Markers stay until the product owner signs off — **do not remove a marker without checking the
cited source.**

## How to read the confidence levels

| Level | Meaning |
|---|---|
| HIGH | Official primary document (statute, regulator PDF) was fetched and read; clause numbers and quotes are as read. |
| MEDIUM | Primary document could not be opened; clause numbers and quotes come from consistent secondary sources (law firms, vendor papers, unofficial reproductions), or from a consultation draft rather than the final text. |
| LOW | Instrument is not final and/or clause numbers are not known; controlIds are placeholders. |

## Access limitations during research (apply across packs)

- `sso.agc.gov.sg` (Singapore Statutes Online) returned **403** at the egress proxy — no PDPA
  section was read from the official statute text.
- `mas.gov.sg` was **blocked** — neither the TRM Guidelines PDF nor the AI Risk Management
  consultation paper was opened.
- PDPC "Key Concepts" PDFs and the PDPC consultation page were **blocked by robots.txt**; the
  PDPC final GenAI-guidelines page rendered titles only.
- A MAS-hosted MindForge PDF that matched "shadow AI" in search returned **403** — unconfirmed.
- No web tools were used while writing the packs; the brief is the only source of clause numbers.

---

## 1. `pdpa_sg.yaml` — PDPA_SG — Personal Data Protection Act 2012 (as amended 2020)

**Confidence: MEDIUM.** Binding statute, but the statute text was not read (SSO 403). Section
numbers and wording come from mutually consistent law-firm sources. Regulation numbers in the
Personal Data Protection Regulations 2021 (reported as regs 10–12) are unconfirmed and were
deliberately not cited as controlIds.

**Controls (6):** s 26 (critical), s 24 (high), s 26D (high), s 11–12 (medium), s 48J (medium),
s 25 (low).

**Pages actually opened:**
- https://www.dlapiperdataprotection.com/index.html?t=law&c=SG
- https://www.dlapiperdataprotection.com/index.html?t=transfer&c=SG
- https://www.drewnapier.com/DrewNapier/media/DrewNapier/Lexology-GTDT-Data-Protection-Privacy-2024-Singapore.pdf
- https://resourcehub.bakermckenzie.com/en/resources/global-data-and-cyber-handbook/asia-pacific/singapore/topics/security-requirements-and-breach-notification
- https://practiceguides.chambers.com/practice-guides/comparison/1129/18520/29027-29028-29029-29030-29031
- https://oecd.ai/en/dashboards/policy-initiatives/personal-data-protection-act-pdpa,-section-26-transfer-limitation-obligation-and-personal-data-protection-regulations-2021
- https://www.allenandgledhill.com/sg/perspectives/articles/20283/sgkh-financial-penalty-cap-for-data-breaches-by-organisations-under-personal-data-protection-act-2012-to-increase-from-1-october-2022
- https://covarage.com/guides/procedural-howto/pdpa-data-breach-notification-3-day-pdpc-singapore

**Caveats:**
- Numbering hazard in Part 6A: the data intermediary's duty to notify the organisation "without
  undue delay" is cited as **s 26C(3)(a)** by Baker McKenzie and as **s 26E** by others — section
  placement unconfirmed. The s 26D marker records this.
- s 25 (Retention Limitation): only the section number and title were verified; no statutory
  wording was read. Severity (low) is a judgment call.
- s 48J: the "whichever is higher" phrasing found in some guides is a simplification of the
  Allen & Gledhill wording used in the pack.
- The PDP Regulations 2021 Part 3 transfer conditions are described in the s 26 control but
  no regulation number is cited.

**Owner checklist (before removing markers):**
- [ ] Open PDPA 2012 on SSO from a network that can reach it; confirm ss 11, 12, 24, 25, 26,
      26A–26E and 48J numbering and wording.
- [ ] Resolve the s 26C(3)(a) vs s 26E placement of the data-intermediary notification duty.
- [ ] Confirm the reg numbers for the transfer conditions in the PDP Regulations 2021 and add
      them to the s 26 description if wanted.
- [ ] Confirm "three calendar days" (s 26D(1)) and the significant-harm / 500-individual
      thresholds against the statute and the Notification of Data Breaches Regulations 2021.

---

## 2. `pdpc_genai.yaml` — PDPC_GENAI — Advisory Guidelines on Use of Personal Data in Generative AI (Jul 2026)

**Confidence: MEDIUM.** Advisory, non-binding. Paragraph numbers and quotes were read from the
**proposed (consultation) text**; the final text (issued 20 Jul 2026 per most sources; 31 Jul per
HLC) was not read — only law-firm summaries of it were.

**Controls (5):** para 9.3 (high), para 9.5 (high), para 9.1 (medium), para 7.4/7.5/8.1
(medium), para 9.4 (low).

**Pages actually opened:**
- https://files.app.optical.gov.sg/pdpc/production/assets/ceb45ef8-294d-4b45-be35-e884d578fd8d.pdf (proposed GenAI guidelines)
- https://www.pdpc.gov.sg/organisations/regulations-decisions/regulatory-guidance/advisory-guidelines-on-use-of-personal-data-in-generative-ai (titles only rendered)
- https://www.rajahtannasia.com/viewpoints/pdpc-proposes-new-advisory-guidelines-on-use-of-personal-data-in-generative-ai/
- https://www.allenandgledhill.com/sg/publication/articles/33126/pdpc-consults-on-proposed-advisory-guidelines-on-use-of-personal-data-in-generative-ai
- https://www.stephensonharwood.com/insights/singapores-pdpc-issues-new-advisory-guidance-on-use-of-personal-data-in-generative-ai/
- https://www.hlc.com/en/publications/singapores-pdpc-finalises-guidance-on-the-use-of-personal-data-in-generative-ai
- https://aitoolkit.rajahtannasia.com/2026/07/30/navigating-the-collection-and-use-of-personal-data-in-generative-ai-development-pdpc-issues-advisory-guidelines/
- https://alder.com.sg/pdpcs-final-advisory-guidelines-on-generative-ai-what-singapore-businesses-need-to-know/
- https://digitalpolicyalert.org/event/40305-personal-data-protection-commission-closes-consultation-on-proposed-advisory-guidelines-on-use-of-personal-data-in-generative-ai
- https://iapp.org/news/a/singapore-launches-ai-training-data-guidelines-expands-pets-resources
- https://www.lw.com/en/insights/singapore-ai-guidance-governance-data-protection-and-legal-responsibility

**Caveats:**
- Final issue date: 20 Jul 2026 (Stephenson Harwood, Digital Policy Alert, Alder); HLC says
  31 Jul; IAPP reported the launch on 23 Jul. Consultation close: 1 Jul 2026 (LW says 14 Jul).
- Reported final-version changes (web-scraping notification proposal dropped in favour of a
  DPIA/written record; more digital-barrier examples; no template for AI-Specific
  Notifications) do not touch the cited paragraphs, but numbering may have shifted.
- The guidelines do not explicitly treat employee use of consumer AI tools, cross-border
  transfer to overseas model endpoints, or breach notification — those remain PDPA matters.
- No separate 2025 PDPC guide on employee AI use was found (July 2025 PDPC releases were the
  Global AI Assurance Sandbox, PETs Adoption Guide, LLM testing Starter Kit and DPTM SS 714:2025).

**Owner checklist:**
- [ ] Obtain the final guidelines PDF from pdpc.gov.sg and re-check paras 7.4, 7.5, 8.1, 8.3,
      9.1, 9.3, 9.4, 9.5 — numbers and quoted wording.
- [ ] Confirm the final issue date and update the displayName/header if needed.

---

## 3. `pdpc_ai_rds.yaml` — PDPC_AI_RDS — Advisory Guidelines on Use of Personal Data in AI Recommendation and Decision Systems (Mar 2024)

**Confidence: HIGH.** Official PDPC PDF opened and read; advisory and non-binding (para 2.2);
explicitly excludes generative AI.

**Controls (4):** para 11.2 (medium), para 7.2 (medium), para 7.1 (low), para 10.2 (low).

**Pages actually opened:**
- https://www.pdpc.gov.sg/-/media/files/pdpc/pdf-files/advisory-guidelines/advisory-guidelines-on-the-use-of-personal-data-in-ai-recommendation-and-decision-systems.pdf
- https://www.pdpc.gov.sg/guidelines-and-consultation/2024/02/advisory-guidelines-on-use-of-personal-data-in-ai-recommendation-and-decision-systems
- https://www.twobirds.com/en/insights/2024/singapore/pdpc-advisory-guidelines-on-the-use-of-personal-data-in-ai-recommendation-and-decision-systems
- https://www.rajahtannasia.com/viewpoints/pdpc-issues-advisory-guidelines-on-use-of-personal-data-in-ai-recommendation-and-decision-systems/
- https://www.drewnetworkasia.com/newsroom/singapore-pdpc-update-advisory-guidelines-on-use-of-personal-data-in-ai-recommendation-and-decision-systems-now-available/

**Caveats:**
- para 10.2 is paraphrased (written policies, pre-emptively published), not quoted verbatim.
- The Business Improvement (5.1(a)) and Research (6.2(b)) exceptions are consent exceptions and
  were left out on purpose.

**Owner checklist:**
- [ ] Spot-check paras 7.1, 7.2, 7.10, 10.2, 11.2 (and fn 21) in the PDF.

---

## 4. `csa_ai_security.yaml` — CSA_AI_SEC — Guidelines and Companion Guide on Securing AI Systems (Oct 2024)

**Confidence: HIGH.** Both official CSA PDFs fetched and read; voluntary.

**Controls (7):** Guidelines §2.3 (high), §2.1 (high), §4.1 (high), §2.4 (medium); Companion
Guide 3.1.3 (medium), 2.3.4 (medium), 2.1.6 (medium).

**Pages actually opened:**
- https://isomer-user-content.by.gov.sg/36/42140c27-030f-4bb9-b4c7-b543ad2ddad4/guidelines-on-securing-ai-systems_2024-10-15.pdf
- https://isomer-user-content.by.gov.sg/36/3cfb3cd5-0228-4d27-a596-3860ef751708/Companion%20Guide%20on%20Securing%20AI%20Systems.pdf
- https://www.csa.gov.sg/resources/publications/guidelines-and-companion-guide-on-securing-ai-systems

**Caveats:**
- CSA page metadata shows a last update of 28 Sep 2026 and a re-uploaded Guidelines file, but no
  revised edition was identified — content appeared to be the Oct 2024 text.
- The Companion Guide is "community-driven" and "not mandatory, prescriptive nor exhaustive".

**Owner checklist:**
- [ ] Confirm the re-uploaded Guidelines file is unchanged from 15 Oct 2024.
- [ ] Spot-check Companion Guide sub-section numbers 2.1.2, 2.1.4, 2.1.6, 2.2.1, 2.3.1, 2.3.2,
      2.3.4, 2.4.1, 2.4.2, 3.1.3, 4.1.1.

---

## 5. `csa_agentic_ai.yaml` — CSA_AGENTIC — Securing Agentic AI, Addendum v1.0 (Jun 2026)

**Confidence: HIGH** for control numbers and quotes (official v1.0 PDF fetched); **MEDIUM** on
what changed between the Oct 2025 draft and the 17 Jun 2026 final, and on Control 4.4's exact
wording.

**Controls (10):** Control 2.6 (critical), 2.7, 2.8, 2.4, 4.3, 3.3, 2.3 (high), 2.1, 2.9, 4.4
(medium).

**Pages actually opened:**
- https://www.csa.gov.sg/resources/publications/addendum-on-securing-ai-systems/
- https://isomer-user-content.by.gov.sg/36/9f94d260-ea28-4b12-bf59-cc77cbc0a421/Securing%20Agentic%20AI–An%20Addendum%20to%20the%20Guidelines%20and%20Companion%20Guide%20on%20Securing%20AI%20Systems%20(20.pdf
- https://www.csa.gov.sg/news-events/press-releases/csa-releases-an-addendum-to-support-system-owners-in-securing-agentic-ai-system/
- https://www.csa.gov.sg/news-events/speeches/opening-address-by-minister-josephine-teo-at-hlp--ai--on-22-oct-2025/
- https://www.bakermckenzie.com/en/insight/publications/2026/07/singapore-csas-addendum-on-securing-agentic-ai-systems
- https://www.rajahtannasia.com/viewpoints/csa-releases-addendum-on-securing-agentic-ai/

**Caveats:**
- The CSA publication page still carries the consultation-era description text.
- Neither Baker McKenzie nor Rajah & Tann identify what changed between draft and final.
- Control 4.4 (human-in-the-loop): the fetch summarised this control; the exact halt/terminate/
  override wording was not verified — the pack's description is a paraphrase.
- The addendum contains no explicit "shadow" or "unsanctioned" agent language; mapping is by
  control substance.

**Owner checklist:**
- [ ] Confirm the fetched PDF is the final v1.0 (17 Jun 2026) and that §4.2 control numbering
      matches (2.1, 2.3, 2.4, 2.6, 2.7, 2.8, 2.9, 3.3, 4.3, 4.4).
- [ ] Read Control 4.4 in full and replace the paraphrase with the actual wording.

---

## 6. `imda_mgf_agentic.yaml` — IMDA_MGF_AGENTIC — Model AI Governance Framework for Agentic AI (v1.0 Jan 2026, updated May 2026)

**Confidence: HIGH** for v1.0 section numbers (official IMDA PDF fetched); **MEDIUM** for the
May 2026 updated edition, which was read only through the IMDA factsheet and law-firm notes.

**Controls (6):** 2.1.2 (high), 2.3.1 (high), 2.3.3 (high), 2.2.2 (medium), 2.4.2 (medium),
2.3.2 (low).

**Pages actually opened:**
- https://www.imda.gov.sg/-/media/imda/files/about/emerging-tech-and-research/artificial-intelligence/mgf-for-agentic-ai.pdf
- https://www.imda.gov.sg/resources/press-releases-factsheets-and-speeches/press-releases/2026/new-model-ai-governance-framework-for-agentic-ai
- https://www.imda.gov.sg/resources/press-releases-factsheets-and-speeches/factsheets/2026/updated-model-ai-governance-framework-for-agentic-ai
- https://www.bakermckenzie.com/en/insight/publications/2026/06/singapore-imda-updates-model-ai-governance-framework-for-agentic-ai
- https://www.allenandgledhill.com/sg/publication/articles/33285/imda-updates-model-ai-governance-framework-for-agentic-ai
- https://www.lw.com/en/insights/singapore-ai-guidance-governance-data-protection-and-legal-responsibility

**Caveats:**
- Version label of the 20 May 2026 update is unconfirmed — "Version 1.5" per Latham & Watkins
  only (which also mentions a 5 Jun 2026 update); IMDA pages opened did not show a label.
- v1.0 has no explicit inventory/registry practice; 2.1.2 is the nearest hook.
- The IMDA May 2026 discussion paper on legal responsibility for AI agents was not opened.
- 2.2.1 (third-party contracts) was not packaged — see "Deliberately left out" below.

**Owner checklist:**
- [ ] Confirm the version label of the May 2026 edition and update the displayName.
- [ ] Re-check section numbers 2.1.2, 2.2.2, 2.3.1, 2.3.2, 2.3.3, 2.4.2 in the updated PDF.

---

## 7. `mas_trm.yaml` — MAS_TRM — Technology Risk Management Guidelines (Jan 2021)

**Confidence: MEDIUM** overall (unofficial reproduction, corroborated by vendor papers for 3.3.2,
3.4.x, 9.x, 11.x, 12.x, 13.x); **LOW** for 6.4.2 and 6.5.2, which are not in the corroborated
list. Supervisory expectation, not binding.

**Controls (8):** 11.1.5 (critical), 6.5.2 (high), 3.3.2 (high), 11.1.1–11.1.2 (high), 9.2.2
(medium), 12.2.2 (medium), 3.4.2 (medium), 6.4.2 (low).

**Pages actually opened:**
- https://gist.github.com/nth-block/79e0858f3517bae24d98805263a44c1d (unofficial reproduction of TRM 2021)
- https://www.oracle.com/a/ocom/docs/mas-tvra-advisory.pdf
- https://assets.beyondtrust.com/assets/documents/Addressing-the-MAS-TRM-with-Privileged-Access-Management-August-2021.pdf
- https://4711332.fs1.hubspotusercontent-na1.net/hubfs/4711332/eBooks/White%20Paper_Compliance_with_MAS_Technology_Risk_Management_Guidelines.pdf
- https://www.pwc.com/sg/en/risk-assurance/assets/revisions-to-guidelines-on-technology-risk-management.pdf
- https://cpl.thalesgroup.com/compliance/apac/mas-trm-guidelines-compliance-singapore
- https://www.globalfinregblog.com/2021/02/singapore-mas-revises-technology-risk-management-guidelines/
- https://www.globalcompliancenews.com/2021/02/16/singapore-mas-revises-technology-risk-management-guidelines280121/
- https://www.allenandgledhill.com/sg/publication/articles/12492/mas-issues-new-legally-binding-requirements-to-raise-cybersecurity-standards-and-strengthen-cyber-resilience-of-financial-sector
- https://www.drewnapier.com/DrewNapier/media/DrewNapier/28-Apr-20-Bringing-hygiene-online-the-MAS-notice-on-cyber-hygiene.pdf
- https://practiceguides.chambers.com/practice-guides/cybersecurity-2026/singapore
- https://resourcehub.bakermckenzie.com/en/resources/global-data-and-cyber-handbook/asia-pacific/singapore/topics/security-requirements-and-breach-notification

**Caveats:**
- The MAS PDF was not opened; every paragraph number comes from the unofficial GitHub
  reproduction. Every review marker in this pack says "paragraph number from unofficial
  reproduction".
- The binding MAS TRM Notices (1-hour incident notification, 14-day root-cause report) and the
  MAS Notice on Cyber Hygiene (2019; notice numbers not confirmed) are not mapped.

**Owner checklist:**
- [ ] Download the TRM Guidelines PDF from mas.gov.sg and verify 3.3.1–3.3.2, 3.4.1–3.4.3,
      4.2.1, 6.4.2–6.4.3, 6.5.1–6.5.2, 9.2.1–9.2.2, 11.1.1–11.1.5, 12.2.1–12.2.5.
- [ ] Decide whether the TRM Notices / Cyber Hygiene Notice deserve a separate binding pack.

---

## 8. `mas_airg.yaml` — MAS_AIRG — Proposed Guidelines on AI Risk Management (consultation Nov 2025 — NOT FINAL)

**Confidence: LOW.** Not finalised as of 2026-09-29; consultation paper not opened; paragraph
numbers not reliably known. controlIds are `PROPOSED-*` placeholders. Severity capped at medium.

**Controls (5):** PROPOSED-Identification, PROPOSED-Inventory, PROPOSED-ThirdPartyAI,
PROPOSED-Monitoring (medium), PROPOSED-Materiality (low).

**Pages actually opened:**
- https://www.twobirds.com/en/insights/2026/singapore/mas-consults-on-proposed-guidelines-on-artificial-intelligence-risk-management
- https://compliance.waystone.com/wp-content/uploads/2026/01/WCS_Consultation-MAS-Guidelines-on-Artificial-Intelligence-Risk-Management.pdf
- https://www.asifma.org/wp-content/uploads/2026/02/2026-01-31-asifma-response-to-mas-ai-risk-mgmt-guidelines-cp.pdf
- https://www.bakermckenzie.com/en/insight/publications/alerts/2025/11/singapore-mas-consults-on-ai-risk
- https://connectontech.bakermckenzie.com/singapore-mas-publishes-consultation-paper-on-proposed-guidelines-on-ai-risk-management-for-financial-institutions/
- https://www.stephensonharwood.com/insights/understanding-the-mas-consultation-paper-on-ai-risk-management-for-financial-institutions/
- https://www.pertamapartners.com/insights/singapore-mas-ai-risk-management-guidelines-financial-services
- https://www.aigl.blog/consultation-paper-on-guidelines-on-artificial-intelligence-risk-management-airg/
- https://financialregulation.linklaters.com/post/102lw8a/singapore-mas-proposes-comprehensive-ai-risk-management-guidelines-for-financial (thin)
- https://covarage.com/guides/regulatory-change/mas-airg-imda-mgf-eu-ai-act-singapore-sme-compliance-timeline
- https://www.hubbis.com/news/mas-partners-industry-to-develop-ai-risk-management-toolkit-for-the-financial-sector (mirror of MAS 24 Mar 2026 release)
- https://www.rajahtannasia.com/viewpoints/best-practices-for-ai-governance-and-risk-management-published-for-singapore-financial-sector/
- https://www.licentium.io/post/mas-mindforge-ai-risk-management-toolkit-singapore-financial-sector-2026
- https://www.hlc.com/en/publications/from-principles-to-practice-maturing-ai-supervision-in-singapores-financial-sector
- https://www.bakermckenzie.com/en/insight/publications/2026/07/singapore-mas-publishes-agentic-ai-safeguards-for-financial-institutions
- https://www.ashurstperkinscoie.com/en/insights/mas-safr-explained-singapores-runtime-governance-standard-for-agentic-ai-in-finance/
- https://www.hubbis.com/news/mas-partners-industry-to-develop-safeguards-for-ai-agents-in-finance

**Caveats:**
- Issue date 13 Nov 2025 (Bird & Bird / Waystone / Baker McKenzie) vs 15 Nov (Rajah & Tann).
- Status: MAS "presently reviewing responses" (24 Mar 2026); HLC (3 Jun 2026) "final Guidelines
  remain pending"; no final issuance found. Proposed transition: 12 months after issue.
- ASIFMA cites paras 4.11, 4.24 and footnotes 10/21 — not verified against the paper; a "kill
  switches" expectation is referenced by ASIFMA with unconfirmed wording.
- Companion voluntary outputs (MindForge Toolkit, ABS GenAI Guardrails Handbook, MAS SAFR v1.0)
  are described in the header but not mapped.

**Owner checklist:**
- [ ] Watch for MAS final issuance; when issued, replace `PROPOSED-*` ids with paragraph
      numbers, re-quote wording from the final text, revisit severities and note the transition
      end date.
- [ ] Until then, keep `reviewRequired:true` for this framework.

---

## 9. `cyber_essentials_sg.yaml` — CSA_CYBER_ESSENTIALS — Cyber Essentials mark v202503 (Mar 2025)

**Confidence: HIGH.** Official CSA PDF fetched and read; voluntary certification mark. The only
Singapore instrument that uses "shadow AI" verbatim.

**Controls (5):** A.2.4(a) (high), A.5.4(a) (high), A.3.4(d) (high), A.6.4(g) (medium),
A.9.4(a) (medium).

**Pages actually opened:**
- https://isomer-user-content.by.gov.sg/36/822679b6-320b-4dc2-8f07-3f694f9a53f7/20250307%20Cyber%20Essentials%20V202503.pdf
- https://www.csa.gov.sg/news-events/press-releases/csa-s-cyber-essentials-and-cyber-trust-marks-expanded-to-include-cloud-security--artificial-intelligence-and-operational-technology/

**Caveats:**
- The AI clauses live in the assessment templates; section headings were not recorded in the
  brief and are deliberately not stated in the pack.
- The sibling Cyber Trust mark v202504 (also opened:
  https://isomer-user-content.by.gov.sg/36/2cff8d23-0f79-4477-9629-377d3bccbcaf/cyber-trust-v202504.pdf)
  is not packaged — see below.

**Owner checklist:**
- [ ] Spot-check clauses A.2.4(a), A.3.4(a), A.3.4(d), A.5.4(a), A.6.4(g), A.9.4(a) in the PDF.

---

## Deliberately left out (no pack written)

| Instrument | Why not packaged |
|---|---|
| Cybersecurity Act 2018 / Cybersecurity (Amendment) Act 2024 (in force 31 Oct 2025) | Binding, but the Act text has no AI content; the only hook (expanded s 14 incident reporting for supplier systems interconnected with CII) applies to designated CII owners, not general shadow-AI events. Supplementary-report timeline (14 days vs 72 h / 30 days) is unconfirmed. Parts 3C/3D (ESCI, FDI) not commenced. Candidate for a later CII-specific pack. |
| IMDA / AI Verify Foundation Model AI Governance Framework for Generative AI (May 2024) | Principle-level dimensions (Accountability, Security, Incident Reporting, Testing and Assurance) without clause numbers suitable as controlIds. |
| AI Verify testing framework, Project Moonshot, LLM Starter Kit (Jan 2026), Global AI Assurance Sandbox | Assurance and testing toolkits, not control frameworks; the Starter Kit's §1.3.3 note on checking agent tool calls is the only near-control text. |
| CSA Cyber Trust mark v202504 | Sibling of Cyber Essentials; B.3.1, B.3.5 and AI Risk 8 could form a pack later. Left out to keep this round to the instruments requested. |
| MAS Notice on Cyber Hygiene (2019) and MAS TRM Notices | Binding, but notice numbers were not confirmed in the pages opened and the six Cyber Hygiene requirements have no AI-specific hook. |
| MAS SAFR white paper v1.0 (3 Jul 2026), MindForge Toolkit, ABS GenAI Guardrails Handbook | Voluntary industry outputs; SAFR "does not constitute regulatory guidance or supervisory expectations"; MindForge PDF returned 403. |
| IMDA Agentic MGF 2.2.1 (third-party contracts) and 2.4 transparency lines | Not in the requested control set; 2.4 wording is folded into the 2.4.2 description instead. |
| MDDI parliamentary reply (5 May 2026), GovTech AI-agent registry (Jun 2026), CSA Cyber Landscape 2025/2026 | Government statements, not control instruments; useful as quotable framing only. |

## Key uncertainties (carried over from the brief)

1. MAS AIRG not confirmed final — describe as "proposed, 12-month transition once issued".
2. PDPC GenAI Guidelines final date 20 Jul 2026 (one source says 31 Jul).
3. IMDA Agentic MGF update version label (v1.5 per Latham & Watkins only).
4. CII supplementary-report timeline (14 days vs 72 h / 30 days) and exact PDPA regulation
   numbers — re-check against SSO from a network that can reach it.
5. TRM paragraph numbers are from an unofficial reproduction, though 3.3.2, 3.4.x, 9.x, 11.x,
   12.x, 13.x are corroborated by Oracle/BeyondTrust/Silverfort papers.

## Global owner checklist for removing review markers

- [ ] For each pack, open the primary source named in the file header (not the secondary
      sources) and tick every clause number and quoted phrase.
- [ ] Remove a `review:` marker only after the clause it cites has been read in the primary
      source; never remove markers in bulk.
- [ ] Keep `MAS_AIRG` markers until MAS issues the final Guidelines.
- [ ] Keep `PDPC_GENAI` markers until the final (Jul 2026) text has been read.
- [ ] Keep `MAS_TRM` markers until the MAS PDF has been checked (6.4.2 and 6.5.2 first — they
      are the least corroborated).
- [ ] Re-run the YAML validation after any edit:
      `python3 -c "import yaml,glob; [yaml.safe_load(open(f)) for f in glob.glob('out/*.yaml')]; print('yaml ok')"`
