FROM golang:latest AS builder
ARG APPLICATION
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o cmd/${APPLICATION}/main -ldflags="-s -w" cmd/${APPLICATION}/main.go

FROM alpine AS app
ARG APPLICATION
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/cmd/${APPLICATION}/main ./
COPY --from=builder /app/setup_example.yaml ./
COPY --from=builder /app/test_wallet ./
EXPOSE 8000

ENTRYPOINT ["./main", "-c", "setup_example.yaml"]
