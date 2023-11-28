FROM golang:latest AS builder
ARG APPLICATION
WORKDIR /app
COPY . .
RUN cd src && go mod tidy
RUN cd src && CGO_ENABLED=0 go build -a -installsuffix cgo -o ../bin/${APPLICATION}/main -ldflags="-s -w" cmd/${APPLICATION}/main.go

FROM alpine AS app
ARG APPLICATION
ARG CONFIG
ARG WALLET
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/${APPLICATION}/main ./
COPY --from=builder /app/conf/${CONFIG} ./
COPY --from=builder /app/artefacts/${WALLET} ./
ENV CONFIG=${CONFIG}

ENTRYPOINT ./main -c ${CONFIG}
