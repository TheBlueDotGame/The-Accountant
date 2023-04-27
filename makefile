start:
	go run cmd/central/main.go

build:
	go build -o bin/central -ldflags="-s -w" cmd/central/main.go
	go build -o bin/validator -ldflags="-s -w" cmd/validator/main.go

documentation:
	./gendocs.sh

generate-secret:
	./secret.sh

