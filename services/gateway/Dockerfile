FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway .

FROM alpine:3.20
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/gateway .
USER appuser
EXPOSE 8000
ENTRYPOINT ["./gateway"]