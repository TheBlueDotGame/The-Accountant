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

docker-all:
	docker compose up -d

docker-central-build:
	docker compose up -d --no-deps --build central-node

docker-validator-build:
	docker compose up -d --no-deps --build validator-node

docker-client-build:
	docker compose up -d --no-deps --build client-node

scan:
	govulncheck ./...

