FROM python:3.10-slim AS builder

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

ARG MIRROR_URL=""

COPY requirements.txt .

RUN --mount=type=cache,target=/root/.cache/pip \
    if [ -n "$MIRROR_URL" ]; then \
        pip install --prefix=/install -i "$MIRROR_URL" -r requirements.txt; \
    else \
        pip install --prefix=/install -r requirements.txt; \
    fi

FROM python:3.10-slim AS runner

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/install/bin:$PATH" \
    PYTHONPATH="/install/lib/python3.10/site-packages"

RUN addgroup --system appgroup && adduser --system --ingroup appgroup appuser

COPY --from=builder /install /install
COPY main.py .

RUN chown -R appuser:appgroup /app

USER appuser

CMD ["python", "main.py"]