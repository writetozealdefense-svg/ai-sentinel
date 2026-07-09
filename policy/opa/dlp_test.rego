package sentinel.dlp_test

import rego.v1

# ---------------------------------------------------------------------------
# Scenario 1: PII x40, unsanctioned provider → BLOCK
# ---------------------------------------------------------------------------
test_block_bulk_pii if {
    result := data.sentinel.dlp.decision with input as {
        "provider": "chatgpt",
        "sanctioned": false,
        "categories": ["PII"],
        "totalFindings": 40,
        "highestScore": 0.99,
        "byCategory": {"PII": 40}
    }
    result.action == "BLOCK"
    count(result.reasons) > 0
}

# ---------------------------------------------------------------------------
# Scenario: PHI present → BLOCK (regardless of count)
# ---------------------------------------------------------------------------
test_block_phi if {
    result := data.sentinel.dlp.decision with input as {
        "provider": "copilot",
        "sanctioned": true,
        "categories": ["PHI"],
        "totalFindings": 1,
        "highestScore": 0.85,
        "byCategory": {"PHI": 1}
    }
    result.action == "BLOCK"
    count(result.reasons) > 0
}

# ---------------------------------------------------------------------------
# Scenario: PCI present → BLOCK
# ---------------------------------------------------------------------------
test_block_pci if {
    result := data.sentinel.dlp.decision with input as {
        "provider": "gemini",
        "sanctioned": false,
        "categories": ["PCI"],
        "totalFindings": 3,
        "highestScore": 0.92,
        "byCategory": {"PCI": 3}
    }
    result.action == "BLOCK"
    count(result.reasons) > 0
}

# ---------------------------------------------------------------------------
# Scenario: SECRET present → BLOCK
# ---------------------------------------------------------------------------
test_block_secret if {
    result := data.sentinel.dlp.decision with input as {
        "provider": "chatgpt",
        "sanctioned": false,
        "categories": ["SECRET"],
        "totalFindings": 2,
        "highestScore": 0.97,
        "byCategory": {"SECRET": 2}
    }
    result.action == "BLOCK"
    count(result.reasons) > 0
}

# ---------------------------------------------------------------------------
# Scenario: PII x2 (< 5) → REDACT
# ---------------------------------------------------------------------------
test_redact_small_pii if {
    result := data.sentinel.dlp.decision with input as {
        "provider": "chatgpt",
        "sanctioned": false,
        "categories": ["PII"],
        "totalFindings": 2,
        "highestScore": 0.80,
        "byCategory": {"PII": 2}
    }
    result.action == "REDACT"
    count(result.reasons) > 0
}

# ---------------------------------------------------------------------------
# Scenario: CODE only → WARN
# ---------------------------------------------------------------------------
test_warn_code_only if {
    result := data.sentinel.dlp.decision with input as {
        "provider": "copilot",
        "sanctioned": true,
        "categories": ["CODE"],
        "totalFindings": 5,
        "highestScore": 0.70,
        "byCategory": {"CODE": 5}
    }
    result.action == "WARN"
    count(result.reasons) > 0
}

# ---------------------------------------------------------------------------
# Scenario: no categories → ALLOW
# ---------------------------------------------------------------------------
test_allow_no_categories if {
    result := data.sentinel.dlp.decision with input as {
        "provider": "copilot",
        "sanctioned": true,
        "categories": [],
        "totalFindings": 0,
        "highestScore": 0.0,
        "byCategory": {}
    }
    result.action == "ALLOW"
    count(result.reasons) > 0
}
