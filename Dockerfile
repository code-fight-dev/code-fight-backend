FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/backend ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/admin-bootstrap ./cmd/admin-bootstrap

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget && update-ca-certificates

COPY --from=builder /out/backend /app/backend
COPY --from=builder /out/admin-bootstrap /app/admin-bootstrap

EXPOSE 8080

CMD ["/app/backend"]
