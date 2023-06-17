FROM golang:latest AS builder_client
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o client -ldflags="-s -w" cmd/client/main.go

FROM alpine AS client
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder_client /app/client ./
COPY --from=builder_client /app/setup_example.yaml ./
COPY --from=builder_client /app/test_wallet ./

ENTRYPOINT ["./client", "-c", "setup_example.yaml"]
#ENTRYPOINT ["tail", "-f", "/dev/null"]
