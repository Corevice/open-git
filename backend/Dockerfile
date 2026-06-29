FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN test -f LICENSE || printf '%s\n' 'Apache-2.0' > LICENSE && \
    test -f licenses.json || printf '[]\n' > licenses.json

RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.20

RUN addgroup -g 1001 -S opengit && adduser -u 1001 -S opengit -G opengit

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/licenses.json ./licenses.json
COPY --from=builder /app/LICENSE ./LICENSE

USER opengit

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./server"]
