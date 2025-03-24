FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o openrouter-watcher main.go

FROM scratch AS runner

COPY --from=builder /app/openrouter-watcher /openrouter-watcher
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["./openrouter-watcher"]
