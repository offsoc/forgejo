FROM code.forgejo.org/oci/alpine:3.20
ARG RELEASE_VERSION=unkown
LABEL maintainer="contact@forgejo.org" \
      org.opencontainers.image.version="${RELEASE_VERSION}"
RUN mkdir -p /app/gitea
RUN ( echo '#!/bin/sh' ; echo "echo forgejo v$RELEASE_VERSION" ) > /app/gitea/forgejo-cli ; chmod +x /app/gitea/forgejo-cli
