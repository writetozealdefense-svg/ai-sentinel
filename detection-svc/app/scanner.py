"""
Core scanner: orchestrates Presidio, YARA, and regex engines;
builds merged Finding list and redacted preview.
"""
from __future__ import annotations

import time
from collections import defaultdict
from pathlib import Path
from typing import List, Optional

from app.engines.presidio_engine import PresidioEngine
from app.engines.regex_engine import RegexEngine
from app.engines.yara_engine import YaraEngine
from app.models import Finding, ScanResult, ScanStats

# Directory containing YARA *.yar files (inside the container)
_RULES_DIR = Path("/app/rules")


class DetectionScanner:
    """
    Singleton-style scanner initialised once on application startup.
    All three engines are real and independently functional.
    """

    def __init__(self, rules_dir: Path = _RULES_DIR) -> None:
        self._presidio = PresidioEngine()
        self._yara = YaraEngine(rules_dir=rules_dir)
        self._regex = RegexEngine()

    # ------------------------------------------------------------------
    # Health / counts
    # ------------------------------------------------------------------

    @property
    def presidio_count(self) -> int:
        return self._presidio.recognizer_count

    @property
    def yara_count(self) -> int:
        return self._yara.rule_count

    @property
    def regex_count(self) -> int:
        return self._regex.pattern_count

    # ------------------------------------------------------------------
    # Scanning
    # ------------------------------------------------------------------

    def scan(
        self,
        content: str,
        language: str = "en",
        entity_types: Optional[List[str]] = None,
    ) -> ScanResult:
        t0 = time.monotonic()

        # Run all three engines
        presidio_findings = self._presidio.analyze(content, language=language, entity_types=entity_types)
        yara_findings = self._yara.scan(content)
        regex_findings = self._regex.scan(content, entity_types=entity_types)

        all_findings: List[Finding] = presidio_findings + yara_findings + regex_findings

        # Sort by start offset for consistent ordering
        all_findings.sort(key=lambda f: (f.start, -f.score))

        # Build redacted preview
        redacted = _build_redacted_preview(content, all_findings)

        # Stats
        by_category: dict[str, int] = defaultdict(int)
        highest = 0.0
        for f in all_findings:
            by_category[f.category.value] += 1
            if f.score > highest:
                highest = f.score

        duration_ms = int((time.monotonic() - t0) * 1000)

        stats = ScanStats(
            totalFindings=len(all_findings),
            byCategory=dict(by_category),
            highestScore=round(highest, 4),
            durationMs=duration_ms,
        )

        return ScanResult(
            findings=all_findings,
            redactedPreview=redacted,
            stats=stats,
        )


# ---------------------------------------------------------------------------
# Redaction helper
# ---------------------------------------------------------------------------

def _merge_spans(findings: List[Finding]) -> List[tuple[int, int, str]]:
    """
    Merge overlapping / adjacent spans into non-overlapping intervals.
    Returns list of (start, end, type) sorted by start, processing right-to-left
    in the caller allows in-place string replacement without offset drift.

    When spans overlap, the one with the highest score wins for the label;
    the merged interval is the union.
    """
    if not findings:
        return []

    # Sort by start, then by descending score so the best label is first
    sorted_f = sorted(findings, key=lambda f: (f.start, -f.score))

    merged: list[tuple[int, int, str]] = []
    cur_start = sorted_f[0].start
    cur_end = sorted_f[0].end
    cur_type = sorted_f[0].type

    for f in sorted_f[1:]:
        if f.start < cur_end:
            # Overlapping — extend the current span
            if f.end > cur_end:
                cur_end = f.end
            # Keep the label of the first (highest-score) span
        else:
            merged.append((cur_start, cur_end, cur_type))
            cur_start = f.start
            cur_end = f.end
            cur_type = f.type

    merged.append((cur_start, cur_end, cur_type))
    return merged


def _build_redacted_preview(content: str, findings: List[Finding]) -> str:
    """
    Replace each merged span with `<TYPE> [REDACTED]`.
    Works right-to-left so earlier offsets stay valid.
    """
    if not findings:
        return content

    spans = _merge_spans(findings)

    # Process right-to-left
    result = content
    for start, end, ftype in reversed(spans):
        placeholder = f"<{ftype}> [REDACTED]"
        result = result[:start] + placeholder + result[end:]

    return result
