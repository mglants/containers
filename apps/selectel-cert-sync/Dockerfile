FROM python:3.13-slim@sha256:d98cc29d3266e535dd9da1099688d9cb3ca4753813571e071763d17725460962
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir --require-hashes -r requirements.txt
COPY sync.py test_sync.py ./
RUN python -m unittest discover -s /app -q
COPY --chmod=755 entrypoint.sh /usr/local/bin/container-entrypoint
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/container-entrypoint"]
