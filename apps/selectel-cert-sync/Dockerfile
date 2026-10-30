FROM python:3.14-slim@sha256:cd89c30eda26ee911a4611b18680086e0ea42e6f5f8cbe653d07fa7958c9cec8
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir --require-hashes -r requirements.txt
COPY sync.py test_sync.py ./
RUN python -m unittest discover -s /app -q
COPY --chmod=755 entrypoint.sh /usr/local/bin/container-entrypoint
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/container-entrypoint"]
