build-local:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/notary -ldflags="-s -w" cmd/notary/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/helper -ldflags="-s -w" cmd/helper/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/client -ldflags="-s -w" cmd/client/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/emulator -ldflags="-s -w" cmd/emulator/main.go

build-all: build-local
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/notary -ldflags="-s -w" cmd/notary/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/helper -ldflags="-s -w" cmd/helper/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/client -ldflags="-s -w" cmd/client/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/emulator -ldflags="-s -w" cmd/emulator/main.go

	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/notary -ldflags="-s -w" cmd/notary/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/helper -ldflags="-s -w" cmd/helper/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/client -ldflags="-s -w" cmd/client/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/emulator -ldflags="-s -w" cmd/emulator/main.go

	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/notary -ldflags="-s -w" cmd/notary/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/helper -ldflags="-s -w" cmd/helper/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/client -ldflags="-s -w" cmd/client/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/emulator -ldflags="-s -w" cmd/emulator/main.go

	GOOS=linux GOARCH=arm GOARM=5 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/raspberry_pi_zero/notary -ldflags="-s -w" cmd/notary/main.go
	GOOS=linux GOARCH=arm GOARM=5 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/raspberry_pi_zero/helper -ldflags="-s -w" cmd/helper/main.go

build-tools:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/generator -ldflags="-s -w" -gcflags -m cmd/generator/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/wallet -ldflags="-s -w" -gcflags -m cmd/wallet/main.go

build-tools-all: build-tools
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/generator -ldflags="-s -w" cmd/generator/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/wallet -ldflags="-s -w" cmd/wallet/main.go

	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/generator -ldflags="-s -w" cmd/generator/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/wallet -ldflags="-s -w" cmd/wallet/main.go

	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/generator -ldflags="-s -w" cmd/generator/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/wallet -ldflags="-s -w" cmd/wallet/main.go

	GOOS=linux GOARCH=arm GOARM=5 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/raspberry_pi_zero/wallet -ldflags="-s -w" cmd/wallet/main.go

documentation:
	./gendocs.sh

generate-secret:
	./secret.sh

generate-protobuf:
	protoc --proto_path=protobuf --go_out=protobufcompiled --go_opt=paths=source_relative block.proto addresses.proto

run-notary:
	./bin/dedicated/notary -c setup_example.yaml &

run-helper:
	./bin/dedicated/helper -c setup_example.yaml &

run-client:
	./bin/dedicated/client -c setup_example.yaml &

emulate-subscriber:
	./bin/dedicated/emulator -c setup_example.yaml -d minmax.json subscriber &

emulate-publisher:
	./bin/dedicated/emulator -c setup_example.yaml -d data.json publisher

run-emulate: emulate-subscriber emulate-subscriber

run-all: run-notary run-client run-helper emulate-subscriber emulate-publisher

start: build-local run-all

docker-dependencies:
	docker compose up -f docker-compose.dependencies.yaml -d

# docker-up|down|logs all take into account the environment variable COMPOSE_PROFILES
# per https://docs.docker.com/compose/profiles/
# To run a complete demo with publisher and subscriber nodes, use:
#   COMPOSE_PROFILES=demo make docker-up|down|logs
# To run core services only, use:
#   make docker-up|down|logs
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-restart-notary:
	docker-compose up -d --no-deps --build notary-node

docker-restart-helper:
	docker-compose up -d --no-deps --build helper-node

docker-build-all: docker-build-notary docker-build-helper docker-build-client docker-build-subscriber docker-build-publisher

docker-build-notary:
	docker compose build notary-node

docker-build-helper:
	docker compose build helper-node

docker-build-client:
	docker compose build client-node

docker-build-subscriber:
	docker compose build subscriber-node

docker-build-publisher:
	docker compose build publisher-node

scan:
	govulncheck ./...

