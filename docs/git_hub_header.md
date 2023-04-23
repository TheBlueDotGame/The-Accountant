# Computantis

[![Go](https://github.com/bartossh/Computantis/actions/workflows/go.yml/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/go.yml)
[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql)
[![pages-build-deployment](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment)

Computantis is a service that keeps track of transactions between wallets.
Each wallet has its own independent history of transactions. There is a set of rules allowing for transactions to happen.
Computantis is not keeping track of all transactions in a single blockchain but rather allows to keep transactions signed by an authority. A signed transaction is valid transaction only if the issuer and receiver of the transaction are existing within the system.

## Execute the server

0. Run database `docker compose up`.
1. Build the server `go build -o path/to/bin/central cmd/central/main.go`.
2. Create `server_settings.yaml` according to `server_settings_example.yaml` file in `path/to/bin/` folder.
3. Run `./path/to/bin/central`.

## Run for development

0. Run database `docker compose up`.
1. Create `server_settings.yaml` according to `server_settings_example.yaml` in the repo root folder.
2. Run `make run` or `go run cmd/central/main.go`.
