FROM python:3.14-slim-bookworm@sha256:516b51cb9db2ef061e465c803e50f64361b82cefd91e525b889b967b83e83527
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
