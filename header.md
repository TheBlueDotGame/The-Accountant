# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql)
[![pages-build-deployment](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment)


The Computantis is a set of services that keeps track of transactions between wallets.
Transactions are not transferring any tokens between wallets but it might be the case if someone wants to use it this way. Just this set of services isn't designed to track token exchange. Instead, transactions are entities holding data that the transaction issuer and transaction receiver agreed upon. Each wallet has its own independent history of transactions. There is a set of strict rules allowing for transactions to happen:

The central server is private to the corporation, government or agency. It is trusted by the above entity and participants. This solution isn't proposing the distributed system of transactions as this is not the case. It is ensuring that the transaction is issued and signed and received and signed to confirm its validity. Blockchain keeps transactions history immutable so the validators can be sure that no one will corrupt the transactions. 
Transaction to be valid needs to:
- Have a valid issuer signature.
- Have a valid receiver signature.
- Have a valid data digest.
- Have a valid issuer public address.
- Have a valid receiver public address.
- Be part of a blockchain.

The full cycle of the transaction and block forging happens as follows:
1. The Issuer creates the transaction and signs it with the issuer's private key, attaching the issuer's public key to the transaction.
2. The Issuer sends the transaction to the central server.
3. The central server validates The Issuer address, signature, data digest and expiration date. If the transaction is valid then it is kept in awaiting transactions repository for the receiver to sign.
4. Receiver asks for awaiting transactions for the receiver's signature in the process of proving his public address and signing data received from the server by the receiver's private key.
5. If the signature is valid then all awaiting transactions are transferred to the receiver.
6. The receiver can sign transactions that the receiver agrees on, and sends them back to the central server. 
7. The central server validates the address, signature, data digest and expiration date then appends the transaction to be added to the next forged block. The transaction is moved from the awaiting transactions repository to the temporary repository (just in case of any unexpected crashes or attacks on the central server).
8. The central servers follow sets of rules from the configuration `yaml` file, which describes how often the block is forged, how many transactions it can hold and what is the difficulty for hashing the block.
9. When the block is forged and added to the blockchain then transactions that are part of the block are moved from the temporary transactions repository to the permanent transactions repository.
10. Information about the new blocks is sent to all validators. Validators cannot reject blocks or rewrite the blockchain. The validator serves the purpose of tracking the central node blockchain to ensure data are not corrupted, the central node isn’t hacked, stores the blocks in its own repository, and serves as an information node for the external clients. If the blockchain is corrupted then the validator raises an alert when noticing corrupted data.
It is good practice to have many validator nodes held by independent entities.

## Microservices

<img src="https://github.com/bartossh/Computantis/blob/main/artefacts/services_diagram.svg">

The Computantis is a set of services that run together to create a fully working solution.
There are 3 services that create the solution but only the central node is required.
- The central node.
    The central node is a REST API and Websocket Server. 
    The central node serves as the main hub of all the computantis actions. 
    The central node stores, validates and approves transactions. 
    Blocks are forged and added to the blockchain by that node. 
    The central node can be scaled and is not competing with other central nodes to forge a block but rather cooperating to make all the work performant.
    The central node is sending new coming transactions and blocks information to validators.
- The validator node.
    The validator is a REST API and Websocket server.
    The validator node is not making any action to validate transactions or forge blocks. It connects to the Websocket of all the central nodes when it's listen for blocks and transactions.
    The responsibility of the validator node is mainly to validate that the blockchain is not corrupted.
    The validator node makes some heavy lifting to inform all the participants about transactions addressed to them. It is done by providing Webhooks for the clients.
    The validator auto reconnects to all new created central nodes.
    The validator is not required to run for the system to function and when detecting corruption on the block chain can only make an alert about that situation.
- The wallet client.
    The wallet client is the REST API that loads encrypted wallet from the file and makes all the computations, validations and signing required by the central node REST API.
    The wallet exposes simplified API that can be used by trusted entities (for example on localhost only) that shares given wallet and wants to access simplified API without need of implementing the wallet logic.
    The wallet client API shouldn't be exposed to the public.
    The wallet client isn't necessary for the computantis solution to function but the wallet should be then implemented by the participants of the system.
- The emulator.
    The emulator contains of two running modes: publisher and subscriber.
    The publisher is the one that is creating the transaction and sending them through the wallet client.
    The subscriber is subscribing to the validator by creating the Webhook and is listening on upcoming transactions info. Then subscriber is pulling all the unsigned transactions and based on given conditions accepts them or rejects them.
    The communication of the emulator is done through the wallet client API. Emulator is not responsible for signing and validating the transactions.
    The signing, validating of the transactions and proving identity is delegated from emulator to the wallet client API.
    The emulator is a testing tool not a production service.

## Setup

All services are using the setup file in a YAML format:
- The central node:
```yaml
bookkeeper:
  difficulty: 1 # The mathematical complexity required to find the hash of the next block in the block chain. Higher the number more complex the problem.
  block_write_timestamp: 300 # Time difference [ s ] between two clocks being forged when block transactions size isn't reached.
  block_transactions_size: 1000 # Max transactions in block. When limit is reached and time difference isn't then new block is forged.
server:
  port: 8080 # Port on which the central node REST API is exposed.
  data_size_bytes: 15000000 # Size of bytes allowed in a single transaction, all above that will be rejected.
  websocket_address: "ws://localhost:8080/ws" # This is the external address of the central node needed to register for other central nodes to use to inform validators.
database:
  conn_str: "postgres://computantis:computantis@localhost:5432" # Database connection string. For now only PostgreSQL is supported.
  database_name: "computantis" # Database name to store all the computantis related data.
  is_ssl: false # Set to true if database requires SSL or to false otherwise, On production SSL true is a must. 
dataprovider:
  longevity: 300 # Data provider provides the data to be signed by the wallet holder in order to verify the wallet public key. This is a time [ s ] describing how long data are valid.
```

- The validator node:
```yaml
validator:
  central_node_address: "http://localhost:8080" # Address of the central node to get discovery information from.
  port: 9090 # Port on which the validator REST API is exposed.
  token: "jykkeD6Tr6xikkYwC805kVoFThm8VGEHStTFk1lIU6RgEf7p3vjFpPQFI3VP9SYeARjYh2jecMSYsmgddjZZcy32iySHijJQ" # Token required by the validator to connect to all the central nodes.
```

- The wallet client:
```yaml
file_operator:
  wallet_path: "test_wallet" # File path where wallet is stored.
  wallet_passwd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d" # Key needed to decrypt the password.
client:
  port: 8095 # Port on which the wallet API is exposed.
  central_node_url: "http://localhost:8080" # Root URL address of a central node or the proxy.
  validator_node_url: "http://localhost:9090" # Root URL of specific validator node to create a Webhook with.
```

- The emulator:
```yaml
emulator:
  timeout_seconds: 5 # Message timeout [ s ]
  tick_seconds: 5 # Tick between publishing the message [ s ]
  random: false # Is the message queue random or consecutive.
  client_url: "http://localhost:8095" # The wallet client root URL.
  port: "8060" # Port on which the emulator API is exposed. This is related to the public URL port.
  public_url: "http://localhost:8060" # Public root URL of the emulator to create the validator Webhook with.
```

## Emulate transaction process starting all basic services
1. To emulate the transaction process first start database using docker-compose.yaml file:


```sh
docker compose up -d
```

This will start database creating schema and populating database with address referring to the `test_wallet`.
Test wallet is encrypted with key stored in `setup_example.yaml` `wallet_passwd` field.

2. When your database is created and running ( should be reachable on `postgres://computantis:computantis@localhost:5432` ) then build and run docker image:


```sh
docker build -t emulation .

docker run --network=host emulation
```

## Run services one by one

1. To run services one by one go compiler is required:
 - Install on Darwin `brew install go`.
 - Install on Linux `apt install go`.

2. Start database:

```sh
docker compose up -d
```

3. Compile binaries:

```sh
make build-local

```

3. Run services:
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

Your system 

## Stress test

Directory `stress/` contains central node REST API performance tests.
Bottleneck is on I/O calls, mostly database writes.
Single PostgreSQL database instance run in docker 1CPU and 2GB RAM allows for 
full cycle processing of 750 transactions per second. This is rough estimate and 
I would soon provide more precise benchmarks.

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
