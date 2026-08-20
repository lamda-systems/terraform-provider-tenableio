# Pinned by digest-free tag on purpose: the project pins every tool version
# explicitly (see Dockerfile-local at the repo root) rather than tracking latest.
FROM python:3.13.7-slim-bookworm

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir  -r requirements.txt

COPY tenableio_mock ./tenableio_mock

# Bind all interfaces: inside a container the useful listener is not loopback.
EXPOSE 8080
CMD ["python", "-m", "tenableio_mock", "--host", "0.0.0.0", "--port", "8080"]
