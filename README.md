# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql)

![Computantis Logo](https://github.com/bartossh/computantis/blob/main/artefacts/logo.png)



## Project overview

The Computantis is a backbone service for creating secure, reliable and performant solution for transaction exchange and Byzantine fault tolerant systems.

### Services and tools

The Computantis contains of:
 - Notary Node - is the main service in the Computantis system, offers secure transfer of transactions within the system, securing data in the DAG structure
 - Webhooks Node - is supporting node that offers webhooks for clients that want to listen for incoming transactions.
 - Client Node - is wallet middleware that abstracts away whole security complexity of request sending, singing transactions and validating transactions and can run alongside client application on the same machine or in proximity of client application.
 - Wallet Tool - is a tool to manage wallet, create the wallet and save it to different formats.
 - Generator Tool - is a helper for creating data for the emulator.
 - Emulator Tool - is a helper for development and testing that pretends to be the client application issuer and receiver sending data via the Computantis system.

### The Computantis system schema

![Computantis Diagram](https://github.com/bartossh/computantis/blob/main/artefacts/Computantis-diagram.png)

#### The computantis system network.

 - Nodes pursue to discover full network and connect to all available nodes.
 - If connection is impossible it is not a problem as gossip about gossip protocol allows to transmit message between not connected nodes via other nodes.
 - Blue node named A is an message issuer. This node is commanding all other receiver nodes to perform action based on the message. 
 - All the green nodes are good actors in the network, while red nodes are bad actors.
 - When transferring the message bad actor receiving the message cannot corrupt the message as the message is cryptographically secured via ED25519 asymmetric signature.
 - Bad actor is required to send the message intact, if he refuses to all the network nodes that knows the node will constantly try to send the message to the bad actor due to the gossip about gossip protocol not seeing the node on the message list of acknowledged nodes.
 - The message will rich all the nodes until there is more connections between nodes than bad actors.
 - Message cannot be replied later as each message is in secured by the unique transaction.
 - Message will be resent to other nodes only if receiving nodes acknowledges the message as cryptographically valid.

### The transaction

 - Transaction is secured by the asymmetric key cryptography signature, it can use different protocols but for now uses ED25519 asymmetric key signature.
 - The data are the core entity and all the other fields in the message are just for the data security.
 - The data are of no significant value for the Computantis system, can be encrypted or decrypted, it is not the case of the Computantis system to validate or use them. 
- The Computantis is responsible for securing data from corruption, saving them in immutable distributed storage and ensuring data to reach the receiver. 

### The secure storage - DAG

 - DAG stands for Direct Acyclic Graph and is post blockchain technology allowing for faster operations on transactions, near zero transaction confirmation time and more democratic data distribution. Very important factor is that securing the DAG is much less computational expensive then the blockchain forging process and may use cheaper and less complicated hardware.
 - On the diagram all the green boxes are the vertices that contains and cryptographically secure the transactions.
 - The red box is a leaf. Leaf when added to the graph needs to validate two of the existing leafs or a leaf and vertex with the highest weight.
 - The leaf when added to the graph is retransmitted to all the nodes in the network, so each of them is able to validate that leaf and transaction against their own DAGs. Because edges which specifies one direction connection between the vertices are based on hashes of vertices all the good players will keep the same vertices and edges in the DAG.
 - The vertex is unique in the DAG and inner transaction is unique per DAG. Sending the same transaction to many nodes will end up in collision and only the first one, received by most of the nodes will be accepted and retransmitted, all the rest of redundant transactions will be rejected.
 - Leaf isn't proving transaction validity until it becomes a vertex by having an edge created from other leaf or vertex.

### Add-ons

Add-ons are the way to interact with Computantis immune system. The immune system shall return an error when data are not meeting the validity criteria.

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

## Coding Philosophy

👀 The Zen of Computantis:
 - Simple is better than complex.
 - Simplicity is prerequisite for reliability.
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

