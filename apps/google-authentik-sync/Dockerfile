FROM python:3.14-slim-bookworm@sha256:945c2676dc2b18e0c1f16879244122c57c0ffede60006bf3df159c507a31bcae
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir --require-hashes -r requirements.txt
COPY sync.py test_sync.py ./
RUN python -m unittest discover -s /app -q
COPY --chmod=755 entrypoint.sh /usr/local/bin/container-entrypoint
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/container-entrypoint"]
CMD ["--dry-run"]
