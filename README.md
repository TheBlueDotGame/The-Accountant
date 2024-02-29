# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql) [![Go](https://github.com/bartossh/Computantis/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/bartossh/Computantis/actions/workflows/go.yml)

![Computantis Logo](https://github.com/bartossh/computantis/blob/main/artefacts/logo.png)

## Project overview

The Computantis is a backbone service for creating secure, reliable and performant solutions for transaction exchange and Byzantine fault-tolerant systems.


### Services and tools

The Computantis contains:
 - Notary Node - is the main service in the Computantis system, offers secure transfer of transactions within the system, securing data in the DAG structure
 - Webhooks Node - is a supporting node that offers webhooks.
 - Client Node - is wallet middleware that abstracts away the whole security complexity of request sending, signing transactions and validating transactions and can run alongside client application on the same machine or in the proximity of client application.
 - Wallet Tool - is a tool to manage, create and save wallets to different formats.
 - Generator Tool - is a helper for creating data for the emulator.
 - Emulator Tool - is a helper for development and testing that pretends to be the client application issuer and receiver sending data via the Computantis system.


### The Computantis system schema

![Computantis Diagram](https://github.com/bartossh/computantis/blob/main/artefacts/Computantis-diagram.png)

#### The Computantis system network.

Nodes pursue to discover the full network and connect to all available nodes.
If the direct connection between two nodes is impossible, it is not a problem as gossip about gossip protocol allows transmissions of messages between non-connected nodes via other nodes.
A blue node named A is a message issuer. This node commands all other receiver nodes to act based on the message.
All the green nodes are good actors in the network, while red nodes are bad actors.
When transferring the message via a bad actor, the bad actor receiving the message cannot corrupt the message as the message is cryptographically secured via ED25519 asymmetric signature.
The bad actor is required to send the message intact, or if the bad actor refuses to send the message, all the network nodes that know the node will constantly try to send the message to the bad actor due to the gossip about gossip protocol assuming this node isn't properly updated.
The message will reach all the nodes until there are more connections between nodes than bad actors.
Message cannot be replied to later as each message is secured by a unique transaction.
The message will be resent to other nodes only if receiving nodes acknowledge the message as cryptographically valid.

#### The transaction

The transaction is secured with the asymmetric key cryptography signature, it can use different protocols but for now, uses ED25519 asymmetric key signature.
The data are the core entity and all the other fields in the message are just for the data security.
The data are of no significant value to the Computantis system and can be encrypted or decrypted, it is not the case for the Computantis system to validate or use them.
The Computantis is responsible for securing data from corruption, saving transactions in an immutable distributed storage and ensuring data reaches the receiver.

#### The secure storage - DAG

DAG stands for Direct Acyclic Graph and is post-blockchain technology allowing faster operations, near zero transaction confirmation time and more democratic data distribution. An important factor is that securing the DAG is much less computationally expensive than the blockchain forging process and may use cheaper and less complicated hardware.
On the diagram, all the green boxes are the vertices that contain and cryptographically secure the transactions.
The red box is a leaf. Leaf when added to the graph needs to validate two of the existing leaves or a leaf and vertex with the highest weight.
Leaf when added to the graph is retransmitted to all the nodes in the network, so each of them can validate that leaf and transaction against DAGs. Because edges which specify one direction connection between the vertices are based on hashes of vertices all the good players will keep the same vertices and edges in the DAG.
The vertex is unique in the DAG and the inner transaction is unique per DAG. Sending the same transaction to many nodes will end up in a collision and only the first one, received by most of the nodes will be accepted and retransmitted, all the rest of the redundant transactions will be rejected.
Leaf isn't proving transaction validity until it becomes a vertex via an edge created from another leaf or vertex.

### Add-ons

Add-ons are a way to interact with the Computantis immune system. The immune system shall return an error when data do not meet the validity criteria.

Example add-ons are located in `src_py` folder. 

#### List of add-ons

1. Add-on tabular classification example is located in `addon_tabular_classification` folder.

- Project requires Python 3.

- Before running create (venv) environment in the folder:
```sh
python -m venv venv
```
- Activate venv:

```sh 
source venv/bin/activate
```
- Run python server via shell script:
```sh
./start.py
```

## Development

### The core rules 

#### Golden rule - creating genesis

When creating a genesis node that creates a Genesis vertex, this node must use a wallet to create a genesis transaction only once. This wallet cannot be used again. 
The genesis transaction receiver should be a separate wallet having all created tokens in the genesis transaction. This receiver is then responsible for distributing tokens to other wallets.
Transferring funds from Genesis wallet to self will fail. Transferring funds to the Genesis wallet by any other wallet will fail too.

#### Silver rule - transferring founds

Funds can be transferred by the issuer via any node excluding the node that is owned by the issuer.
This is so the transaction will be sealed in vertex by two different cryptographic keys - to separate wallets.

### Only dependencies

To develop and run node with only dependencies:

 - Start all the dependencies:
```sh
docker compose up -f docker_deployment/docker-compose.dependencies.yaml -d
```
or use a makefile:
```sh
make docker-dependencies
```

- Then run node with `go` and passing `setup_bare.yaml` configuration. 
```sh
cd src && go run cmd/node/main.go -c ../conf/setup_bare.yaml
```

### Dependencies with network

- Start all dependencies and run three notary nodes:
```sh
docker compose -f docker_deployment/docker-compose.yaml up -d
```
- To restart the notary genesis node after the code change run:
```sh
docker compose up --no-deps -build notary-node-genessis -d
```

### Dependencies with emulation tests
- Start all dependencies and run three notary nodes:
```sh
docker compose -f docker_deployment/docker-compose.yaml --profile demo up-d
```
- To restart the notary node after the code change run:
```sh
docker compose up --no-deps -build notary-node-genessis -d
```

### Building GRPC dependencies and compiling packages

Generate protobuf files with:
```sh
protoc --proto_path=protobuf --go-grpc_out=src/protobufcompiled --go_out=src/protobufcompiled --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative  computantistypes.proto wallet.proto gossip.proto notary.proto webhooks.proto
```

### Building binaries

Run in terminal to run services in separate docker containers:

- all services
```sh
make build-all
```

### Vulnerability scanning.

Install govulncheck to perform vulnerability scanning  `go install golang.org/x/vuln/cmd/govulncheck@latest`.

Run 

```sh
govulncheck src/...
```

### Wallet C implementation

C implementation of the wallet is located in `src_c` folder.

#### Development

C version of client-node isn't cross platform.
This software is developed to be run on Linux OS and is tested for x86_64 Linux 5.15.0-76-generic kernel version, but it runs on aarch64 architecture too.
This software was tested with `gcc` compiler and while it might work with `clan`, `g++` or `c++` it is highly recommended to not use them.
The `gcc` compiler used for the test and development is `gcc version 11.4.0`.

1. Install dependencies:

- Install build essentials.

```sh
sudo apt install build-essential
```

- Install openssl library.

```sh
sudo apt install openssl
```

 - Install openssl development library.
 
```sh
sudo apt install libssl-dev
```

- Install autoconf

```sh
sudo apt-get install autoconf
```

- Install libtool

```sh
sudo apt install libtool
```

#### Tests

In `c/` folder contains protocol implementation for client node written in C. 
All below commands shall be run from `c/` folder in the terminal.
- To test the implementation run `make test`.
- To make memory leak checks and tests run `make memcheck`
- Remember to cleanup the test with `make clean`.

#### Build

UNDER CONSTRUCTION

## Production

### K8s

### Bare metal

UNDER CONSTRUCTION

## Releasing new version

The version is using [Semantic Versioning 2.0.0](https://semver.org/)

Update version in `src/versioning/versioning.go`

## Coding Philosophy

👀 The Zen of Computantis:
 - Simple is better than complex.
 - Simplicity is a prerequisite for reliability.
 - Controlling complexity is the essence of computer programming.
 - Explicit is better than implicit.
 - Errors should never pass silently.
 - Flat is better than nested. Return early rather than nesting deeply.
 - APIs should be easy to use and hard to misuse.
 - Readability Counts.
 - Use composition over inheritance.
 - Avoid package level state.
 - Moderation is a virtue. Use with moderation: go routines, channels, atomic types, generics, interfaces, 'any' type and pointers.

💻 Useful resources:

- https://go-proverbs.github.io/
- https://ntrs.nasa.gov/api/citations/19950022400/downloads/19950022400.pdf
- https://medium.com/eureka-engineering/understanding-allocations-in-go-stack-heap-memory-9a2631b5035d
- https://www.ardanlabs.com/blog/2023/07/getting-friendly-with-cpu-caches.html
- https://eli.thegreenplace.net/2023/common-pitfalls-in-go-benchmarking/


### Addressing the complexity of today software design

- Today companies have more micro-services than clients. So let's have something that can run as mono-service and make shit done.
- Today tools have more features than clients. So let's make very few tools in this mono-repo that offer basic features.
