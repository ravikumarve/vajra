# Vajra — single static Go binary (no CGO runtime deps beyond libc).
# Local-first default is `go build -o vajra .` (no Docker required).
FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/vajra .

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/vajra /vajra
# Buyer mounts: policies + data dir (SQLite audit, config)
VOLUME ["/data"]
ENV VAJRA_POLICY_DIR=/data/policies \
    VAJRA_DATA_DIR=/data
ENTRYPOINT ["/vajra"]
CMD ["serve"]
