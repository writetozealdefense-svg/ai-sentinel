"""Pydantic models for request/response matching contracts/openapi/dlp.yaml."""
from __future__ import annotations
from enum import Enum
from typing import Any, Dict, List, Optional
from pydantic import BaseModel, Field


class ContentType(str, Enum):
    text = "text"
    csv = "csv"
    code = "code"
    json = "json"


class Category(str, Enum):
    PII = "PII"
    PHI = "PHI"
    PCI = "PCI"
    SECRET = "SECRET"
    CODE = "CODE"
    OTHER = "OTHER"


class Engine(str, Enum):
    presidio = "presidio"
    yara = "yara"
    regex = "regex"


class ScanRequest(BaseModel):
    content: str = Field(..., description="Raw text or file content to scan")
    contentType: ContentType = ContentType.text
    filename: Optional[str] = None
    language: str = "en"
    entityTypes: Optional[List[str]] = None


class Finding(BaseModel):
    type: str
    category: Category
    engine: Engine
    text: str
    start: int
    end: int
    score: float = Field(ge=0.0, le=1.0)
    ruleName: Optional[str] = None


class ScanStats(BaseModel):
    totalFindings: int
    byCategory: Dict[str, int]
    highestScore: float
    durationMs: int


class ScanResult(BaseModel):
    findings: List[Finding]
    redactedPreview: str
    stats: ScanStats


class DetectionEngineStatus(BaseModel):
    presidio: int
    yara: int
    regex: int


class DetectionHealth(BaseModel):
    status: str = "ok"
    engines: DetectionEngineStatus
