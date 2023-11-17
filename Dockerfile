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
COPY --from=builder /app/${CONFIG} ./
COPY --from=builder /app/test_wallet ./
ENV CONFIG=${CONFIG}
EXPOSE 8000
EXPOSE 8020

ENTRYPOINT ./main -c ${CONFIG}
