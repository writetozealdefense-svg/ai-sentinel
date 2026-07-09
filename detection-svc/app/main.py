"""
detection-svc  —  FastAPI entrypoint.

Endpoints:
  GET  /healthz  — liveness probe with engine counts
  POST /scan     — full DLP scan (Presidio + YARA + regex)

Contract: contracts/openapi/dlp.yaml
"""
from __future__ import annotations

import logging
import os
from contextlib import asynccontextmanager
from pathlib import Path
from typing import AsyncIterator

from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse

from app.models import DetectionEngineStatus, DetectionHealth, ScanRequest, ScanResult
from app.scanner import DetectionScanner

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
log = logging.getLogger("detection-svc")

# -----------------------------------------------------------------------
# Globals — populated during lifespan startup
# -----------------------------------------------------------------------
_scanner: DetectionScanner | None = None

RULES_DIR = Path(os.getenv("RULES_DIR", "/app/rules"))


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    global _scanner
    log.info("Initialising detection engines (Presidio + YARA + regex) …")
    _scanner = DetectionScanner(rules_dir=RULES_DIR)
    log.info(
        "Engines ready — presidio=%d recognizers, yara=%d rules, regex=%d patterns",
        _scanner.presidio_count,
        _scanner.yara_count,
        _scanner.regex_count,
    )
    yield
    log.info("detection-svc shutting down")


app = FastAPI(
    title="ZealDefense AI Sentinel — Detection Service",
    version="0.1.0",
    description="DLP scanning: Presidio + YARA + custom regex",
    lifespan=lifespan,
)


# -----------------------------------------------------------------------
# Routes
# -----------------------------------------------------------------------

@app.get("/healthz", response_model=DetectionHealth, tags=["detection"])
async def healthz() -> DetectionHealth:
    if _scanner is None:
        raise HTTPException(status_code=503, detail="Scanner not initialised")
    return DetectionHealth(
        status="ok",
        engines=DetectionEngineStatus(
            presidio=_scanner.presidio_count,
            yara=_scanner.yara_count,
            regex=_scanner.regex_count,
        ),
    )


@app.post("/scan", response_model=ScanResult, tags=["detection"])
async def scan(request: ScanRequest) -> ScanResult:
    if _scanner is None:
        raise HTTPException(status_code=503, detail="Scanner not initialised")

    if not request.content:
        raise HTTPException(status_code=400, detail="content must not be empty")

    try:
        result = _scanner.scan(
            content=request.content,
            language=request.language or "en",
            entity_types=request.entityTypes,
        )
    except Exception as exc:
        log.exception("Scan failed: %s", exc)
        raise HTTPException(status_code=500, detail=str(exc))

    return result


# -----------------------------------------------------------------------
# Generic error handler
# -----------------------------------------------------------------------

@app.exception_handler(Exception)
async def generic_error_handler(request: Request, exc: Exception) -> JSONResponse:
    log.exception("Unhandled error: %s", exc)
    return JSONResponse(status_code=500, content={"error": "internal server error", "detail": str(exc)})
