FROM golang:latest AS builder_central
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o central -ldflags="-s -w" cmd/central/main.go

FROM alpine AS central
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder_central /app/central ./
COPY --from=builder_central /app/setup_example.yaml ./
EXPOSE 8000

ENTRYPOINT ["./central", "-c", "setup_example.yaml"]
#ENTRYPOINT ["tail", "-f", "/dev/null"]
