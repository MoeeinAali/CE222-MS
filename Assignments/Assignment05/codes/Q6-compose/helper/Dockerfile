# Helper service (provided helper_server.py) - stdlib only, so a slim base is enough.
FROM python:3.12-alpine

WORKDIR /app

COPY helper_server.py ./helper_server.py

RUN addgroup -S helper && adduser -S -G helper -u 10001 helper
USER helper

# documentation only - this service is deliberately NOT port-mapped
EXPOSE 9090

# -u keeps stdout unbuffered so `docker compose logs` is live
CMD ["python", "-u", "helper_server.py"]
