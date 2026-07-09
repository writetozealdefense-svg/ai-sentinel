"""
internal_data_pipeline.py  —  ZealCorp Internal Analytics Pipeline
WARNING: This file contains production credentials. Handle with care.
"""
from __future__ import annotations

import os
import json
import hashlib
import logging
import requests
from typing import Optional, List, Dict

# ─── DO NOT COMMIT ───────────────────────────────────────────────────────────
# AWS access key for prod analytics bucket (rotate quarterly)
AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"
AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
DATABASE_PASSWORD = "Sup3rS3cr3t!Prod2024"
INTERNAL_API_KEY = "api_key=zc-prod-1a2b3c4d5e6f7g8h9i0j"

# ─── Private key (used for service-to-service JWT signing) ───────────────────
_PRIVATE_KEY_PEM = """-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7o4qne60TB3wo
pHpSCNHJo4EXAMPLE000000FAKEKEY000000NOTREALKEYDATA==
-----END PRIVATE KEY-----"""

log = logging.getLogger(__name__)


class DataPipelineClient:
    """Connects to the ZealCorp internal analytics API."""

    BASE_URL = "https://api-internal.zealcorp.example.com/v2"

    def __init__(self, api_key: str = INTERNAL_API_KEY, timeout: int = 30) -> None:
        self.api_key = api_key
        self.timeout = timeout
        self._session = requests.Session()
        self._session.headers.update({
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        })

    def fetch_records(self, table: str, limit: int = 100) -> List[Dict]:
        """Fetch records from the internal analytics store."""
        url = f"{self.BASE_URL}/tables/{table}/records"
        resp = self._session.get(url, params={"limit": limit}, timeout=self.timeout)
        resp.raise_for_status()
        return resp.json().get("data", [])

    def push_event(self, event_type: str, payload: dict) -> bool:
        """Push a telemetry event to the ingestion endpoint."""
        url = f"{self.BASE_URL}/events"
        body = {"type": event_type, "payload": payload, "key": AWS_ACCESS_KEY_ID}
        resp = self._session.post(url, json=body, timeout=self.timeout)
        return resp.status_code == 202

    @staticmethod
    def _sign(data: str, secret: str = AWS_SECRET_ACCESS_KEY) -> str:
        return hashlib.sha256(f"{data}:{secret}".encode()).hexdigest()


def load_config(path: str = "/etc/zealcorp/pipeline.json") -> dict:
    """Load pipeline config; fall back to env vars if file missing."""
    try:
        with open(path) as fh:
            return json.load(fh)
    except FileNotFoundError:
        return {
            "db_password": os.getenv("DB_PASSWORD", DATABASE_PASSWORD),
            "aws_key": os.getenv("AWS_ACCESS_KEY_ID", AWS_ACCESS_KEY_ID),
        }


if __name__ == "__main__":
    cfg = load_config()
    client = DataPipelineClient(api_key=cfg.get("aws_key", INTERNAL_API_KEY))
    records = client.fetch_records("user_events", limit=500)
    log.info("Fetched %d records", len(records))
