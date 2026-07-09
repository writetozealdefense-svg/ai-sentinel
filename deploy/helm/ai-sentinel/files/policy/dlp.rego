package sentinel.dlp

import rego.v1

# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------

# True when the named category appears in input.categories
has_category(cat) if cat in input.categories

# ---------------------------------------------------------------------------
# BLOCK trigger conditions
# ---------------------------------------------------------------------------

_block_triggered if has_category("SECRET")
_block_triggered if has_category("PHI")
_block_triggered if has_category("PCI")
_block_triggered if {
    has_category("PII")
    input.totalFindings >= 5
}

# ---------------------------------------------------------------------------
# REDACT trigger conditions
# ---------------------------------------------------------------------------

_redact_triggered if {
    not _block_triggered
    has_category("PII")
    input.totalFindings < 5
}

# ---------------------------------------------------------------------------
# WARN trigger conditions
# ---------------------------------------------------------------------------

_warn_triggered if {
    not _block_triggered
    not _redact_triggered
    has_category("CODE")
}

# ---------------------------------------------------------------------------
# action — single string with precedence BLOCK > REDACT > WARN > ALLOW
# ---------------------------------------------------------------------------

_action := "BLOCK"  if _block_triggered
_action := "REDACT" if _redact_triggered
_action := "WARN"   if _warn_triggered
_action := "ALLOW"  if {
    not _block_triggered
    not _redact_triggered
    not _warn_triggered
}

# ---------------------------------------------------------------------------
# sanctioned label
# ---------------------------------------------------------------------------

_sanctioned_label := "unsanctioned AI system" if not input.sanctioned
_sanctioned_label := "sanctioned AI system"   if input.sanctioned

# ---------------------------------------------------------------------------
# reason fragments (partial set rules — all matching reasons are collected)
# ---------------------------------------------------------------------------

_reasons contains msg if {
    has_category("SECRET")
    msg := sprintf("SECRET data detected (%d finding(s)) sent to %s %s", [input.byCategory.SECRET, _sanctioned_label, input.provider])
}

_reasons contains msg if {
    has_category("PHI")
    msg := sprintf("PHI data detected (%d finding(s)) sent to %s %s", [input.byCategory.PHI, _sanctioned_label, input.provider])
}

_reasons contains msg if {
    has_category("PCI")
    msg := sprintf("PCI data detected (%d finding(s)) sent to %s %s", [input.byCategory.PCI, _sanctioned_label, input.provider])
}

_reasons contains msg if {
    has_category("PII")
    input.totalFindings >= 5
    msg := sprintf("bulk PII (%d) to %s %s", [input.totalFindings, _sanctioned_label, input.provider])
}

_reasons contains msg if {
    not _block_triggered
    has_category("PII")
    input.totalFindings < 5
    msg := sprintf("small PII volume (%d finding(s)) to %s %s — redact before sending", [input.totalFindings, _sanctioned_label, input.provider])
}

_reasons contains msg if {
    not _block_triggered
    not _redact_triggered
    has_category("CODE")
    msg := sprintf("source CODE detected (%d finding(s)) sent to %s %s — review recommended", [input.byCategory.CODE, _sanctioned_label, input.provider])
}

_reasons contains msg if {
    not _block_triggered
    not _redact_triggered
    not _warn_triggered
    msg := sprintf("no sensitive categories detected in content sent to %s %s", [_sanctioned_label, input.provider])
}

# ---------------------------------------------------------------------------
# Public rule — decision
# ---------------------------------------------------------------------------

decision := {"action": _action, "reasons": reasons_array} if {
    reasons_array := [r | r := _reasons[_]]
    count(reasons_array) > 0
}
