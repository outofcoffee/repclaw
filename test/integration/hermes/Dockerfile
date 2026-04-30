# Builds the Hermes Agent image from a pinned upstream tag. Pinning
# isolates the integration test from upstream config-schema churn — bump
# HERMES_REF deliberately rather than tracking main.
#
# This is a thin wrapper around the upstream Dockerfile: we clone the
# repo at HERMES_REF and chain into upstream's build steps. Any drift in
# the build process is upstream's concern; the wrapper only owns the
# pinned ref and the entrypoint pass-through.

FROM debian:13.4 AS source

ARG HERMES_REF=v2026.4.23

RUN apt-get update && \
    apt-get install -y --no-install-recommends git ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --branch "${HERMES_REF}" \
    https://github.com/NousResearch/hermes-agent.git /src

# Stage 2 onwards mirrors upstream Dockerfile structure. Kept inline
# rather than `FROM ghcr.io/nousresearch/hermes-agent` because there is
# no published image on GHCR for hermes-agent at time of writing.

FROM ghcr.io/astral-sh/uv:0.11.6-python3.13-trixie@sha256:b3c543b6c4f23a5f2df22866bd7857e5d304b67a564f4feab6ac22044dde719b AS uv_source
FROM tianon/gosu:1.19-trixie@sha256:3b176695959c71e123eb390d427efc665eeb561b1540e82679c15e992006b8b9 AS gosu_source

FROM debian:13.4

ENV PYTHONUNBUFFERED=1
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/hermes/.playwright

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential nodejs npm python3 ripgrep ffmpeg gcc python3-dev \
        libffi-dev procps git openssh-client docker-cli tini && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -u 10000 -m -d /opt/data hermes

COPY --chmod=0755 --from=gosu_source /gosu /usr/local/bin/
COPY --chmod=0755 --from=uv_source /usr/local/bin/uv /usr/local/bin/uvx /usr/local/bin/

WORKDIR /opt/hermes

COPY --from=source /src/package.json /src/package-lock.json ./
COPY --from=source /src/web/package.json /src/web/package-lock.json web/
COPY --from=source /src/ui-tui/package.json /src/ui-tui/package-lock.json ui-tui/
COPY --from=source /src/ui-tui/packages/hermes-ink/package.json \
                   /src/ui-tui/packages/hermes-ink/package-lock.json \
                   ui-tui/packages/hermes-ink/

RUN npm install --prefer-offline --no-audit && \
    npx playwright install --with-deps chromium --only-shell && \
    (cd web && npm install --prefer-offline --no-audit) && \
    (cd ui-tui && npm install --prefer-offline --no-audit) && \
    npm cache clean --force

COPY --from=source --chown=hermes:hermes /src/ ./

RUN cd web && npm run build && \
    cd ../ui-tui && npm run build && \
    rm -rf node_modules/@hermes/ink && \
    rm -rf packages/hermes-ink/node_modules && \
    cp -R packages/hermes-ink node_modules/@hermes/ink && \
    npm install --omit=dev --prefer-offline --no-audit --prefix node_modules/@hermes/ink && \
    rm -rf node_modules/@hermes/ink/node_modules/react && \
    node --input-type=module -e "await import('@hermes/ink')"

USER root
RUN chmod -R a+rX /opt/hermes

RUN uv venv && uv pip install --no-cache-dir -e ".[all]"

ENV HERMES_WEB_DIST=/opt/hermes/hermes_cli/web_dist
ENV HERMES_HOME=/opt/data
ENV PATH="/opt/data/.local/bin:${PATH}"
VOLUME [ "/opt/data" ]
ENTRYPOINT [ "/usr/bin/tini", "-g", "--", "/opt/hermes/docker/entrypoint.sh" ]
