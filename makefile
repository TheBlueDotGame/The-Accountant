start:
	go run cmd/central/main.go

build:
	go build -o bin/central cmd/central/main.go

documentation:
	./gendocs.sh