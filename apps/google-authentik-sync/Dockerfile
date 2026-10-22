FROM python:3.13-slim-bookworm@sha256:e32be0e55814a08977501ca863271d9bd02443710db8f2afaf8372e2c4508530
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
