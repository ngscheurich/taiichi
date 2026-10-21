ARG ALPINE_VERSION=3.24

FROM alpine:${ALPINE_VERSION} AS base

ARG MISE_VERSION=v2026.9.18

ENV LANG=C.UTF-8 \
    PATH="/root/.local/bin:${PATH}"

# Install system packages
RUN apk add --no-cache \
      bash ca-certificates curl git inotify-tools jq tzdata tar perl \
      ncurses-libs openssl zlib build-base autoconf automake libtool \
      m4 linux-headers ncurses-dev openssl-dev zlib-dev python3

# Install Claude Code
RUN set -eux; \
  curl -fsSL https://claude.ai/install.sh | bash; \
  claude --version

# Install mise-en-place
ENV PATH="/root/.local/share/mise/shims:${PATH}" \
    MISE_DATA_DIR=/root/.local/share/mise \
    MISE_TRUSTED_CONFIG_PATHS="/workspace"
RUN set -eux; \
    curl -fsSL https://mise.run | MISE_VERSION="$MISE_VERSION" sh; \
    mise --version && \
    echo 'eval "$(mise activate bash)"' >> ~/.bashrc

CMD ["bash"]
