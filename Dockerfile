# syntax=docker/dockerfile:1

# Builder
FROM golang:1.21.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 go build -o /app -ldflags "-s -w"

# Runner
# hadolint ignore=DL3006
FROM gcr.io/distroless/static-debian11 AS runner

WORKDIR /app

COPY --from=builder /app /

CMD ["/images"]
