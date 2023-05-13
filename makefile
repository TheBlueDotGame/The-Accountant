build:
	go build -o bin/central -ldflags="-s -w" cmd/central/main.go
	go build -o bin/validator -ldflags="-s -w" cmd/validator/main.go
	go build -o bin/wallet -ldflags="-s -w" cmd/wallet/main.go

documentation:
	./gendocs.sh

generate-secret:
	./secret.sh

