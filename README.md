# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql) [![Go](https://github.com/bartossh/Computantis/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/bartossh/Computantis/actions/workflows/go.yml)

![Computantis Logo](https://github.com/bartossh/computantis/blob/main/artefacts/logo.png)

## Project overview

The Computantis is a backbone service for creating secure, reliable and performant solutions for transaction exchange and Byzantine fault-tolerant systems.

### Problems it solves

- The main problem Computantis solves is offering cheaper to run and less cumbersome solutions for cryptocurrency and distributed state machines.

- Today distributed state machines and cryptocurrency systems are complicated. Computantis offers the core functionality that can be easily extended.

- Proof of work hashing is computationally expensive and slow, proof of stake restricts participants and requires additional protocols that decide who will forge the block. Computantis is performant and can run without expensive GPUs and super servers, and doesn't require the decision-making protocol which saves on computation and improves security and clarity.

- Gossip about gossip protocol offers better robustness against connection failures and doesn't require a direct connection between nodes to transfer transactions.

- DAG vertex creation and validation is much faster than the blockchain forging process and gives an almost instant response in balance checking.

- DAG can be truncated, it saves response time, graph traversal time and double spending validation process.

- The Computantis is a fun project that can be improved. Now transfers more than 200 transactions per second compared to ETC with less than 20 transactions per second and BTC with less than 10 transactions per second.

- The sufficient requirements to run a node are 2vCPU and 512MB RAM. Disc space depends on how often DAG is truncated and backed up.

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

#### Build and run

##### Build all (node, wallet, webhooks, emulator).

Make sure you have [Go](https://go.dev/) programming language installed.
Run `make build-all` to build all binaries.

##### Building node only

Make sure you have [Go](https://go.dev/) programming language installed.
Run `make build-node` to build only dedicated node binary.

###### The emulator

Internal tool used for test and benchmarks only. Do not use for production.

###### The node

Use dedicated binary from bin/dedicated directory.

Run `./bin/dedicated/node -c <path to your setup.yaml>` to start node.

The `setup.taml` file example:
```yaml
is_profiling: false # Set to true if you want to run PGO profile. To use PGO for binary compilation copy default.pgo to your src root directory.
notary_server: # This section allows you to set up notary server parameters. The notary server is the one to be accessed via wallets. GRPC.
  public_url: localhost:8000 # The notary server public IP that server will use to introduce itself.
  port: 8000 # The port at which server will run.
  data_size_bytes: 15000 # Max data size per transaction in bytes.
gossip_server: # This section allows to set up gossip protocol server. The gossip protocol endpoints are run by this server. GRPC.
  url: "localhost:8080" # The notary server URL that server will use to introduce itself in the gossip network.
  genesis_url: # The genesis node URL from which the server will read all URLs of other nodes interconnected in that gossip network and introduce itself via gossip discovery protocol. When empty it starts the node as the first one in the network waiting for connections.
  load_dag_url: # The URL of the node that will serve the DAG update. Any node from the gossip network, usually the same as genesis_url. If empty then genesis transaction and vertex are created. Requires the wallet for genesis that is used only once and will not receive tokens.
  genesis_receiver: "1HspmQ7wjnKh9qhNdZ94Ta9c3ugsT9XoWJ9CdS32B1kSTBckpZ" # Genesis receiver is used only from the genesis node. It is the wallet that will have all the tokens created during genesis vertex creation. This happens once when creating the genesis transaction and vertex.
  genesis_spice:
    currency: 1000000 # Amount of primus tokens created during genesis.
    supplementary_currency: 0 # Amount of secundus tokens created during genesis.
  port: 8080 # Port on which GRPC server of gossip protocol to run.
accountant: # Accountant section allows to set up DAG accounting details.
  trusted_nodes_db_path: # Path to storage on disc for trusted nodes. When empty stored in RAM. Vertices created by trusted nodes have permission to be added to the DAG without balance accounting.  
  tokens_db_path: # Path to storage of access tokens. When empty stored in RAM.
  trxs_to_vertices_map_db_path: # Path to storage of transaction - vertex relation. When empty stored in RAM. 
  vertices_db_path: # Path to database that vertex will be saved after truncation. When empty stored in RAM.
  truncate_at_weight: 0 # Vertices weight at which truncate the DAG. When zero then default is used. It is recommended to use default. 
nats:
  server_address: # Nats server address. Nats collects information about transactions and vertices and pipes them to webhook nodes. When empty nats will not be used.
  client_name: "notary-genesis" # Name of the Nats client.
  token: "D9pHfuiEQPXtqPqPdyxozi8kU2FlHqC0FlSRIzpwDI0=" # Token to create connection whit nats set in nats.conf.
dataprovider: # Dataprovider is the section to set up how long the unmatching vertices will live to be reused in the future.
  longevity: 300 # Unmatching vertices longevity in seconds.
file_operator: # File operator section allows to provide the path and decoding key (AES HEX) for the wallet.
  wallet_path: "artefacts/wallet_notary_genesis" # Path to wallet.
  wallet_passwd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d" # HEX string to encode wallet. 
  pem_path: "" # if PEM is used provide pem file.
zinc_logger: # Zinc search section allows to connect the node to the zinc search so all logs are send to the zinc search server. 
  address: # Address of zinc search server. When empty logs goes to stdout.  
  index: genesis # Specify the index for logs from currant node. Should be unique between all nodes.
  token: Basic YWRtaW46emluY3NlYXJjaA== # The zinc search token for given index.
```


###### The wallet:


Run `./bin/dedicated/webhooks -h` to get help or follow below description.

```sh
NAME:
   wallet - Wallet CLI tool allows to create a new Wallet or act on the local Wallet by using keys from different formats and transforming them between formats.
            Please use with the best security practices. GOBINARY is safer to move between machines as this file format is encrypted with AES key.
            Tool provides Spice and Contract transfer, reading balance, reading contracts, approving and rejecting contracts.

USAGE:
   wallet [global options] command [command options] [arguments...]

COMMANDS:
   new, n      Creates new wallet and saves it to encrypted GOBINARY file and PEM format.
   topem, tp   Reads GOBINARY and saves it to PEM file format.
   togob, tg   Reads PEM file format and saves it to GOBINARY encrypted file format.
   address, a  Reads wallet public address.
   connect, c  Establish connection with node.
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```

*If you are running node with the same wallet you would like to use for transactions, please do not connect to this node.*
The wallet and node cannot share the same cryptographic key pairs for security reasons. 

###### The webhooks:

Do not required to run the network. Use only if you want to follow transactions or created another service for statistics, reports, ect, that runs as subscribers.

Run `./bin/dedicated/webhooks -c <path to your setup.yaml>` to start node.
```yaml
webhooks_server: # This section specifies the web hook server setup.
  port: 8000 # Port at which to start the server.
```


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

