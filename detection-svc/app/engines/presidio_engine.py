"""
Presidio-based detection engine with custom PHI recognizers.

Uses AnalyzerEngine with the full default recognizer registry plus custom
recognizers for:
  - ICD-10 codes  (PHI)
  - Medical Record Numbers / MRN  (PHI)
  - Medical terminology (diagnosis, patient, prescription, etc.)  (PHI)
"""
from __future__ import annotations

import re
from typing import List, Optional

from presidio_analyzer import (
    AnalyzerEngine,
    PatternRecognizer,
    Pattern,
    RecognizerResult,
)
from presidio_analyzer.nlp_engine import NlpEngineProvider

from app.models import Category, Engine, Finding

# ---------------------------------------------------------------------------
# Entity → Category mapping
# ---------------------------------------------------------------------------
ENTITY_TO_CATEGORY: dict[str, Category] = {
    # PII
    "PERSON": Category.PII,
    "EMAIL_ADDRESS": Category.PII,
    "PHONE_NUMBER": Category.PII,
    "US_SSN": Category.PII,
    "LOCATION": Category.PII,
    "DATE_TIME": Category.PII,
    "IP_ADDRESS": Category.PII,
    "US_DRIVER_LICENSE": Category.PII,
    "US_PASSPORT": Category.PII,
    "US_ITIN": Category.PII,
    "URL": Category.PII,
    "NRP": Category.PII,
    # PCI
    "CREDIT_CARD": Category.PCI,
    "IBAN_CODE": Category.PCI,
    "US_BANK_NUMBER": Category.PCI,
    "CRYPTO": Category.PCI,
    # PHI — built-in
    "MEDICAL_LICENSE": Category.PHI,
    # PHI — custom (registered below)
    "ICD10_CODE": Category.PHI,
    "MRN": Category.PHI,
    "MEDICAL_TERM": Category.PHI,
}

DEFAULT_CATEGORY = Category.OTHER


def _category(entity_type: str) -> Category:
    return ENTITY_TO_CATEGORY.get(entity_type, DEFAULT_CATEGORY)


# ---------------------------------------------------------------------------
# Custom Presidio recognizers for PHI
# ---------------------------------------------------------------------------

def _build_icd10_recognizer() -> PatternRecognizer:
    """ICD-10 codes: letter + 2 digits + optional decimal + 1-4 chars, e.g. J18.9, Z00.00, M79.3."""
    pattern = Pattern(
        name="icd10_code",
        regex=r"\b[A-Z][0-9]{2}(?:\.[0-9A-Z]{1,4})?\b",
        score=0.75,
    )
    return PatternRecognizer(
        supported_entity="ICD10_CODE",
        patterns=[pattern],
        context=["diagnosis", "icd", "code", "condition", "disorder"],
    )


def _build_mrn_recognizer() -> PatternRecognizer:
    """Medical Record Number patterns: MRN: digits, or 7-10 digit standalone numbers
    preceded by 'mrn', 'medical record', 'patient id', 'chart #'.

    Enhancement (Phase 2): added mrn_inline pattern for column-value format (MRN-XXXXXXX)
    that appears in CSV data, and lowered context requirement so PHI fires on structured data.
    """
    patterns = [
        Pattern(
            name="mrn_labeled",
            regex=r"(?i)\b(?:mrn|medical[\s_-]*record[\s_-]*(?:number|num|no|#)?)[\s:=#\-]*([A-Z0-9]{5,12})\b",
            score=0.90,
        ),
        Pattern(
            name="mrn_patient_id",
            regex=r"(?i)\b(?:patient[\s_-]*id|chart[\s#]*)[\s:=#\-]*([A-Z0-9]{5,12})\b",
            score=0.85,
        ),
        # Inline column-value format: MRN-7741023 or MRN_7741023 (common in CSV exports)
        Pattern(
            name="mrn_inline",
            regex=r"\bMRN[-_][0-9]{5,12}\b",
            score=0.88,
        ),
    ]
    return PatternRecognizer(
        supported_entity="MRN",
        patterns=patterns,
        # No context required — MRN patterns are high-confidence on their own
        context=["patient", "chart", "record", "hospital", "clinic", "mrn"],
    )


def _build_medical_term_recognizer() -> PatternRecognizer:
    """Detect explicit medical/clinical terms that signal PHI context.

    Enhancement (Phase 2): removed mandatory context constraint so clinical terms
    in CSV rows reliably trigger PHI detection without requiring adjacent context words.
    Added broader clinical vocabulary covering ICD-10 code headers and pharma terms.
    """
    # Deny-list style recognizer: words that on their own suggest PHI
    return PatternRecognizer(
        supported_entity="MEDICAL_TERM",
        deny_list=[
            "diagnosis",
            "diagnosed",
            "prescription",
            "prescribed",
            "treatment",
            "prognosis",
            "pathology",
            "radiology",
            "oncology",
            "chemotherapy",
            "medication",
            "dosage",
            "patient",
            "discharge",
            "admission",
            "hospitalized",
            "icd10",
            "icd_10",
            "icd-10",
            "physician",
            "dosage",
            "referral",
            "nephrology",
            "psychiatry",
            "cardiology",
            "anticoagulation",
            "hypertension",
            "diabetes",
            "mellitus",
            "neoplasm",
        ],
        # No mandatory context — the clinical terms are self-evidently PHI-bearing
        context=["medical", "health", "clinical", "doctor", "hospital", "patient", "diagnosis",
                 "prescription", "medication", "icd", "mrn"],
    )


def _build_icd10_standalone_recognizer() -> PatternRecognizer:
    """ICD-10 codes in CSV column context: matches codes that appear after a comma
    or at the start of a line (column-value position in structured data).

    Enhancement (Phase 2): added a high-confidence inline ICD-10 recognizer that
    fires without requiring context words — needed for CSV scanning where the column
    header 'icd10_diagnosis' is not adjacent to the code value.
    """
    pattern = Pattern(
        name="icd10_csv_value",
        # ICD-10 format: single letter + 2 digits + optional dot + 1-4 alphanums
        # Anchored to appear after comma/space (CSV cell position) for precision
        regex=r"(?<=[,\s])[A-Z][0-9]{2}(?:\.[0-9A-Z]{1,4})?(?=[,\s\r\n]|$)",
        score=0.82,
    )
    return PatternRecognizer(
        supported_entity="ICD10_CODE",
        patterns=[pattern],
        context=["diagnosis", "icd", "code", "condition", "disorder", "icd10", "icd_10"],
    )


# ---------------------------------------------------------------------------
# Engine class
# ---------------------------------------------------------------------------

class PresidioEngine:
    """Wraps Presidio AnalyzerEngine; initialised once at startup."""

    def __init__(self) -> None:
        # Build NLP engine backed by spaCy en_core_web_sm
        # (en_core_web_sm is used instead of lg to keep the Docker image lean;
        # accuracy on entity boundaries is slightly lower but sufficient for PII scanning)
        provider = NlpEngineProvider(nlp_configuration={
            "nlp_engine_name": "spacy",
            "models": [{"lang_code": "en", "model_name": "en_core_web_sm"}],
        })
        nlp_engine = provider.create_engine()

        self._analyzer = AnalyzerEngine(nlp_engine=nlp_engine, default_score_threshold=0.3)

        # Register custom PHI recognizers
        self._analyzer.registry.add_recognizer(_build_icd10_recognizer())
        self._analyzer.registry.add_recognizer(_build_icd10_standalone_recognizer())
        self._analyzer.registry.add_recognizer(_build_mrn_recognizer())
        self._analyzer.registry.add_recognizer(_build_medical_term_recognizer())

    @property
    def recognizer_count(self) -> int:
        return len(self._analyzer.registry.recognizers)

    def analyze(
        self,
        text: str,
        language: str = "en",
        entity_types: Optional[List[str]] = None,
    ) -> List[Finding]:
        """Run Presidio analysis and return a list of Finding objects."""
        try:
            results = self._analyzer.analyze(
                text=text,
                language=language,
                entities=entity_types,  # None = all
                return_decision_process=False,
            )
        except Exception:
            return []

        findings: List[Finding] = []
        for r in results:
            cat = _category(r.entity_type)
            findings.append(
                Finding(
                    type=r.entity_type,
                    category=cat,
                    engine=Engine.presidio,
                    text=text[r.start:r.end],
                    start=r.start,
                    end=r.end,
                    score=round(r.score, 4),
                    ruleName=None,
                )
            )
        return findings
