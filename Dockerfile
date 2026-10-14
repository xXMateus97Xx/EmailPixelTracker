FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# docker build --build-arg VERSION=1.2.3 .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/api ./cmd/api

# Static image with CA certificates and time zone data, running as non-root.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
ENV HTTP_ADDR=:8080
EXPOSE 8080
# The image has no curl/wget: the binary checks /health/ready itself.
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=3 CMD ["/api", "-healthcheck"]
ENTRYPOINT ["/api"]
