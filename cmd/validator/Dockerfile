FROM golang:latest AS builder_validator
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o validator -ldflags="-s -w" cmd/validator/main.go

FROM alpine AS validator
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder_validator /app/validator ./
COPY --from=builder_validator /app/setup_example.yaml ./
COPY --from=builder_validator /app/test_wallet ./

ENTRYPOINT ["./validator", "-c", "setup_example.yaml"]
#ENTRYPOINT ["tail", "-f", "/dev/null"]
