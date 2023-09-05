build-local:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/central -ldflags="-s -w" cmd/central/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/validator -ldflags="-s -w" cmd/validator/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/client -ldflags="-s -w" cmd/client/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/emulator -ldflags="-s -w" cmd/emulator/main.go

build-all: build-local
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/central -ldflags="-s -w" cmd/central/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/validator -ldflags="-s -w" cmd/validator/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/client -ldflags="-s -w" cmd/client/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/emulator -ldflags="-s -w" cmd/emulator/main.go

	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/central -ldflags="-s -w" cmd/central/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/validator -ldflags="-s -w" cmd/validator/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/client -ldflags="-s -w" cmd/client/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/emulator -ldflags="-s -w" cmd/emulator/main.go

	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/central -ldflags="-s -w" cmd/central/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/validator -ldflags="-s -w" cmd/validator/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/client -ldflags="-s -w" cmd/client/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/emulator -ldflags="-s -w" cmd/emulator/main.go

build-tools:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/generator -ldflags="-s -w" cmd/generator/main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/dedicated/wallet -ldflags="-s -w" cmd/wallet/main.go

build-tools-all: build-tools
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/generator -ldflags="-s -w" cmd/generator/main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_x86/wallet -ldflags="-s -w" cmd/wallet/main.go
	
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/generator -ldflags="-s -w" cmd/generator/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_arm/wallet -ldflags="-s -w" cmd/wallet/main.go
	
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/generator -ldflags="-s -w" cmd/generator/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/darwin_arm/wallet -ldflags="-s -w" cmd/wallet/main.go

documentation:
	./gendocs.sh

generate-secret:
	./secret.sh

run-central:
	./bin/dedicated/central -c setup_example.yaml &

run-validator:
	./bin/dedicated/validator -c setup_example.yaml &

run-client:
	./bin/dedicated/client -c setup_example.yaml &

emulate-subscriber:
	./bin/dedicated/emulator -c setup_example.yaml -d minmax.json subscriber &

emulate-publisher:
	./bin/dedicated/emulator -c setup_example.yaml -d data.json publisher

run-emulate: emulate-subscriber emulate-subscriber

run-all: run-central run-client run-validator emulate-subscriber emulate-publisher

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

docker-build-all: docker-build-central docker-build-validator docker-build-client docker-build-subscriber docker-build-publisher

docker-build-central:
	docker compose build central-node

docker-build-validator:
	docker compose build validator-node

docker-build-client:
	docker compose build client-node

docker-build-subscriber:
	docker compose build subscriber-node

docker-build-publisher:
	docker compose build publisher-node

scan:
	govulncheck ./...

