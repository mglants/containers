FROM python:3.14-slim as base

# Setup env
ENV LANG C.UTF-8
ENV LC_ALL C.UTF-8
ENV PYTHONDONTWRITEBYTECODE 1
ENV PYTHONFAULTHANDLER 1


FROM base AS python-deps
ARG VERSION
# Install pipenv and compilation dependencies
RUN pip install pipenv
RUN apt-get update && apt-get install -y --no-install-recommends gcc git g++
RUN git clone --depth 1 --branch v${VERSION} https://github.com/BBQigniter/kube-vip-watcher.git /src
RUN chmod +x /src/*.py
RUN chmod +x /src/healthchecks/*.py
# Install python dependencies in /.venv
COPY Pipfile .
RUN PIPENV_VENV_IN_PROJECT=1 pipenv install --deploy


FROM base AS runtime
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS
LABEL xyz.glants.image.target_platform=$TARGETPLATFORM
LABEL xyz.glants.image.target_architecture=$TARGETARCH
LABEL xyz.glants.image.target_os=$TARGETOS
LABEL org.opencontainers.image.title="emonoda"
LABEL org.opencontainers.image.source="https://github.com/mdevaev/emonoda"
# Copy virtual env from python-deps stage
COPY --from=python-deps /.venv /.venv
COPY --from=python-deps /src/kube-vip-watcher.py /app/
COPY --from=python-deps /src/healthchecks /app/
COPY --from=python-deps /src/lib /app/lib/

ENV PATH="/.venv/bin:/app:$PATH"

# Create and switch to a new user
RUN useradd --create-home app
WORKDIR /home/app
USER app
ENTRYPOINT ["kube-vip-watcher.py"]
