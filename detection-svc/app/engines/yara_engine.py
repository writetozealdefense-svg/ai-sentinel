"""
YARA-based detection engine.

Compiles all *.yar files from the rules/ directory at startup.
Each YARA match is converted to a Finding with:
  - engine = "yara"
  - category from rule meta.category (SECRET or CODE)
  - ruleName = rule identifier
  - start/end derived from the match string offset within the content
"""
from __future__ import annotations

import os
import re
from pathlib import Path
from typing import List

import yara  # type: ignore

from app.models import Category, Engine, Finding

# Map meta.category string → Category enum
_CATEGORY_MAP: dict[str, Category] = {
    "SECRET": Category.SECRET,
    "CODE": Category.CODE,
    "PII": Category.PII,
    "PHI": Category.PHI,
    "PCI": Category.PCI,
    "OTHER": Category.OTHER,
}

# Default score per meta.severity
_SEVERITY_SCORE: dict[str, float] = {
    "critical": 0.95,
    "high": 0.85,
    "medium": 0.70,
    "low": 0.55,
}


class YaraEngine:
    """Loads and compiles YARA rules; runs matches against supplied content."""

    def __init__(self, rules_dir: str | Path = "/app/rules") -> None:
        rules_dir = Path(rules_dir)
        yar_files: dict[str, str] = {}

        for yar_path in sorted(rules_dir.glob("*.yar")):
            yar_files[yar_path.stem] = str(yar_path)

        if not yar_files:
            # Fall back to the directory YARA compilation (all *.yar in dir)
            self._rules = None
            self._rule_count = 0
        else:
            self._rules = yara.compile(filepaths=yar_files)
            # Count rules by inspecting the compiled rules object
            self._rule_count = sum(1 for _ in self._rules)

    @property
    def rule_count(self) -> int:
        return self._rule_count

    def scan(self, content: str) -> List[Finding]:
        """Scan content string with YARA rules; returns Finding list."""
        if self._rules is None:
            return []

        findings: List[Finding] = []
        content_bytes = content.encode("utf-8", errors="replace")

        try:
            matches: list = self._rules.match(data=content_bytes)
        except Exception:
            return []

        for match in matches:
            # Determine category from rule meta
            meta = match.meta if hasattr(match, "meta") else {}
            cat_str = meta.get("category", "OTHER").upper()
            category = _CATEGORY_MAP.get(cat_str, Category.OTHER)

            severity = meta.get("severity", "medium").lower()
            score = _SEVERITY_SCORE.get(severity, 0.70)

            rule_name = match.rule

            # Each match has a list of string_instances (YARA >= 4.x)
            if match.strings:
                for string_match in match.strings:
                    # string_match is a StringMatch object; iterate instances
                    for instance in string_match.instances:
                        raw_offset: int = instance.offset
                        matched_bytes: bytes = instance.matched_data

                        # Convert byte offset to char offset (UTF-8 safe)
                        start_char = len(content_bytes[:raw_offset].decode("utf-8", errors="replace"))
                        matched_text = matched_bytes.decode("utf-8", errors="replace")
                        end_char = start_char + len(matched_text)

                        findings.append(
                            Finding(
                                type=rule_name,
                                category=category,
                                engine=Engine.yara,
                                text=matched_text[:120],  # truncate very long matches
                                start=start_char,
                                end=end_char,
                                score=score,
                                ruleName=rule_name,
                            )
                        )
            else:
                # No string matches (condition-only rule) — emit a finding at offset 0
                findings.append(
                    Finding(
                        type=rule_name,
                        category=category,
                        engine=Engine.yara,
                        text="",
                        start=0,
                        end=0,
                        score=score,
                        ruleName=rule_name,
                    )
                )

        return findings
