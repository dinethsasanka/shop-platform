FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o inventory-service .

FROM alpine:3.20
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/inventory-service .
USER appuser
EXPOSE 8082
ENTRYPOINT ["./inventory-service"]
