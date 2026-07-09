"""
Custom regex detection engine.

Covers patterns not already caught by Presidio or YARA, providing an additional
layer of coverage and ensuring regex engine count > 0.

Each pattern is compiled once at module load.
"""
from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import List, Optional

from app.models import Category, Engine, Finding


@dataclass
class RegexPattern:
    name: str
    pattern: re.Pattern
    category: Category
    score: float
    entity_type: str
    rule_name: str


# ---------------------------------------------------------------------------
# Pattern registry — add more here as needed
# ---------------------------------------------------------------------------
_RAW_PATTERNS: list[tuple] = [
    # (name, regex_str, category, score, entity_type)
    # ── PCI: bank routing numbers ────────────────────────────────────────────
    # US ABA routing number: exactly 9 digits (starts with 0-3 for Federal Reserve district)
    # Enhancement (Phase 2): added to catch payroll CSV routing numbers alongside Presidio's
    # US_BANK_NUMBER and IBAN recognizers which cover account numbers and IBANs.
    (
        "us_aba_routing_number",
        r"\b(?:0[1-9]|[12][0-9]|3[0-2])[0-9]{7}\b",
        Category.PCI, 0.78, "US_ROUTING_NUMBER",
    ),
    (
        "slack_token",
        r"xox[baprs]-[0-9A-Za-z\-]{10,72}",
        Category.SECRET, 0.90, "SLACK_TOKEN",
    ),
    (
        "google_api_key",
        r"AIza[0-9A-Za-z\-_]{35}",
        Category.SECRET, 0.90, "GOOGLE_API_KEY",
    ),
    (
        "stripe_key",
        r"(?:sk|pk)_(?:test|live)_[0-9a-zA-Z]{24,}",
        Category.SECRET, 0.90, "STRIPE_KEY",
    ),
    (
        "generic_password_field",
        r'(?i)(?:password|passwd|pwd)\s*[=:]\s*["\']?(?!.*\bexample\b)[^\s"\']{8,}',
        Category.SECRET, 0.75, "PASSWORD_FIELD",
    ),
    (
        "us_ein",
        # US Employer Identification Number: NN-NNNNNNN
        r"\b[0-9]{2}-[0-9]{7}\b",
        Category.PII, 0.65, "US_EIN",
    ),
    (
        "ipv6_address",
        r"\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b",
        Category.PII, 0.80, "IPV6_ADDRESS",
    ),
    (
        "us_zip_code",
        r"\b[0-9]{5}(?:-[0-9]{4})?\b",
        Category.PII, 0.40, "US_ZIP_CODE",
    ),
    (
        "date_iso",
        # ISO-8601 date yyyy-mm-dd (not already caught as DATE_TIME by Presidio with spacy-sm)
        r"\b(19|20)\d{2}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])\b",
        Category.PII, 0.55, "DATE_ISO",
    ),
]


def _compile_patterns() -> List[RegexPattern]:
    compiled = []
    for (name, regex_str, category, score, entity_type) in _RAW_PATTERNS:
        try:
            compiled.append(
                RegexPattern(
                    name=name,
                    pattern=re.compile(regex_str),
                    category=category,
                    score=score,
                    entity_type=entity_type,
                    rule_name=name,
                )
            )
        except re.error:
            pass  # skip broken patterns rather than crashing startup
    return compiled


_PATTERNS: List[RegexPattern] = _compile_patterns()


class RegexEngine:
    """Custom regex engine — compiled patterns applied to the content string."""

    @property
    def pattern_count(self) -> int:
        return len(_PATTERNS)

    def scan(self, content: str, entity_types: Optional[List[str]] = None) -> List[Finding]:
        """Apply all regex patterns; return Findings for every match."""
        findings: List[Finding] = []
        for pat in _PATTERNS:
            if entity_types and pat.entity_type not in entity_types:
                continue
            for m in pat.pattern.finditer(content):
                findings.append(
                    Finding(
                        type=pat.entity_type,
                        category=pat.category,
                        engine=Engine.regex,
                        text=m.group(0),
                        start=m.start(),
                        end=m.end(),
                        score=pat.score,
                        ruleName=pat.rule_name,
                    )
                )
        return findings
