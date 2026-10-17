FROM alpine:3.24.0

ARG MISE_VERSION=2026.9.18

ENV LANG=C.UTF-8 \
    PATH="/root/.local/bin:${PATH}"

# Install system packages
RUN apk add --no-cache \
    bash curl fd file git jq python3 ripgrep

# Install Claude Code
RUN set -eux; \
  curl -fsSL https://claude.ai/install.sh | bash; \
  claude --version

CMD ["bash"]
