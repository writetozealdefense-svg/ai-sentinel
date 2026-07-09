# ─────────────────────────────────────────────────────────────────────────────
# detection-svc  —  Python 3.11 / FastAPI / Presidio / YARA / spaCy
#
# spaCy model: en_core_web_sm  (smaller than lg; saves ~400 MB in the image;
#   accuracy on NER boundaries is slightly lower but fully acceptable for
#   PII/PHI scanning on structured data like CSV rows.  Switch to
#   en_core_web_lg here if higher recall is ever required.)
# ─────────────────────────────────────────────────────────────────────────────
FROM python:3.11-slim

# ---------- system deps for yara-python and spaCy ----------
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        automake \
        libtool \
        libssl-dev \
        libjansson-dev \
        libmagic-dev \
        pkg-config \
        git \
        wget \
    && rm -rf /var/lib/apt/lists/*

# ---------- pip: upgrade first ----------
RUN pip install --no-cache-dir --upgrade pip

# ---------- install yara-python before the rest (needs build deps) ----------
RUN pip install --no-cache-dir yara-python==4.5.1

# ---------- install app requirements ----------
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# ---------- download spaCy language model ----------
# en_core_web_sm: ~12 MB, sufficient for PII/PHI NER
RUN python -m spacy download en_core_web_sm

# ---------- copy application source ----------
COPY app/ ./app/
COPY rules/ ./rules/

# ---------- non-root user for security hardening ----------
RUN useradd -r -u 1001 -m appuser && chown -R appuser /app
USER appuser

EXPOSE 8000

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
