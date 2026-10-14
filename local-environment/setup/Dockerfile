# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.24.8-bookworm AS build
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends zip \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY lambda/management-device-and-world-data-lambda/ ./lambda/management-device-and-world-data-lambda/
COPY lambda/power-data-registration-lambda/ ./lambda/power-data-registration-lambda/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -eu; mkdir /artifacts; \
    for module in management-device-and-world-data-lambda power-data-registration-lambda; do \
      (cd "lambda/$module" && make zip GOARCH="$TARGETARCH"); \
      cp "lambda/$module/lambda_function.zip" "/artifacts/$module.zip"; \
    done; \
    case "$TARGETARCH" in \
      amd64) echo x86_64 > /artifacts/architecture ;; \
      arm64) echo arm64 > /artifacts/architecture ;; \
      *) echo "Unsupported architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    GOOS=linux GOARCH="$TARGETARCH" CGO_ENABLED=0 \
      go install -tags postgres -ldflags '-X main.Version=v4.18.3' \
      github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3; \
    find /go/bin -type f -name migrate -exec cp {} /artifacts/migrate \;

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends awscli ca-certificates postgresql-client \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /opt/data-collector
COPY --from=build /artifacts/migrate /usr/local/bin/migrate
COPY --from=build /artifacts/architecture /opt/lambda-architecture
COPY --from=build /artifacts/management-device-and-world-data-lambda.zip ./lambda/management-device-and-world-data-lambda/lambda_function.zip
COPY --from=build /artifacts/power-data-registration-lambda.zip ./lambda/power-data-registration-lambda/lambda_function.zip
COPY migrations/ ./migrations/
COPY local-environment/scripts/ ./local-environment/scripts/
COPY local-environment/setup/entrypoint.sh /usr/local/bin/enemaru-setup
RUN chmod +x /usr/local/bin/enemaru-setup
ENTRYPOINT ["/usr/local/bin/enemaru-setup"]
