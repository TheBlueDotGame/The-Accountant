# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql)
[![pages-build-deployment](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment)
## Computantis protocol, DAG and gossiping.

### High level description and purpose.

Secure and performant transaction broker hosting data in the private, redundant and immutable repository. The system is guarded with observatory helpers nodes, that are independently controlling all the notary nodes.
It offers these key features:
 - Transaction immutability and uniqueness (no replay attack possible).
 - Transaction privacy outside the system.
 - Transaction correctness. Cryptographic anti-corruption security of the transaction.
 - Privacy. Data are owned by the private system.
 - Speed - fast and reliable transaction throughput.
 - Integrity - Only allowed wallets are able to participate in the transactions. Transaction anti-forgery is secured with the highest cryptographic standards.
 - Redundancy - Helper nodes are able to independently store data.
 - Lightness - The client node can be deployed on a Raspberry Pi Zero type of device with minimal RAM and CPU footprint (20MB RAM, low CPU usage)
- Scalability - service scales horizontally and vertically. 
- Immutability - Transactions are preserved in the blockchain repository that secures immutability. 
- Maintainability - System shutdown or node failure has no effect over transaction integrity or transaction loss.

### Overview

1. Protocol overview.

- The protocol works within the application layer in the OSI network model.
- The protocol wraps data within the transaction.
- The transaction seals the data cryptographically.
- The transaction data are irrelevant to the protocol, and so is its encoding. Encoding is the responsibility of the final application.
- The central node participates in the transmission process of the transaction.
- The central node acts as a middleware service and ensures transaction legitimacy.
- The transaction receiver and the transaction issuer are known as the client.
- The clients are not aware of each other network URLs, they participate in the transaction transmission using the central node (network of central nodes).
- The client URL is known only for the computantis nodes in the network.
- The URL of the client may change while data are transmitted and it is not affecting the transmission consistency.
- The client is recognized in the network by the public address of cryptographic key pairs
- The client may listen on a webhook for approved transactions.
- The client node is working on the client machine or as an edge device proxying traffic to the device or a client application.
- The traffic cannot omit the computantis network when transaction is transmitted from client to client.
- The client is additionally validating the message's legitimacy, decrypting and decoding the message.
- The central nodes stores all the transactions in the immutable repository in the form of a DAG with Hashed edges and signed vertices.
- The central nodes are concurrently cooperating in creating and validating the DAG.
- The value transfer in form of a token may occur and is a subject of validation by the node taking the vertex as the ancestor. The double spending and sufficient amount of tokens is calculated and validated. The name of the token in computantis network is: 'spice'.
- DAG is truncated every some vertexes to allow for fast graph traversal for 'spice' accounting.

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

- Transfer of the spice occurs only when both issuer and receiver signed the transaction.

3. The DAG.

- DAG stands for Directed Acyclic Graph.
- DAG contains leafs, vertices and edges.
- Leafs are like vertices but are not confirmed yet so they have no children connected by the edges.
- Vertices are connected by the edges and have children which proofs them being valid.
- Edges are connecting vertices and leafs. Edges are single direction connection, from the parent vertex to the child vertex or a leaf.
- Vertex contains the transaction and seals it by the signature of the node that validated the transaction. If two edges are connecting two leafs with two parent vertices, those parent vertices are assumed to be valid and to contain valid transaction. Valid transaction is checked for sufficient spice and is not having a double spending transaction.  
- DAG seals the transactions immutability and allows for the accounting of the spice transfer.
- DAG is truncated and all the edges and vertices are stored in permanent storage. 
- When truncated all the transactions are accounted and the next vertex is created and signed by the node with all the leafs being referred in the edge between new leaf vertex and leafs from the truncated DAG.
- The leaf validation may happen in any node, not only the one that created the leaf.
- Creating a leaf means to create a new leaf with the transaction embedded in to the leaf and then the leaf is gossiped to other nodes in the node network.
- Adding a leaf means it was created by other node and shared with the nodes network in gossip protocol.
- Adding and creating a leaf in to the DAG per node is done in a consecutive way to allow for transaction validation consistency.
- Adding and creating a leaf in to the DAG in the network of nodes is done in the concurrent way. This allows for application scaling, more nodes may compute more transactions.

4. The networking

- The network of nodes communicate via gossip about gossip protocol.
- Gossip about gossip protocol is used in many crypto DAG protocols but each of them may use slightly different form.
- Some gossip about gossip protocols are patented.
- Computantis uses its own implementation of the gossip about gossip protocol that is not based on the gossip about gossip patents.


5. Wallet 

- Wallet is the central entity allowing for sealing data with signatures.
- Wallet holds a pair of asymmetric cryptographic keys. In this case we are implementing asymmetric cryptography based on 256 bits ed25519 elliptic curve algorithm. 
- Wallet public address is encoded in to the transaction as well shared over network as a base58 encoded string. (Bitcoin standard).
- Wallet has capability to create data digest, and sign that digest cryptographically.
- Wallet has capabilities to validate signatures.

## Setup

All services are using the setup file in a YAML format:
- The notary node:
```yaml
is_profiling: false
bookkeeper:
  difficulty: 1
  block_write_timestamp: 300
  block_transactions_size: 1000
notary_server:
  node_public_url: notary-node:8000
  port: 8000
  data_size_bytes: 15000
local_cache:
  max_len: 5000
nats:
  server_address: "nats://nats:4222"
  client_name: "notary-1"
  token: "D9pHfuiEQPXtqPqPdyxozi8kU2FlHqC0FlSRIzpwDI0="
storage_config:
  transaction_database:
    conn_str: "postgres://computantis:computantis@postgres:5432"
    database_name: "computantis"
    is_ssl: false
  blockchain_database:
    conn_str: "postgres://computantis:computantis@postgres:5432"
    database_name: "computantis"
    is_ssl: false
  node_register_database:
    conn_str: "postgres://computantis:computantis@postgres:5432"
    database_name: "computantis"
    is_ssl: false
  address_database:
    conn_str: "postgres://computantis:computantis@postgres:5432"
    database_name: "computantis"
    is_ssl: false
  token_database:
    conn_str: "postgres://computantis:computantis@postgres:5432"
    database_name: "computantis"
    is_ssl: false
dataprovider:
  longevity: 300
zinc_logger:
  address: http://zincsearch:4080 
  index: notary-1
  token: Basic YWRtaW46emluY3NlYXJjaA==
```

- The helper node specific:
```yaml
is_profiling: false
helper_server:
  port: 8000
nats:
  server_address: "nats://nats:4222"
  client_name: "notary-1"
  token: "D9pHfuiEQPXtqPqPdyxozi8kU2FlHqC0FlSRIzpwDI0="
storage_config:
  helper_status_database:
    conn_str: "postgres://computantis:computantis@postgres:5432"
    database_name: "computantis"
    is_ssl: false
zinc_logger:
  address: http://zincsearch:4080 
  index: helper-1
  token: Basic YWRtaW46emluY3NlYXJjaA==
```

- The client node:
```yaml
file_operator: # file operator allows to read wallet in gob and pem format from the file
  wallet_path: "test_wallet"
  wallet_passwd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d"
  pem_path: "ed25519"
notary:
  port: 8095
  central_node_url: "http://localhost:8080" 
  validator_node_url: "http://localhost:9090" 
zinc_logger:  
  address: http://zincsearch:4080 
  index: wallet-1 
  token: Basic YWRtaW46emluY3NlYXJjaA== 
```

- The emulator:
```yaml
emulator: # emulates data 
  timeout_seconds: 20
  tick_seconds: 1
  random: false
  client_url: "http://client-node:8000" # client node middleware URL
  port: "8060"
  public_url: "http://subscriber-node:8060" # If running emulators locally, best to set it up as your local network machine IP.
```

## Start locally all services

Required services setup:
 - PostgreSQL database
 - Central node
 - Validator node
 - Client node
 - Exporter node
 - Prometheus node
 - Zincsearch node
 - Nats node


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

 - Central:
   
   ```sh
    ./bin/dedicated/central -c setup_example.yaml

   ``` 
 - Validator:

   ```sh
    ./bin/dedicated/validator -c setup_example.yaml

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

### Demo resource usage

- System parameter
```sh
OS: Ubuntu 20.04 focal
Kernel: x86_64 Linux 5.15.0-76-generic
CPU: AMD Ryzen 7 PRO 4750U with Radeon Graphics @ 16x 1,7GHz
GPU: Advanced Micro Devices, Inc. [AMD/ATI] Renoir (rev d1)
RAM: 31451MiB
SERVICES: Running in Docker
```

- Stats:
```sh
CONTAINER ID   NAME                    CPU %     MEM USAGE
294fe037553d   client-node             0.41%     13.99MiB 
892c7a00df55   prometheus              0.00%     42.24MiB 
046e5abc3f90   node-exporter           0.00%     10.48MiB 
b898d27d9ebb   validator-node          0.31%     15.73MiB 
cf65e697b277   central-node            1.17%     12.95MiB 
793fe32f060c   postgres                0.70%     38.15MiB 
d075fab56e0e   computantis-grafana-1   0.06%     86.7MiB 
b49f6921f75b   zincsearch              1.13%     50.54MiB 
```

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

👀 Development guidelines:

- Correctness is the first principle.
- Performance counts.
- Performance applies equally to computational performance and development performance.
- Write code that performs well and benchmark it.
- Don't microbenchmark, do the benchmarking in the context.
- Unit test your code, especially critical parts.
- Write integration tests for the API calls or use integration testing tools such as Postman.
- Programming Language counts. Pick the effective, performant, safe and simple one.
- Be open-minded, do not fall into the pitfalls of one ideology, non solve all the problems.
- Less is almost always more.
- Abstraction is your superpower. Unnecessary abstraction and complicated abstraction are your kryptonite.
- Avoid the inheritance it is the root of all evil. But sometimes we pick the inheritance as the lesser evil.
- Use composition. Please keep it simple.
- Problems are complex do not make them more complicated than they are.
- Write documentation, don't write comments (comments lie, code never lies).
- Never panic, handle errors gracefully.
- Focus on data first, avoid pointers if possible, and paginate structures.
- Prealocate continuous memory if possible. Keep things on the stack if possible.
- Have the courage to change your opinion.
- Don't be clever be boring.

💻 Useful resources:

- https://go-proverbs.github.io/
- https://ntrs.nasa.gov/api/citations/19950022400/downloads/19950022400.pdf
- https://medium.com/eureka-engineering/understanding-allocations-in-go-stack-heap-memory-9a2631b5035d
- https://www.ardanlabs.com/blog/2023/07/getting-friendly-with-cpu-caches.html
- https://eli.thegreenplace.net/2023/common-pitfalls-in-go-benchmarking/

