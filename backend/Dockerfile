# --- Stage 1: build (full Go toolchain, discarded afterwards) ---
FROM golang:1.27-alpine AS build
WORKDIR /src

# Copy only dependency files first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 -> fully static binary that runs on an image with no libc.
# Build every command (api, migrate) into /out; one image, different entrypoints.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

# --- Stage 2: runtime (tiny, no shell, runs as non-root) ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ /app/
EXPOSE 8080
ENTRYPOINT ["/app/api"]
