# syntax=docker/dockerfile:1

# Samakan versi minor dengan baris `go` di go.mod.
FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Distroless static tidak punya libc. Binary hasil CGO akan gagal dijalankan dengan pesan
# "no such file or directory" walaupun filenya ada (kejadian di compro).
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/server && \
    go build -trimpath -ldflags="-s -w" -o /out/doomsday-admin ./cmd/doomsday-admin

# Menggagalkan build saat itu juga, bukan menunggu container restart terus di VM.
RUN for bin in /out/api /out/doomsday-admin; do \
      go version -m "$bin" | grep -q 'CGO_ENABLED=0' \
        || { echo "$bin tidak dibuild dengan CGO_ENABLED=0" >&2; exit 1; }; \
    done

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /out/doomsday-admin /usr/local/bin/
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
