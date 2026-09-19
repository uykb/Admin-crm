# Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/apeadmin-server ./cmd/server

# Runtime Stage
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /app/apeadmin-server ./apeadmin-server
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/uploads ./uploads
COPY --from=builder /app/internal/plugin/builtin ./internal/plugin/builtin
COPY --from=builder /app/release-package/frontend/dist ./frontend/dist

EXPOSE 8001

CMD ["./apeadmin-server", "-config", "configs/config.yaml"]
