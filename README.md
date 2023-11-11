# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql)
[![pages-build-deployment](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment)
## Computantis DAG cryptographic protocol.

1. The network.

- The protocol works within the application layer in the OSI network model.
- The protocol wraps data within the transaction.
- The transaction seals the data cryptographically.
- The transaction data are irrelevant to the protocol, and so is its encoding. Encoding is the responsibility of the final application.
- The node participates in the transmission process of the transaction.
- The node acts as a middleware service and ensures transaction legitimacy.
- The transaction receiver and the transaction issuer are known as the client.
- The clients are not aware of each other network URLs, they participate in the transaction transmission using the central node (network of central nodes).
- The client URL is known only for the computantis nodes in the network.
- The URL of the client may change while data are transmitted and it is not affecting the transmission consistency.
- The client is recognized in the network by the public address of cryptographic key pairs
- The client may listen on a webhook for approved transactions on dedicated node that has supporting functionalities.
- The client node is working on the client machine or as an edge device proxying traffic to the device or a client application.
- The traffic cannot omit the computantis network when transaction is transmitted from client to client.
- The client is additionally validating the message's legitimacy, decrypting and decoding the message.
- The central nodes stores all the transactions in the immutable repository in the form of a DAG with Hashed edges and signed vertices.
- The central nodes are concurrently cooperating in creating and validating the DAG vertices and edges.
- The value transfer in form of the 'spice' token may occur and is a subject of validation. The double spending and sufficient amount of tokens is calculated and validated. This process is described in the DAG protocol section.
- When a vertex is created on the node, it is shared with other nodes participating in the network by using gossip about gossip protocol.
- Nodes sends the message containing vertex and list of nodes that knows about the vertex to all nodes that are known to the node, adding itself to the list of nodes.
- Each node receiving the message continues to share the message with all the nodes that are known for the receiving node that are not on the list, but first node adds itself to the list. Process continues until message list contains all the nodes in the network.
- This form of sharing the information creates a redundancy in the traffic, but prevents the bad actors from rejecting legitimate vertices by stopping the message to spread. This as well allows the message to reach nodes that are not connected to the message source node.

2. The transaction.

- Transaction is a central entity of the protocol. 
- The transaction consists of the minimal information required to seal the protocol:
    - ID: Unique repository key / or hash, it is not transmitted over the network.
    - CreatedAt: Timestamp when the transaction was created.
    - IssuerAddress: Public address of the client issuing the transaction.
    - ReceiverAddress: Public address of the client that is the receiver of the transaction.
    - Subject: The string representing the data encoding, type or else, known for the client. The receiver and the issuer node can agree on any enumeration of that variable. For example, if the receiver and the issuer are sending data in many different formats they may indicate it in the subject.
    - Data: Data that are sealed by cryptographic signatures of the receiver and the issuer and encoded with private keys if necessary.
    - IssuerSignature: The cryptographic signature of the issuer.
    - ReceiverSignature: The cryptographic signature of the receiver.
    - Hash: Message hash acting as the control sum of the transaction.
    - Spice: The amount of tokens in the transactions. Spice is transferred always from the Issuer to the Receiver.
- The transaction footprint on the transmitted data size depends on the relation between the size of the ‘Data’ field in the transaction. That is highly recommended to transmit as much data in a single request as possible. 
- The transaction has an upper limit on the size of transmitted data, that is set according to the requirements.
- The transaction is validated on any mutation by the central node:
    - If it is a new transaction, the issuer's signature and hash are checked.
    - If it is signed by the receiver, the issuer signature, receiver signature and hash are checked.
- The client node validates the transaction before it transmits to the application:
    - The issuer address is checked to ensure messages from the given address can be used.
    - The issuer signature and hash are validated.
    - The message is encoded using a private key if necessary.
- Transaction have three possible types.
    - Only the 'spice' transfer, when no 'Data' field is populated and 'spice' has a value. This requires to be only signed by the issuer to be validated and added to DAG.
    - Only contract agreement, when no 'spice' is transferred but 'Data' field is populated. This requires to be signed by the issuer and receiver in order to be added to DAG.
    - Both, the 'spice' transfer and the contract agreement, when the 'spice' has a value and the 'Data' field is populated, This requires to be signed by the issuer and receiver in order to be added to DAG.

3. The DAG.

- DAG stands for Directed Acyclic Graph.
- DAG contains leafs, vertices and edges.
- Leafs are like vertices but are not confirmed yet, so they have no children connected by the edges.
- Vertices are connected by the edges and have children which proofs them being valid.
- Edges are connecting vertices and leafs. Edges are single direction connection, from the parent vertex to the child vertex or a leaf.
- Vertex contains the transaction and seals it by the signature of the node that validated the transaction and digest of all the data vertex contains, Vertex is identified in the graph by its digest.
- Connection between vertices in the graph is achieved by the reference to the vertex digest - hash. 
- If two edges are connecting two leafs with two parent vertices, those parent vertices are assumed to be valid and to contain valid transaction. Valid transaction is checked for sufficient spice and is not having a double spending transaction.  
- DAG seals the transactions immutability and allows for the accounting of the spice transfer.
- DAG is truncated and all the edges and vertices are stored in permanent storage.
- When truncated all the transactions are accounted and the next vertex is created and signed by the node with all the leafs being referred in the edge between new leaf vertex and leafs from the truncated DAG.
- The leaf validation may happen in any node, not only the one that created the leaf.
- Creating a leaf means to create a new leaf with the transaction embedded in to the leaf and then the leaf is gossiped to other nodes in the node network.
- Adding a leaf means it was created by other node and shared with the nodes network in gossip protocol.
- Adding and creating a leaf in to the DAG per node is done in a consecutive way to allow for transaction validation consistency.
- Adding and creating a leaf in to the DAG in the network of nodes is done in the concurrent way. This allows for application scaling, more nodes may compute more transactions.

4. Wallet 

- Wallet is the central entity allowing for sealing data with signatures.
- Wallet holds a pair of asymmetric cryptographic keys. In this case we are implementing asymmetric cryptography based on 256 bits ed25519 elliptic curve algorithm. 
- Wallet public address is encoded in to the transaction as well shared over network as a base58 encoded string. (Bitcoin standard).
- Wallet has capability to create data digest, and sign that digest cryptographically.
- Wallet has capabilities to validate signatures.

## Setup

Use `setup_example.yaml` file as an example how to configure setup. The file is ready to be used with docker-compose command. 

## Start locally services

1. Required minimal node services setup:
 - Computantis node
 - Exporter node
 - Prometheus node
 - Zincsearch node

The computantis network shall contains multiple nodes. More nodes in the network more secure and democratic the network becomes. 

2. Additional services:
 - Nats node
 - Web-Hook node

3. Edge device service:
 - Client node

Install protobuf generator: 
```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

Generate protobuf files with:
```sh
protoc --proto_path=protobuf --go_out=protobufcompiled --go_opt=paths=source_relative block.proto addresses.proto
```


Run in terminal to run services in separate docker containers:

- all services
```sh
make docker-all
```

- dependencies only
```sh
make docker-dependencies
```

## Run services one by one

```sh
make build-local
```

 - Computnatis Node:
   
   ```sh
    ./bin/dedicated/node -c setup_example.yaml

   ``` 
 - Computantis Web-Hooks:

   ```sh
    ./bin/dedicated/webhooks -c setup_example.yaml

   ``` 

 - Client:

   ```sh
    ./bin/dedicated/client -c setup_example.yaml

   ``` 

 - Emulator subscriber:

   ```sh
    ./bin/dedicated/emulator -c setup_example.yaml -d minmax.json subscriber

   ``` 

 - Emulator publisher:

   ```sh
    ./bin/dedicated/emulator -c setup_example.yaml -d data.json publisher

   ``` 

This will compile all the components when docker image is run. All the processes are running in the single docker container.

## Run emulation demo:

1. There is a possibility to run the example demo that will emulate subscriber and publisher:
- Publisher is publishing the messages from `data.json` and it is possible to alter the data, but the structure and format and data types shall be preserved.
- Subscriber will subscribe ans validate transmitted transactions and data in the transaction based on `minmax.json` file, it is possible to alter the data but the structure and format and data types shall be preserved.
2. The configuration for each service is in the `setup_example.yaml` and only one parameter needs to be adjusted. 
- Check your machine IP address in the local network `ifconfig` on Linux.
- Set this parameter in `setup_example.yaml`: `public_url: "http://<your.local.ip.address>:8060"`

3. Run the demo.
- CAUTION: THIS NEEDS TO BE RUN IN DEVELOPMENT ENVIRONMENT AND ALL THE DATA ON YOUR LOCAL COMPUTANTIS ENVIRONMENT WILL BE ALTERED.
- There will be three steps:
    - Run `make docker-all`.
    - Run `go run cmd/emulator/main.go -c setup_example.yaml -d data.json p`
    - Run `go run cmd/emulator/main.go -c setup_example.yaml -d minmax.json s`
- Enjoy.

## Stress test

Directory `stress/` contains central node REST API performance tests.
Bottleneck is on I/O calls, mostly database writes.
Single PostgreSQL database instance run in docker 1CPU and 2GB RAM allows for 
full cycle processing of 750 transactions per second. This is rough estimate and 
I would soon provide more precise benchmarks.

## Vulnerability scanning.

Install govulncheck to perform vulnerability scanning  `go install golang.org/x/vuln/cmd/govulncheck@latest`.

## C - implementation

### Development

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

### Production

UNDER CONSTRUCTION

## GO Packages Documentation:

1. Install gomarkdoc to generate documentation: `go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest`.

Documentation is generated using `header.md` file and the code documentation, then saved in the `README.md`.
Do not modify `README.md` file, all the changes will be overwritten. 

## Package provides webassembly package that expose client API to the front-end applications.

To use client API allowing for creating a wallet and communication with Central Server REST API
copy `wasm/bin/wallet.wasm` and `wasm/js/wasm_exec.js` to you fronted project and execute as in example below.

```html
<html>  
    <head>
        <meta charset="utf-8"/>
        <script src="wasm_exec.js"></script>
        <script>
            const go = new Go();
            WebAssembly.instantiateStreaming(fetch("wallet.wasm"), go.importObject).then((result) => {
                go.run(result.instance);
            });
        </script>
    </head>
    <body></body>
</html>  
```

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

