build:
	GOOS=linux GOARCH=amd64 go build -o bin/linux_x86/central -ldflags="-s -w" cmd/central/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/linux_x86/validator -ldflags="-s -w" cmd/validator/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/linux_x86/client -ldflags="-s -w" cmd/client/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/linux_x86/emulator -ldflags="-s -w" cmd/emulator/main.go

	GOOS=linux GOARCH=arm64 go build -o bin/linux_arm/central -ldflags="-s -w" cmd/central/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/linux_arm/validator -ldflags="-s -w" cmd/validator/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/linux_arm/wallet -ldflags="-s -w" cmd/client/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/linux_arm/emulator -ldflags="-s -w" cmd/emulator/main.go

	GOOS=darwin GOARCH=arm64 go build -o bin/darwin_arm/central -ldflags="-s -w" cmd/central/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/darwin_arm/validator -ldflags="-s -w" cmd/validator/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/darwin_arm/wallet -ldflags="-s -w" cmd/client/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/darwin_arm/emulator -ldflags="-s -w" cmd/emulator/main.go

documentation:
	./gendocs.sh

generate-secret:
	./secret.sh

