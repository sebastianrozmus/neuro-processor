# Build from the repository root:
#   docker build -f deploy/share/Dockerfile -t neuro-share .
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY internal/share ./internal/share
COPY internal/shareserver ./internal/shareserver
COPY cmd/neuro-share ./cmd/neuro-share
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/neuro-share ./cmd/neuro-share \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/neuro-share /neuro-share
COPY --from=build --chown=65532:65532 /out/data /data
ENV SHARE_DATA=/data SHARE_ADDR=:8080
EXPOSE 8080
USER 65532:65532
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["/neuro-share", "health"]
ENTRYPOINT ["/neuro-share"]
