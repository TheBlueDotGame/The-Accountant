# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql)
[![pages-build-deployment](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment)
## Computantis protocol.

1. General description.

- The protocol works within the application layer in the OSI network model.
- The protocol wraps data within the transaction.
- The transaction seals the data cryptographically.
- The transaction data are irrelevant to the protocol, and so is its encoding. Encoding is the responsibility of the final application.
- The central node participates in the transmission process of the transaction.
- The central node acts as a middleware service and ensures transaction legitimacy.
- The transaction receiver and the transaction issuer are known as the client.
- The clients are not aware of each other network URLs, they participate in the transaction transmission using the central node (network of central nodes).
- The client URL is known only for the network of central nodes and the validators.
- The URL of the client may change while data are transmitted and it is not affecting the transmission consistency.
- The client is recognized in the network by the public address of cryptographic key pairs.
- The client's responsibility is to inform the validator of the URL change. The validators are public nodes that are validating the central node's network legitimacy and inform the client nodes about the awaiting transactions.
- The client node is working on the client machine or as an edge device proxying traffic to the device or a client application.
- The traffic cannot omit the central node when transmitted from client to client.
- The client is additionally validating the message's legitimacy, decrypting and decoding the message.
- The central nodes stores all the transactions in the immutable repository in the form of a blockchain.
- The central nodes are not competing over the forging of the block but cooperate to do so.
- There is no token transfer happening and there is no prize given for forging the block. The whole network of nodes is a privately owned entity without the possibility of branching or corrupting the blockchain. Any attempt to corrupt the system by any node will be recognized as a violation of procedure and such central node will be removed from the network and blacklisted, even thou this is a privately owned central node. Any misbehaving node will be treated as broken or compromised by a third party or a hacker.
- The validators and the central nodes are voting using a weighted system over violation of procedure. There is a minimum of 51% of votes to disconnect the central node and blacklist the node, the same applies to the validators.

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
- The transaction footprint on the transmitted data size depends on the relation between the size of the ‘Data’ field in the transaction. That is highly recommended to transmit as much data in a single request as possible. 
- The transaction has an upper limit on the size of transmitted data, that is set according to the requirements.
- The transaction is validated on any mutation by the central node:
- If it is a new transaction, the issuer's signature and hash are checked.
- If it is signed by the receiver, the issuer signature, receiver signature and hash are checked.
- The client node validates the transaction before it transmits to the application.:
- The issuer address is checked to ensure messages from the given address can be used.
- The issuer signature and hash are validated.
- The message is encoded using a private key if necessary.


3. The blockchain.

- The block consists of:
- ID: Unique repository key / or hash, it is not transmitted over the network.
- TrxHashes: The merkle tree of transaction hashes.
- Hash: All the other fields hash.
- PrevHash: Previous block hash.
- Index: Consecutive number of the block. A unique number describing the block position in the blockchain.
- Timestamp: Time when the block was forged.
- Nonce: 64-bit unsigned integer value that was calculated to create a hash to reach a given difficulty.
- Difficulty: The difficulty of the forging process for looking for the nonce value to calculate block hash. Higher difficulty ensures the immutability of past blocks, there will be harder to rewrite blocks when new ones are created and catch up with the existing chain.
- The blockchain cannot be mutated, which is ensured by the:
- Hashed merkle three of all the transactions are part of the current block.
- The block is hashed.
- The previous block hash is part of the current block and is taken as a part of the data to hash the current block.
- No branching of the blockchain is possible, nonce and difficulty prevent rewriting history by creating a requirement for high computational power to overcome the challenge of outperforming the network of nodes that forges the blocks.

4. The networking

- The network consists of three main participants:
- The central node - validates transactions and forges blocks.
- The validator - validates blocks and informs the client over webhook about awaiting transactions.
- The client - proxy between the application or server and the system. Signs the transactions and takes care of transmitting transactions.
- The central node network:
- The central node network communicates over the inner pub- sub-system.
- The transactions and blockchain repository are shared between all the nodes.
- The repository is sharded and distributed.
- The central node allows for HTTP and Web Socket connection. HTTP is used for interaction with clients where a Web Socket connection is used to communicate with Validators.
- Central nodes are offering nodes discovery protocol over the Web Socket.
- The validators nodes network:
- Validators are connected to each central node in the computantis network.
- Validators are able to discover all the central nodes by using the central node discovery protocol over Web Socket.
- Validators validate the block.
- Validators consist of a webhook endpoint to which a client node can assign its URL and wait for the information about the awaiting transactions.
- Validators will reconnect if the connection is lost.
- Validators will automatically connect to a new central node created in the computantis network if such is started.
- The client node network:
- The client sends its location to the validator node.
- The validator responds with the list of available central nodes to communicate with.
- The client activates the webhook in the validator node each time the URL is changed. This allows the client node to receive information about transactions waiting, being altered or rejected.
- The client node pulls transactions from the known central node.
- The client node sends signed or rejected transactions to the central node.


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
zinc_logger: # Zinc search (elastic-search like service) for convenient access to logs. 
  address: http://zincsearch:4080 # Zinc search address in computantis network.
  index: central # Name of the micro-service for easy logs filtering.
  token: Basic YWRtaW46emluY3NlYXJjaA== # Token allows to validate legitimacy of the service that is sending the log.
```

- The validator node:
```yaml
validator:
  central_node_address: "http://localhost:8080" # Address of the central node to get discovery information from.
  port: 9090 # Port on which the validator REST API is exposed.
  token: "jykkeD6Tr6xikkYwC805kVoFThm8VGEHStTFk1lIU6RgEf7p3vjFpPQFI3VP9SYeARjYh2jecMSYsmgddjZZcy32iySHijJQ" # Token required by the validator to connect to all the central nodes.
zinc_logger: # Zinc search (elastic-search like service) for convenient access to logs. 
  address: http://zincsearch:4080 # Zinc search address in computantis network.
  index: validator # Name of the micro-service for easy logs filtering.
  token: Basic YWRtaW46emluY3NlYXJjaA== # Token allows to validate legitimacy of the service that is sending the log.
```

- The client node:
```yaml
file_operator:
  wallet_path: "test_wallet" # File path where wallet is stored.
  wallet_passwd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d" # Key needed to decrypt the password.
client:
  port: 8095 # Port on which the wallet API is exposed.
  central_node_url: "http://localhost:8080" # Root URL address of a central node or the proxy.
  validator_node_url: "http://localhost:9090" # Root URL of specific validator node to create a Webhook with.
zinc_logger: # Zinc search (elastic-search like service) for convenient access to logs. 
  address: http://zincsearch:4080 # Zinc search address in computantis network.
  index: wallet # Name of the micro-service for easy logs filtering.
  token: Basic YWRtaW46emluY3NlYXJjaA== # Token allows to validate legitimacy of the service that is sending the log.
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

## Start locally all services

Required services setup:
 - PostgreSQL database
 - Central node
 - Validator node
 - Client node
 - Exporter node
 - Prometheus node
 - Zincsearch node

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

## Documentation:

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

<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# address

```go
import "github.com/bartossh/Computantis/address"
```

## Index

- [type Address](<#Address>)


<a name="Address"></a>
## type [Address](<https://github.com/bartossh/Computantis/blob/main/address/address.go#L4-L7>)

Address holds information about unique PublicKey.

```go
type Address struct {
    ID        any    `json:"-"          bson:"_id,omitempty" db:"id"`
    PublicKey string `json:"public_key" bson:"public_key"    db:"public_key"`
}
```

# aeswrapper

```go
import "github.com/bartossh/Computantis/aeswrapper"
```

## Index

- [Variables](<#variables>)
- [type Helper](<#Helper>)
  - [func New\(\) Helper](<#New>)
  - [func \(h Helper\) Decrypt\(key, data \[\]byte\) \(\[\]byte, error\)](<#Helper.Decrypt>)
  - [func \(h Helper\) Encrypt\(key, data \[\]byte\) \(\[\]byte, error\)](<#Helper.Encrypt>)


## Variables

<a name="ErrInvalidKeyLength"></a>

```go
var (
    ErrInvalidKeyLength   = errors.New("invalid key length, must be longer then 32 bytes")
    ErrCipherFailure      = errors.New("cipher creation failure")
    ErrGCMFailure         = errors.New("gcm creation failure")
    ErrRandomNonceFailure = errors.New("random nonce creation failure")
    ErrOpenDataFailure    = errors.New("open data failure, cannot decrypt data")
)
```

<a name="Helper"></a>
## type [Helper](<https://github.com/bartossh/Computantis/blob/main/aeswrapper/aeswrapper.go#L25>)

Helper wraps eas encryption and decryption. Uses Galois Counter Mode \(GCM\) for encryption and decryption.

```go
type Helper struct{}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/aeswrapper/aeswrapper.go#L28>)

```go
func New() Helper
```

Creates a new Helper.

<a name="Helper.Decrypt"></a>
### func \(Helper\) [Decrypt](<https://github.com/bartossh/Computantis/blob/main/aeswrapper/aeswrapper.go#L61>)

```go
func (h Helper) Decrypt(key, data []byte) ([]byte, error)
```

Decrypt decrypts data with key. Key must be at least 32 bytes long.

<a name="Helper.Encrypt"></a>
### func \(Helper\) [Encrypt](<https://github.com/bartossh/Computantis/blob/main/aeswrapper/aeswrapper.go#L34>)

```go
func (h Helper) Encrypt(key, data []byte) ([]byte, error)
```

Encrypt encrypts data with key. Key must be at least 32 bytes long.

# block

```go
import "github.com/bartossh/Computantis/block"
```

## Index

- [type Block](<#Block>)
  - [func New\(difficulty, next uint64, prevHash \[32\]byte, trxHashes \[\]\[32\]byte\) Block](<#New>)
  - [func \(b \*Block\) Validate\(trxHashes \[\]\[32\]byte\) bool](<#Block.Validate>)


<a name="Block"></a>
## type [Block](<https://github.com/bartossh/Computantis/blob/main/block/block.go#L20-L29>)

Block holds block information. Block is a part of a blockchain assuring immutability of the data. Block mining difficulty may change if needed and is a part of a hash digest. Block ensures that transactions hashes are valid and match the transactions stored in the repository.

```go
type Block struct {
    ID         any        `json:"-"          bson:"_id"        db:"id"`
    TrxHashes  [][32]byte `json:"trx_hashes" bson:"trx_hashes" db:"trx_hashes"`
    Hash       [32]byte   `json:"hash"       bson:"hash"       db:"hash"`
    PrevHash   [32]byte   `json:"prev_hash"  bson:"prev_hash"  db:"prev_hash"`
    Index      uint64     `json:"index"      bson:"index"      db:"index"`
    Timestamp  uint64     `json:"timestamp"  bson:"timestamp"  db:"timestamp"`
    Nonce      uint64     `json:"nonce"      bson:"nonce"      db:"nonce"`
    Difficulty uint64     `json:"difficulty" bson:"difficulty" db:"difficulty"`
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/block/block.go#L35>)

```go
func New(difficulty, next uint64, prevHash [32]byte, trxHashes [][32]byte) Block
```

New creates a new Block hashing it with given difficulty. Higher difficulty requires more computations to happen to find possible target hash. Difficulty is stored inside the Block and is a part of a hashed data. Transactions hashes are prehashed before calculating the Block hash with merkle tree.

<a name="Block.Validate"></a>
### func \(\*Block\) [Validate](<https://github.com/bartossh/Computantis/blob/main/block/block.go#L60>)

```go
func (b *Block) Validate(trxHashes [][32]byte) bool
```

Validate validates the Block. Validations goes in the same order like Block hashing algorithm, just the proof of work part is not required as Nonce is already known.

# blockchain

```go
import "github.com/bartossh/Computantis/blockchain"
```

## Index

- [Variables](<#variables>)
- [func GenesisBlock\(ctx context.Context, rw BlockReadWriter\) error](<#GenesisBlock>)
- [type BlockReadWriter](<#BlockReadWriter>)
- [type BlockReader](<#BlockReader>)
- [type BlockWriter](<#BlockWriter>)
- [type Blockchain](<#Blockchain>)
  - [func New\(ctx context.Context, rw BlockReadWriter\) \(\*Blockchain, error\)](<#New>)
  - [func \(c \*Blockchain\) LastBlockHashIndex\(ctx context.Context\) \(\[32\]byte, uint64, error\)](<#Blockchain.LastBlockHashIndex>)
  - [func \(c \*Blockchain\) ReadBlocksFromIndex\(ctx context.Context, idx uint64\) \(\[\]block.Block, error\)](<#Blockchain.ReadBlocksFromIndex>)
  - [func \(c \*Blockchain\) ReadLastNBlocks\(ctx context.Context, n int\) \(\[\]block.Block, error\)](<#Blockchain.ReadLastNBlocks>)
  - [func \(c \*Blockchain\) WriteBlock\(ctx context.Context, block block.Block\) error](<#Blockchain.WriteBlock>)


## Variables

<a name="ErrBlockNotFound"></a>

```go
var (
    ErrBlockNotFound        = errors.New("block not found")
    ErrInvalidBlockPrevHash = errors.New("block prev hash is invalid")
    ErrInvalidBlockHash     = errors.New("block hash is invalid")
    ErrInvalidBlockIndex    = errors.New("block index is invalid")
)
```

<a name="GenesisBlock"></a>
## func [GenesisBlock](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L45>)

```go
func GenesisBlock(ctx context.Context, rw BlockReadWriter) error
```

GenesisBlock creates a genesis block. It is a first block in the blockchain. The genesis block is created only if there is no other block in the repository. Otherwise returning an error.

<a name="BlockReadWriter"></a>
## type [BlockReadWriter](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L30-L33>)

BlockReadWriter provides read and write access to the blockchain repository.

```go
type BlockReadWriter interface {
    BlockReader
    BlockWriter
}
```

<a name="BlockReader"></a>
## type [BlockReader](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L19-L22>)

BlockReader provides read access to the blockchain repository.

```go
type BlockReader interface {
    LastBlock(ctx context.Context) (block.Block, error)
    ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
}
```

<a name="BlockWriter"></a>
## type [BlockWriter](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L25-L27>)

BlockWriter provides write access to the blockchain repository.

```go
type BlockWriter interface {
    WriteBlock(ctx context.Context, block block.Block) error
}
```

<a name="Blockchain"></a>
## type [Blockchain](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L38-L40>)

Blockchain keeps track of the blocks creating immutable chain of data. Blockchain is stored in repository as separate blocks that relates to each other based on the hash of the previous block.

```go
type Blockchain struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L58>)

```go
func New(ctx context.Context, rw BlockReadWriter) (*Blockchain, error)
```

New creates a new Blockchain that has access to the blockchain stored in the repository. The access to the repository is injected via BlockReadWriter interface. You can use any implementation of repository that implements BlockReadWriter interface and ensures unique indexing for Block Hash, PrevHash and Index.

<a name="Blockchain.LastBlockHashIndex"></a>
### func \(\*Blockchain\) [LastBlockHashIndex](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L65>)

```go
func (c *Blockchain) LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error)
```

LastBlockHashIndex returns last block hash and index.

<a name="Blockchain.ReadBlocksFromIndex"></a>
### func \(\*Blockchain\) [ReadBlocksFromIndex](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L96>)

```go
func (c *Blockchain) ReadBlocksFromIndex(ctx context.Context, idx uint64) ([]block.Block, error)
```

ReadBlocksFromIndex reads all blocks from given index till the current block in consecutive order.

<a name="Blockchain.ReadLastNBlocks"></a>
### func \(\*Blockchain\) [ReadLastNBlocks](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L74>)

```go
func (c *Blockchain) ReadLastNBlocks(ctx context.Context, n int) ([]block.Block, error)
```

ReadLastNBlocks reads the last n blocks in reverse consecutive order.

<a name="Blockchain.WriteBlock"></a>
### func \(\*Blockchain\) [WriteBlock](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L122>)

```go
func (c *Blockchain) WriteBlock(ctx context.Context, block block.Block) error
```

WriteBlock writes block in to the blockchain repository.

# bookkeeping

```go
import "github.com/bartossh/Computantis/bookkeeping"
```

## Index

- [Variables](<#variables>)
- [type AddressChecker](<#AddressChecker>)
- [type BlockFindWriter](<#BlockFindWriter>)
- [type BlockReactivePublisher](<#BlockReactivePublisher>)
- [type BlockReadWriter](<#BlockReadWriter>)
- [type BlockReader](<#BlockReader>)
- [type BlockWriter](<#BlockWriter>)
- [type BlockchainLockSubscriber](<#BlockchainLockSubscriber>)
- [type Config](<#Config>)
  - [func \(c Config\) Validate\(\) error](<#Config.Validate>)
- [type DataBaseProvider](<#DataBaseProvider>)
- [type Ledger](<#Ledger>)
  - [func New\(config Config, bc BlockReadWriter, db DataBaseProvider, ac AddressChecker, vr SignatureVerifier, tf BlockFindWriter, log logger.Logger, blcPub BlockReactivePublisher, trxIssuedPub TrxIssuedReactivePunlisher, sub BlockchainLockSubscriber\) \(\*Ledger, error\)](<#New>)
  - [func \(l \*Ledger\) Run\(ctx context.Context\) error](<#Ledger.Run>)
  - [func \(l \*Ledger\) VerifySignature\(message, signature \[\]byte, hash \[32\]byte, address string\) error](<#Ledger.VerifySignature>)
  - [func \(l \*Ledger\) WriteCandidateTransaction\(ctx context.Context, trx \*transaction.Transaction\) error](<#Ledger.WriteCandidateTransaction>)
  - [func \(l \*Ledger\) WriteIssuerSignedTransactionForReceiver\(ctx context.Context, trx \*transaction.Transaction\) error](<#Ledger.WriteIssuerSignedTransactionForReceiver>)
- [type NodeRegister](<#NodeRegister>)
- [type SignatureVerifier](<#SignatureVerifier>)
- [type Synchronizer](<#Synchronizer>)
- [type TrxIssuedReactivePunlisher](<#TrxIssuedReactivePunlisher>)
- [type TrxWriteReadMover](<#TrxWriteReadMover>)


## Variables

<a name="ErrTrxExistsInTheLadger"></a>

```go
var (
    ErrTrxExistsInTheLadger            = errors.New("transaction is already in the ledger")
    ErrTrxExistsInTheBlockchain        = errors.New("transaction is already in the blockchain")
    ErrAddressNotExists                = errors.New("address does not exist in the addresses repository")
    ErrBlockTxsCorrupted               = errors.New("all transaction failed, block corrupted")
    ErrDifficultyNotInRange            = errors.New("invalid difficulty, difficulty can by in range [1 : 255]")
    ErrBlockWriteTimestampNoInRange    = errors.New("block write timestamp is not in range of [one second : four hours]")
    ErrBlockTransactionsSizeNotInRange = errors.New("block transactions size is not in range of [1 : 60000]")
)
```

<a name="ErrSynchronizerWatchFailure"></a>

```go
var (
    ErrSynchronizerWatchFailure   = errors.New("synchronizer failure")
    ErrSynchronizerReleaseFailure = errors.New("synchronizer release failure")
    ErrSynchronizerStopped        = errors.New("synchronizer stopped")
)
```

<a name="AddressChecker"></a>
## type [AddressChecker](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L72-L74>)

AddressChecker provides address existence check method. If you use other repository than addresses repository, you can implement this interface but address should be uniquely indexed in your repository implementation.

```go
type AddressChecker interface {
    CheckAddressExists(ctx context.Context, address string) (bool, error)
}
```

<a name="BlockFindWriter"></a>
## type [BlockFindWriter](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L82-L84>)

BlockFindWriter provides block find and write method.

```go
type BlockFindWriter interface {
    FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
}
```

<a name="BlockReactivePublisher"></a>
## type [BlockReactivePublisher](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L101-L103>)

BlockReactivePublisher provides block publishing method. It uses reactive package. It you are using your own implementation of reactive package take care of Publish method to be non\-blocking.

```go
type BlockReactivePublisher interface {
    Publish(block.Block)
}
```

<a name="BlockReadWriter"></a>
## type [BlockReadWriter](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L64-L67>)

BlockReadWriter provides block read and write methods.

```go
type BlockReadWriter interface {
    BlockReader
    BlockWriter
}
```

<a name="BlockReader"></a>
## type [BlockReader](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L54-L56>)

BlockReader provides block read methods.

```go
type BlockReader interface {
    LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error)
}
```

<a name="BlockWriter"></a>
## type [BlockWriter](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L59-L61>)

BlockWriter provides block write methods.

```go
type BlockWriter interface {
    WriteBlock(ctx context.Context, block block.Block) error
}
```

<a name="BlockchainLockSubscriber"></a>
## type [BlockchainLockSubscriber](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/synch.go#L14-L16>)



```go
type BlockchainLockSubscriber interface {
    SubscribeToLockBlockchainNotification(ctx context.Context, c chan<- bool, node string)
}
```

<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L113-L117>)

Config is a configuration of the Ledger.

```go
type Config struct {
    Difficulty            uint64 `json:"difficulty"              bson:"difficulty"              yaml:"difficulty"`
    BlockWriteTimestamp   uint64 `json:"block_write_timestamp"   bson:"block_write_timestamp"   yaml:"block_write_timestamp"`
    BlockTransactionsSize int    `json:"block_transactions_size" bson:"block_transactions_size" yaml:"block_transactions_size"`
}
```

<a name="Config.Validate"></a>
### func \(Config\) [Validate](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L120>)

```go
func (c Config) Validate() error
```

Validate validates the Ledger configuration.

<a name="DataBaseProvider"></a>
## type [DataBaseProvider](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L92-L96>)

DataBaseProvider abstracts all the methods that are expected from repository.

```go
type DataBaseProvider interface {
    Synchronizer
    TrxWriteReadMover
    NodeRegister
}
```

<a name="Ledger"></a>
## type [Ledger](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L140-L154>)

Ledger is a collection of ledger functionality to perform bookkeeping. It performs all the actions on the transactions and blockchain. Ladger seals all the transaction actions in the blockchain.

```go
type Ledger struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L157-L168>)

```go
func New(config Config, bc BlockReadWriter, db DataBaseProvider, ac AddressChecker, vr SignatureVerifier, tf BlockFindWriter, log logger.Logger, blcPub BlockReactivePublisher, trxIssuedPub TrxIssuedReactivePunlisher, sub BlockchainLockSubscriber) (*Ledger, error)
```

New creates new Ledger if config is valid or returns error otherwise.

<a name="Ledger.Run"></a>
### func \(\*Ledger\) [Run](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L193>)

```go
func (l *Ledger) Run(ctx context.Context) error
```

Run runs the Ladger engine that writes blocks to the blockchain repository. Run starts a goroutine and can be stopped by cancelling the context. It is non\-blocking and concurrent safe.

<a name="Ledger.VerifySignature"></a>
### func \(\*Ledger\) [VerifySignature](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L276>)

```go
func (l *Ledger) VerifySignature(message, signature []byte, hash [32]byte, address string) error
```

VerifySignature verifies signature of the message.

<a name="Ledger.WriteCandidateTransaction"></a>
### func \(\*Ledger\) [WriteCandidateTransaction](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L261>)

```go
func (l *Ledger) WriteCandidateTransaction(ctx context.Context, trx *transaction.Transaction) error
```

WriteCandidateTransaction validates and writes a transaction to the repository. Transaction is not yet a part of the blockchain at this point. Ladger will perform all the necessary checks and validations before writing it to the repository. The candidate needs to be signed by the receiver later in the process to be placed as a candidate in the blockchain.

<a name="Ledger.WriteIssuerSignedTransactionForReceiver"></a>
### func \(\*Ledger\) [WriteIssuerSignedTransactionForReceiver](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L240-L243>)

```go
func (l *Ledger) WriteIssuerSignedTransactionForReceiver(ctx context.Context, trx *transaction.Transaction) error
```

WriteIssuerSignedTransactionForReceiver validates issuer signature and writes a transaction to the repository for receiver.

<a name="NodeRegister"></a>
## type [NodeRegister](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L87-L89>)

NodeRegister abstracts node registration operations.

```go
type NodeRegister interface {
    CountRegistered(ctx context.Context) (int, error)
}
```

<a name="SignatureVerifier"></a>
## type [SignatureVerifier](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L77-L79>)

SignatureVerifier provides signature verification method.

```go
type SignatureVerifier interface {
    Verify(message, signature []byte, hash [32]byte, address string) error
}
```

<a name="Synchronizer"></a>
## type [Synchronizer](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/synch.go#L19-L23>)

Synchronizer abstracts blockchain synchronization operations.

```go
type Synchronizer interface {
    AddToBlockchainLockQueue(ctx context.Context, nodeID string) error
    RemoveFromBlockchainLocks(ctx context.Context, nodeID string) error
    CheckIsOnTopOfBlockchainsLocks(ctx context.Context, nodeID string) (bool, error)
}
```

<a name="TrxIssuedReactivePunlisher"></a>
## type [TrxIssuedReactivePunlisher](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L108-L110>)

IssuerTrxSubscription provides trx issuer address publishing method. It uses reactive package. It you are using your own implementation of reactive package take care of Publish method to be non\-blocking.

```go
type TrxIssuedReactivePunlisher interface {
    Publish(string)
}
```

<a name="TrxWriteReadMover"></a>
## type [TrxWriteReadMover](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L44-L51>)

TrxWriteReadMover provides transactions write, read and move methods. It allows to access temporary, permanent and awaiting transactions.

```go
type TrxWriteReadMover interface {
    WriteIssuerSignedTransactionForReceiver(ctx context.Context, trx *transaction.Transaction) error
    MoveTransactionsFromTemporaryToPermanent(ctx context.Context, blockHash [32]byte, hashes [][32]byte) error
    MoveTransactionFromAwaitingToTemporary(ctx context.Context, trx *transaction.Transaction) error
    ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadTemporaryTransactions(ctx context.Context, offset, limit int) ([]transaction.Transaction, error)
}
```

# configuration

```go
import "github.com/bartossh/Computantis/configuration"
```

## Index

- [type Configuration](<#Configuration>)
  - [func Read\(path string\) \(Configuration, error\)](<#Read>)


<a name="Configuration"></a>
## type [Configuration](<https://github.com/bartossh/Computantis/blob/main/configuration/configuration.go#L22-L32>)

Configuration is the main configuration of the application that corresponds to the \*.yaml file that holds the configuration.

```go
type Configuration struct {
    Server       server.Config         `yaml:"server"`
    Database     repository.DBConfig   `yaml:"database"`
    Client       walletapi.Config      `yaml:"client"`
    FileOperator fileoperations.Config `yaml:"file_operator"`
    ZincLogger   zincaddapter.Config   `yaml:"zinc_logger"`
    Validator    validator.Config      `yaml:"validator"`
    Emulator     emulator.Config       `yaml:"emulator"`
    DataProvider dataprovider.Config   `yaml:"data_provider"`
    Bookkeeper   bookkeeping.Config    `yaml:"bookkeeper"`
}
```

<a name="Read"></a>
### func [Read](<https://github.com/bartossh/Computantis/blob/main/configuration/configuration.go#L35>)

```go
func Read(path string) (Configuration, error)
```

Read reads the configuration from the file and returns the Configuration with set fields according to the yaml setup.

# dataprovider

```go
import "github.com/bartossh/Computantis/dataprovider"
```

## Index

- [type Cache](<#Cache>)
  - [func New\(ctx context.Context, cfg Config\) \*Cache](<#New>)
  - [func \(c \*Cache\) ProvideData\(address string\) \[\]byte](<#Cache.ProvideData>)
  - [func \(c \*Cache\) ValidateData\(address string, data \[\]byte\) bool](<#Cache.ValidateData>)
- [type Config](<#Config>)


<a name="Cache"></a>
## type [Cache](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L22-L26>)

Cache is a simple in\-memory cache for storing generated data.

```go
type Cache struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L29>)

```go
func New(ctx context.Context, cfg Config) *Cache
```

New creates new Cache and runs the cleaner.

<a name="Cache.ProvideData"></a>
### func \(\*Cache\) [ProvideData](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L65>)

```go
func (c *Cache) ProvideData(address string) []byte
```

ProvideData generates data and stores it referring to given address.

<a name="Cache.ValidateData"></a>
### func \(\*Cache\) [ValidateData](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L80>)

```go
func (c *Cache) ValidateData(address string, data []byte) bool
```

ValidateData checks if data is stored for given address and is not expired.

<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L12-L14>)

Config holds configuration for Cache.

```go
type Config struct {
    Longevity uint64 `yaml:"longevity"` // Data longevity in seconds.
}
```

# emulator

```go
import "github.com/bartossh/Computantis/emulator"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [func RunPublisher\(ctx context.Context, cancel context.CancelFunc, config Config, data \[\]byte\) error](<#RunPublisher>)
- [func RunSubscriber\(ctx context.Context, cancel context.CancelFunc, config Config, data \[\]byte\) error](<#RunSubscriber>)
- [type Config](<#Config>)
- [type Measurement](<#Measurement>)
- [type Message](<#Message>)


## Constants

<a name="WebHookEndpointTransaction"></a>

```go
const (
    WebHookEndpointTransaction = "/hook/transaction"
    WebHookEndpointBlock       = "hook/block"
    MessageEndpoint            = "/message"
)
```

## Variables

<a name="ErrFailedHook"></a>

```go
var ErrFailedHook = errors.New("failed to create web hook")
```

<a name="RunPublisher"></a>
## func [RunPublisher](<https://github.com/bartossh/Computantis/blob/main/emulator/publisher.go#L27>)

```go
func RunPublisher(ctx context.Context, cancel context.CancelFunc, config Config, data []byte) error
```

RunPublisher runs publisher emulator that emulates data in a buffer. Running emmulator is stopped by canceling context.

<a name="RunSubscriber"></a>
## func [RunSubscriber](<https://github.com/bartossh/Computantis/blob/main/emulator/subscriber.go#L58>)

```go
func RunSubscriber(ctx context.Context, cancel context.CancelFunc, config Config, data []byte) error
```

RunSubscriber runs subscriber emulator. To stop the subscriber cancel the context.

<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/emulator/emulator.go#L4-L11>)

Config contains configuration for the emulator Publisher and Subscriber.

```go
type Config struct {
    ClientURL      string `yaml:"client_url"`
    Port           string `yaml:"port"`
    PublicURL      string `yaml:"public_url"`
    TimeoutSeconds int64  `yaml:"timeout_seconds"`
    TickSeconds    int64  `yaml:"tick_seconds"`
    Random         bool   `yaml:"random"`
}
```

<a name="Measurement"></a>
## type [Measurement](<https://github.com/bartossh/Computantis/blob/main/emulator/emulator.go#L14-L18>)

Measurement is data structure containing measurements received in a single transaction.

```go
type Measurement struct {
    Volts int `json:"volts"`
    Mamps int `json:"m_amps"`
    Power int `json:"power"`
}
```

<a name="Message"></a>
## type [Message](<https://github.com/bartossh/Computantis/blob/main/emulator/subscriber.go#L38-L45>)

Message holds timestamp info.

```go
type Message struct {
    Status      string                  `json:"status"`
    Transaction transaction.Transaction `json:"transaction"`
    Timestamp   int64                   `json:"timestamp"`
    Volts       int                     `json:"volts"`
    MiliAmps    int                     `json:"mili_amps"`
    Power       int                     `json:"power"`
}
```

# fileoperations

```go
import "github.com/bartossh/Computantis/fileoperations"
```

## Index

- [type Config](<#Config>)
- [type Helper](<#Helper>)
  - [func New\(cfg Config, s Sealer\) Helper](<#New>)
  - [func \(h Helper\) ReadWallet\(\) \(wallet.Wallet, error\)](<#Helper.ReadWallet>)
  - [func \(h Helper\) SaveWallet\(w wallet.Wallet\) error](<#Helper.SaveWallet>)
- [type Sealer](<#Sealer>)


<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/fileoperations/fileoperations.go#L4-L7>)

Config holds configuration of the file operator Helper.

```go
type Config struct {
    WalletPath   string `yaml:"wallet_path"`   // wallet path to the wallet file
    WalletPasswd string `yaml:"wallet_passwd"` // wallet password to the wallet file in hex format
}
```

<a name="Helper"></a>
## type [Helper](<https://github.com/bartossh/Computantis/blob/main/fileoperations/fileoperations.go#L10-L13>)

Helper holds all file operation methods.

```go
type Helper struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/fileoperations/fileoperations.go#L16>)

```go
func New(cfg Config, s Sealer) Helper
```

New creates new Helper.

<a name="Helper.ReadWallet"></a>
### func \(Helper\) [ReadWallet](<https://github.com/bartossh/Computantis/blob/main/fileoperations/wallet.go#L17>)

```go
func (h Helper) ReadWallet() (wallet.Wallet, error)
```

RereadWallet reads wallet from the file.

<a name="Helper.SaveWallet"></a>
### func \(Helper\) [SaveWallet](<https://github.com/bartossh/Computantis/blob/main/fileoperations/wallet.go#L41>)

```go
func (h Helper) SaveWallet(w wallet.Wallet) error
```

SaveWallet saves wallet to the file.

<a name="Sealer"></a>
## type [Sealer](<https://github.com/bartossh/Computantis/blob/main/fileoperations/wallet.go#L11-L14>)

Sealer offers behaviour to seal the bytes returning the signature on the data.

```go
type Sealer interface {
    Encrypt(key, data []byte) ([]byte, error)
    Decrypt(key, data []byte) ([]byte, error)
}
```

# generator

```go
import "github.com/bartossh/Computantis/generator"
```

## Index

- [func GenerateToFile\(filePath string, count, vMin, vMax, maMin, maMax int\) error](<#GenerateToFile>)


<a name="GenerateToFile"></a>
## func [GenerateToFile](<https://github.com/bartossh/Computantis/blob/main/generator/generator.go#L13>)

```go
func GenerateToFile(filePath string, count, vMin, vMax, maMin, maMax int) error
```

GenerateToFile generates data to file in json format.

# httpclient

```go
import "github.com/bartossh/Computantis/httpclient"
```

## Index

- [Variables](<#variables>)
- [func MakeGet\(timeout time.Duration, url string, in any\) error](<#MakeGet>)
- [func MakeGetAuth\(timeout time.Duration, token, url string, in any\) error](<#MakeGetAuth>)
- [func MakePost\(timeout time.Duration, url string, out, in any\) error](<#MakePost>)
- [func MakePostAuth\(timeout time.Duration, token, url string, out, in any\) error](<#MakePostAuth>)


## Variables

<a name="ErrApiVersionMismatch"></a>

```go
var (
    ErrApiVersionMismatch            = fmt.Errorf("api version mismatch")
    ErrApiHeaderMismatch             = fmt.Errorf("api header mismatch")
    ErrStatusCodeMismatch            = fmt.Errorf("status code mismatch")
    ErrContentTypeMismatch           = fmt.Errorf("content type mismatch")
    ErrWalletChecksumMismatch        = fmt.Errorf("wallet checksum mismatch")
    ErrWalletVersionMismatch         = fmt.Errorf("wallet version mismatch")
    ErrServerReturnsInconsistentData = fmt.Errorf("server returns inconsistent data")
    ErrRejectedByServer              = fmt.Errorf("rejected by server")
    ErrWalletNotReady                = fmt.Errorf("wallet not ready, read wallet first")
    ErrSigningFailed                 = fmt.Errorf("signing failed")
)
```

<a name="MakeGet"></a>
## func [MakeGet](<https://github.com/bartossh/Computantis/blob/main/httpclient/httpclient.go#L36>)

```go
func MakeGet(timeout time.Duration, url string, in any) error
```



<a name="MakeGetAuth"></a>
## func [MakeGetAuth](<https://github.com/bartossh/Computantis/blob/main/httpclient/httpclient.go#L64>)

```go
func MakeGetAuth(timeout time.Duration, token, url string, in any) error
```

MakeGetAuth make a get request to the given 'url' with authorization token 'in' is a pointer to the structure to be deserialized from the received json data.

<a name="MakePost"></a>
## func [MakePost](<https://github.com/bartossh/Computantis/blob/main/httpclient/httpclient.go#L25>)

```go
func MakePost(timeout time.Duration, url string, out, in any) error
```



<a name="MakePostAuth"></a>
## func [MakePostAuth](<https://github.com/bartossh/Computantis/blob/main/httpclient/httpclient.go#L48>)

```go
func MakePostAuth(timeout time.Duration, token, url string, out, in any) error
```

MakePostAuth make a post request with serialized 'out' structure which is send to the given 'url' with authorization token 'in' is a pointer to the structure to be deserialized from the received json data.

# logger

```go
import "github.com/bartossh/Computantis/logger"
```

## Index

- [type Log](<#Log>)
- [type Logger](<#Logger>)


<a name="Log"></a>
## type [Log](<https://github.com/bartossh/Computantis/blob/main/logger/logger.go#L8-L13>)

Log is log marshaled and written in to the io.Writer of the helper implementing Logger abstraction.

```go
type Log struct {
    ID        any       `json:"_id"        bson:"_id"        db:"id"`
    CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`
    Level     string    `jon:"level"       bson:"level"      db:"level"`
    Msg       string    `json:"msg"        bson:"msg"        db:"msg"`
}
```

<a name="Logger"></a>
## type [Logger](<https://github.com/bartossh/Computantis/blob/main/logger/logger.go#L16-L22>)

Logger provides logging methods for debug, info, warning, error and fatal.

```go
type Logger interface {
    Debug(msg string)
    Info(msg string)
    Warn(msg string)
    Error(msg string)
    Fatal(msg string)
}
```

# logging

```go
import "github.com/bartossh/Computantis/logging"
```

## Index

- [type Helper](<#Helper>)
  - [func New\(callOnWriteLogErr, callOnFatal func\(error\), writers ...io.Writer\) Helper](<#New>)
  - [func \(h Helper\) Debug\(msg string\)](<#Helper.Debug>)
  - [func \(h Helper\) Error\(msg string\)](<#Helper.Error>)
  - [func \(h Helper\) Fatal\(msg string\)](<#Helper.Fatal>)
  - [func \(h Helper\) Info\(msg string\)](<#Helper.Info>)
  - [func \(h Helper\) Warn\(msg string\)](<#Helper.Warn>)


<a name="Helper"></a>
## type [Helper](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L16-L20>)

Helper helps with writing logs to io.Writers. Helper implements logger.Logger interface. Writing is done concurrently with out blocking the current thread.

```go
type Helper struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L23>)

```go
func New(callOnWriteLogErr, callOnFatal func(error), writers ...io.Writer) Helper
```

New creates new Helper.

<a name="Helper.Debug"></a>
### func \(Helper\) [Debug](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L28>)

```go
func (h Helper) Debug(msg string)
```

Debug writes debug log.

<a name="Helper.Error"></a>
### func \(Helper\) [Error](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L61>)

```go
func (h Helper) Error(msg string)
```

Error writes error log.

<a name="Helper.Fatal"></a>
### func \(Helper\) [Fatal](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L72>)

```go
func (h Helper) Fatal(msg string)
```

Fatal writes fatal log.

<a name="Helper.Info"></a>
### func \(Helper\) [Info](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L39>)

```go
func (h Helper) Info(msg string)
```

Info writes info log.

<a name="Helper.Warn"></a>
### func \(Helper\) [Warn](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L50>)

```go
func (h Helper) Warn(msg string)
```

Warn writes warning log.

# logo

```go
import "github.com/bartossh/Computantis/logo"
```

## Index

- [func Display\(\)](<#Display>)


<a name="Display"></a>
## func [Display](<https://github.com/bartossh/Computantis/blob/main/logo/logo.go#L8>)

```go
func Display()
```



# providers

```go
import "github.com/bartossh/Computantis/providers"
```

## Index

- [type GaugeProvider](<#GaugeProvider>)
- [type HistogramProvider](<#HistogramProvider>)


<a name="GaugeProvider"></a>
## type [GaugeProvider](<https://github.com/bartossh/Computantis/blob/main/providers/providers.go#L13-L21>)

GaugeProvider provides gauge telemetry capabilites.

```go
type GaugeProvider interface {
    CreateUpdateObservableGauge(name, description string)
    AddToGauge(name string, f float64) bool
    RemoveFromGauge(name string, f float64) bool
    IncrementGauge(name string) bool
    DecrementGauge(name string) bool
    SetGauge(name string, f float64) bool
    SetToCurrentTimeGauge(name string) bool
}
```

<a name="HistogramProvider"></a>
## type [HistogramProvider](<https://github.com/bartossh/Computantis/blob/main/providers/providers.go#L6-L10>)

HistogramProvider provides histogram telemetry capabilietes.

```go
type HistogramProvider interface {
    CreateUpdateObservableHistogtram(name, description string)
    RecordHistogramTime(name string, t time.Duration) bool
    RecordHistogramValue(name string, f float64) bool
}
```

# reactive

```go
import "github.com/bartossh/Computantis/reactive"
```

## Index

- [type Observable](<#Observable>)
  - [func New\[T any\]\(size int\) \*Observable\[T\]](<#New>)
  - [func \(o \*Observable\[T\]\) Publish\(v T\)](<#Observable[T].Publish>)
  - [func \(o \*Observable\[T\]\) Subscribe\(\) \*subscriber\[T\]](<#Observable[T].Subscribe>)


<a name="Observable"></a>
## type [Observable](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L25-L29>)

Observable creates a container for subscribers. This works in single producer multiple consumer pattern.

```go
type Observable[T any] struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L33>)

```go
func New[T any](size int) *Observable[T]
```

New creates Observable container that holds channels for all subscribers. size is the buffer size of each channel.

<a name="Observable[T].Publish"></a>
### func \(\*Observable\[T\]\) [Publish](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L54>)

```go
func (o *Observable[T]) Publish(v T)
```

Publish publishes value to all subscribers.

<a name="Observable[T].Subscribe"></a>
### func \(\*Observable\[T\]\) [Subscribe](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L42>)

```go
func (o *Observable[T]) Subscribe() *subscriber[T]
```

Subscribe subscribes to the container.

# repository

```go
import "github.com/bartossh/Computantis/repository"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [type DBConfig](<#DBConfig>)
- [type DataBase](<#DataBase>)
  - [func Connect\(\_ context.Context, cfg DBConfig\) \(\*DataBase, error\)](<#Connect>)
  - [func \(db DataBase\) AddToBlockchainLockQueue\(ctx context.Context, nodeID string\) error](<#DataBase.AddToBlockchainLockQueue>)
  - [func \(db DataBase\) CheckAddressExists\(ctx context.Context, addr string\) \(bool, error\)](<#DataBase.CheckAddressExists>)
  - [func \(db DataBase\) CheckIsOnTopOfBlockchainsLocks\(ctx context.Context, nodeID string\) \(bool, error\)](<#DataBase.CheckIsOnTopOfBlockchainsLocks>)
  - [func \(db DataBase\) CheckToken\(ctx context.Context, tkn string\) \(bool, error\)](<#DataBase.CheckToken>)
  - [func \(db DataBase\) CountRegistered\(ctx context.Context\) \(int, error\)](<#DataBase.CountRegistered>)
  - [func \(db DataBase\) Disconnect\(ctx context.Context\) error](<#DataBase.Disconnect>)
  - [func \(db DataBase\) FindAddress\(ctx context.Context, search string, limit int\) \(\[\]string, error\)](<#DataBase.FindAddress>)
  - [func \(db DataBase\) FindTransactionInBlockHash\(ctx context.Context, trxHash \[32\]byte\) \(\[32\]byte, error\)](<#DataBase.FindTransactionInBlockHash>)
  - [func \(db DataBase\) InvalidateToken\(ctx context.Context, token string\) error](<#DataBase.InvalidateToken>)
  - [func \(db DataBase\) IsAddressAdmin\(ctx context.Context, addr string\) \(bool, error\)](<#DataBase.IsAddressAdmin>)
  - [func \(db DataBase\) IsAddressStandard\(ctx context.Context, addr string\) \(bool, error\)](<#DataBase.IsAddressStandard>)
  - [func \(db DataBase\) IsAddressSuspended\(ctx context.Context, addr string\) \(bool, error\)](<#DataBase.IsAddressSuspended>)
  - [func \(db DataBase\) IsAddressTrusted\(ctx context.Context, addr string\) \(bool, error\)](<#DataBase.IsAddressTrusted>)
  - [func \(db DataBase\) LastBlock\(ctx context.Context\) \(block.Block, error\)](<#DataBase.LastBlock>)
  - [func \(db DataBase\) MoveTransactionFromAwaitingToTemporary\(ctx context.Context, trx \*transaction.Transaction\) error](<#DataBase.MoveTransactionFromAwaitingToTemporary>)
  - [func \(db DataBase\) MoveTransactionsFromTemporaryToPermanent\(ctx context.Context, blockHash \[32\]byte, hashes \[\]\[32\]byte\) error](<#DataBase.MoveTransactionsFromTemporaryToPermanent>)
  - [func \(db DataBase\) Ping\(ctx context.Context\) error](<#DataBase.Ping>)
  - [func \(db DataBase\) ReadApprovedTransactions\(ctx context.Context, address string, offset, limit int\) \(\[\]transaction.Transaction, error\)](<#DataBase.ReadApprovedTransactions>)
  - [func \(db DataBase\) ReadAwaitingTransactionsByIssuer\(ctx context.Context, address string\) \(\[\]transaction.Transaction, error\)](<#DataBase.ReadAwaitingTransactionsByIssuer>)
  - [func \(db DataBase\) ReadAwaitingTransactionsByReceiver\(ctx context.Context, address string\) \(\[\]transaction.Transaction, error\)](<#DataBase.ReadAwaitingTransactionsByReceiver>)
  - [func \(db DataBase\) ReadBlockByHash\(ctx context.Context, hash \[32\]byte\) \(block.Block, error\)](<#DataBase.ReadBlockByHash>)
  - [func \(db DataBase\) ReadLastNValidatorStatuses\(ctx context.Context, last int64\) \(\[\]validator.Status, error\)](<#DataBase.ReadLastNValidatorStatuses>)
  - [func \(db DataBase\) ReadRegisteredNodesAddresses\(ctx context.Context\) \(\[\]string, error\)](<#DataBase.ReadRegisteredNodesAddresses>)
  - [func \(db DataBase\) ReadRejectedTransactionsPagginate\(ctx context.Context, address string, offset, limit int\) \(\[\]transaction.Transaction, error\)](<#DataBase.ReadRejectedTransactionsPagginate>)
  - [func \(db DataBase\) ReadTemporaryTransactions\(ctx context.Context, offset, limit int\) \(\[\]transaction.Transaction, error\)](<#DataBase.ReadTemporaryTransactions>)
  - [func \(db DataBase\) RegisterNode\(ctx context.Context, n, ws string\) error](<#DataBase.RegisterNode>)
  - [func \(db DataBase\) RejectTransactions\(ctx context.Context, receiver string, trxs \[\]transaction.Transaction\) error](<#DataBase.RejectTransactions>)
  - [func \(db DataBase\) RemoveFromBlockchainLocks\(ctx context.Context, nodeID string\) error](<#DataBase.RemoveFromBlockchainLocks>)
  - [func \(DataBase\) RunMigration\(\_ context.Context\) error](<#DataBase.RunMigration>)
  - [func \(db DataBase\) UnregisterNode\(ctx context.Context, n string\) error](<#DataBase.UnregisterNode>)
  - [func \(db DataBase\) Write\(p \[\]byte\) \(n int, err error\)](<#DataBase.Write>)
  - [func \(db DataBase\) WriteAddress\(ctx context.Context, addr string\) error](<#DataBase.WriteAddress>)
  - [func \(db DataBase\) WriteBlock\(ctx context.Context, block block.Block\) error](<#DataBase.WriteBlock>)
  - [func \(db DataBase\) WriteIssuerSignedTransactionForReceiver\(ctx context.Context, trx \*transaction.Transaction\) error](<#DataBase.WriteIssuerSignedTransactionForReceiver>)
  - [func \(db DataBase\) WriteToken\(ctx context.Context, tkn string, expirationDate int64\) error](<#DataBase.WriteToken>)
  - [func \(db DataBase\) WriteValidatorStatus\(ctx context.Context, vs \*validator.Status\) error](<#DataBase.WriteValidatorStatus>)
- [type Listener](<#Listener>)
  - [func Listen\(conn string, report func\(ev pq.ListenerEventType, err error\)\) \(Listener, error\)](<#Listen>)
  - [func Subscribe\(\_ context.Context, cfg DBConfig\) \(Listener, error\)](<#Subscribe>)
  - [func \(l Listener\) Close\(\)](<#Listener.Close>)
  - [func \(l Listener\) SubscribeToLockBlockchainNotification\(ctx context.Context, c chan\<\- bool, node string\)](<#Listener.SubscribeToLockBlockchainNotification>)


## Constants

<a name="MaxLimit"></a>

```go
const (
    MaxLimit = math.MaxInt16 // MaxLimit is the maximum limit of entities read in a single for the query.
)
```

## Variables

<a name="ErrInsertFailed"></a>

```go
var (
    ErrInsertFailed                            = fmt.Errorf("insert failed")
    ErrUpdateFailed                            = fmt.Errorf("update failed")
    ErrSelectFailed                            = fmt.Errorf("select failed")
    ErrMoveFailed                              = fmt.Errorf("move failed")
    ErrScanFailed                              = fmt.Errorf("scan failed")
    ErrUnmarshalFailed                         = fmt.Errorf("unmarshal failed")
    ErrCommitFailed                            = fmt.Errorf("transaction commit failed")
    ErrTrxBeginFailed                          = fmt.Errorf("transaction begin failed")
    ErrAddingToLockQueueBlockChainFailed       = fmt.Errorf("adding to lock blockchain failed")
    ErrRemovingFromLockQueueBlockChainFailed   = fmt.Errorf("removing from lock blockchain failed")
    ErrListenFailed                            = fmt.Errorf("listen failed")
    ErrCheckingIsOnTopOfBlockchainsLocksFailed = fmt.Errorf("checking is on top of blockchains locks failed")
    ErrNodeRegisterFailed                      = fmt.Errorf("node register failed")
    ErrNodeUnregisterFailed                    = fmt.Errorf("node unregister failed")
    ErrNodeLookupFailed                        = fmt.Errorf("node lookup failed")
    ErrNodeRegisteredAddressesQueryFailed      = fmt.Errorf("node registered addresses query failed")
)
```

<a name="DBConfig"></a>
## type [DBConfig](<https://github.com/bartossh/Computantis/blob/main/repository/repopostgre.go#L37-L41>)

Config contains configuration for the database.

```go
type DBConfig struct {
    ConnStr      string `yaml:"conn_str"`      // ConnStr is the connection string to the database.
    DatabaseName string `yaml:"database_name"` // DatabaseName is the name of the database.
    IsSSL        bool   `yaml:"is_ssl"`        // IsSSL is the flag that indicates if the connection should be encrypted.
}
```

<a name="DataBase"></a>
## type [DataBase](<https://github.com/bartossh/Computantis/blob/main/repository/repopostgre.go#L44-L46>)

Database provides database access for read, write and delete of repository entities.

```go
type DataBase struct {
    // contains filtered or unexported fields
}
```

<a name="Connect"></a>
### func [Connect](<https://github.com/bartossh/Computantis/blob/main/repository/repopostgre.go#L63>)

```go
func Connect(_ context.Context, cfg DBConfig) (*DataBase, error)
```

Connect creates new connection to the repository and returns pointer to the DataBase.

<a name="DataBase.AddToBlockchainLockQueue"></a>
### func \(DataBase\) [AddToBlockchainLockQueue](<https://github.com/bartossh/Computantis/blob/main/repository/notifier.go#L105>)

```go
func (db DataBase) AddToBlockchainLockQueue(ctx context.Context, nodeID string) error
```

AddToBlockchainLockQueue adds blockchain lock to queue.

<a name="DataBase.CheckAddressExists"></a>
### func \(DataBase\) [CheckAddressExists](<https://github.com/bartossh/Computantis/blob/main/repository/address.go#L25>)

```go
func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error)
```

CheckAddressExists checks if address exists in the database.

<a name="DataBase.CheckIsOnTopOfBlockchainsLocks"></a>
### func \(DataBase\) [CheckIsOnTopOfBlockchainsLocks](<https://github.com/bartossh/Computantis/blob/main/repository/notifier.go#L124>)

```go
func (db DataBase) CheckIsOnTopOfBlockchainsLocks(ctx context.Context, nodeID string) (bool, error)
```

CheckIsOnTopOfBlockchainsLocks checks if node is on top of blockchain locks queue.

<a name="DataBase.CheckToken"></a>
### func \(DataBase\) [CheckToken](<https://github.com/bartossh/Computantis/blob/main/repository/token.go#L14>)

```go
func (db DataBase) CheckToken(ctx context.Context, tkn string) (bool, error)
```

CheckToken checks if token exists in the database is valid and didn't expire.

<a name="DataBase.CountRegistered"></a>
### func \(DataBase\) [CountRegistered](<https://github.com/bartossh/Computantis/blob/main/repository/node.go#L27>)

```go
func (db DataBase) CountRegistered(ctx context.Context) (int, error)
```

CountRegistered counts registered nodes in the database.

<a name="DataBase.Disconnect"></a>
### func \(DataBase\) [Disconnect](<https://github.com/bartossh/Computantis/blob/main/repository/repopostgre.go#L77>)

```go
func (db DataBase) Disconnect(ctx context.Context) error
```

Disconnect disconnects user from database

<a name="DataBase.FindAddress"></a>
### func \(DataBase\) [FindAddress](<https://github.com/bartossh/Computantis/blob/main/repository/address.go#L36>)

```go
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error)
```

FindAddress finds address in the database.

<a name="DataBase.FindTransactionInBlockHash"></a>
### func \(DataBase\) [FindTransactionInBlockHash](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L154>)

```go
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
```

FindTransactionInBlockHash returns block hash in to which transaction with given hash was added. If transaction is not yet added to any block, empty hash is returned.

<a name="DataBase.InvalidateToken"></a>
### func \(DataBase\) [InvalidateToken](<https://github.com/bartossh/Computantis/blob/main/repository/token.go#L44>)

```go
func (db DataBase) InvalidateToken(ctx context.Context, token string) error
```

InvalidateToken invalidates token.

<a name="DataBase.IsAddressAdmin"></a>
### func \(DataBase\) [IsAddressAdmin](<https://github.com/bartossh/Computantis/blob/main/repository/address.go#L97>)

```go
func (db DataBase) IsAddressAdmin(ctx context.Context, addr string) (bool, error)
```

IsAddressAdmin checks if address has access level admin.

<a name="DataBase.IsAddressStandard"></a>
### func \(DataBase\) [IsAddressStandard](<https://github.com/bartossh/Computantis/blob/main/repository/address.go#L79>)

```go
func (db DataBase) IsAddressStandard(ctx context.Context, addr string) (bool, error)
```

IsAddressStandard checks if address has access level standard.

<a name="DataBase.IsAddressSuspended"></a>
### func \(DataBase\) [IsAddressSuspended](<https://github.com/bartossh/Computantis/blob/main/repository/address.go#L70>)

```go
func (db DataBase) IsAddressSuspended(ctx context.Context, addr string) (bool, error)
```

IsAddressAdmin checks if address has access level suspended.

<a name="DataBase.IsAddressTrusted"></a>
### func \(DataBase\) [IsAddressTrusted](<https://github.com/bartossh/Computantis/blob/main/repository/address.go#L88>)

```go
func (db DataBase) IsAddressTrusted(ctx context.Context, addr string) (bool, error)
```

IsAddressTrusted checks if address has access level trusted.

<a name="DataBase.LastBlock"></a>
### func \(DataBase\) [LastBlock](<https://github.com/bartossh/Computantis/blob/main/repository/block.go#L12>)

```go
func (db DataBase) LastBlock(ctx context.Context) (block.Block, error)
```

LastBlock returns last block from the database.

<a name="DataBase.MoveTransactionFromAwaitingToTemporary"></a>
### func \(DataBase\) [MoveTransactionFromAwaitingToTemporary](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L21>)

```go
func (db DataBase) MoveTransactionFromAwaitingToTemporary(ctx context.Context, trx *transaction.Transaction) error
```

MoveTransactionFromAwaitingToTemporary moves awaiting transaction marking it as temporary.

<a name="DataBase.MoveTransactionsFromTemporaryToPermanent"></a>
### func \(DataBase\) [MoveTransactionsFromTemporaryToPermanent](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L123>)

```go
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, blockHash [32]byte, hashes [][32]byte) error
```

MoveTransactionsFromTemporaryToPermanent moves transactions by marking transactions with matching hash to be permanent and sets block hash field to referenced block hash.

<a name="DataBase.Ping"></a>
### func \(DataBase\) [Ping](<https://github.com/bartossh/Computantis/blob/main/repository/repopostgre.go#L82>)

```go
func (db DataBase) Ping(ctx context.Context) error
```

Ping checks if the connection to the database is still alive.

<a name="DataBase.ReadApprovedTransactions"></a>
### func \(DataBase\) [ReadApprovedTransactions](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L80>)

```go
func (db DataBase) ReadApprovedTransactions(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error)
```

ReadApprovedTransactions reads the approved transactions with pagination.

<a name="DataBase.ReadAwaitingTransactionsByIssuer"></a>
### func \(DataBase\) [ReadAwaitingTransactionsByIssuer](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L62>)

```go
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
```

ReadAwaitingTransactionsByIssuer reads up to the limit awaiting transactions paired with given issuer address. Upper limit of read all is MaxLimit constant.

<a name="DataBase.ReadAwaitingTransactionsByReceiver"></a>
### func \(DataBase\) [ReadAwaitingTransactionsByReceiver](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L56>)

```go
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
```

ReadAwaitingTransactionsByReceiver reads up to the limit transactions paired with given receiver address. Upper limit of read all is MaxLimit constant.

<a name="DataBase.ReadBlockByHash"></a>
### func \(DataBase\) [ReadBlockByHash](<https://github.com/bartossh/Computantis/blob/main/repository/block.go#L41>)

```go
func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
```

ReadBlockByHash returns block with given hash.

<a name="DataBase.ReadLastNValidatorStatuses"></a>
### func \(DataBase\) [ReadLastNValidatorStatuses](<https://github.com/bartossh/Computantis/blob/main/repository/validator.go#L25>)

```go
func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error)
```

ReadLastNValidatorStatuses reads last validator statuses from the database.

<a name="DataBase.ReadRegisteredNodesAddresses"></a>
### func \(DataBase\) [ReadRegisteredNodesAddresses](<https://github.com/bartossh/Computantis/blob/main/repository/node.go#L37>)

```go
func (db DataBase) ReadRegisteredNodesAddresses(ctx context.Context) ([]string, error)
```

ReadAddresses reads registered nodes addresses from the database.

<a name="DataBase.ReadRejectedTransactionsPagginate"></a>
### func \(DataBase\) [ReadRejectedTransactionsPagginate](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L67>)

```go
func (db DataBase) ReadRejectedTransactionsPagginate(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error)
```

ReadRejectedTransactionsPagginate reads rejected transactions with pagination.

<a name="DataBase.ReadTemporaryTransactions"></a>
### func \(DataBase\) [ReadTemporaryTransactions](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L93>)

```go
func (db DataBase) ReadTemporaryTransactions(ctx context.Context, offset, limit int) ([]transaction.Transaction, error)
```

ReadTemporaryTransactions reads transactions that are marked as temporary with offset and limit.

<a name="DataBase.RegisterNode"></a>
### func \(DataBase\) [RegisterNode](<https://github.com/bartossh/Computantis/blob/main/repository/node.go#L9>)

```go
func (db DataBase) RegisterNode(ctx context.Context, n, ws string) error
```

RegisterNode registers node in the database.

<a name="DataBase.RejectTransactions"></a>
### func \(DataBase\) [RejectTransactions](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L166>)

```go
func (db DataBase) RejectTransactions(ctx context.Context, receiver string, trxs []transaction.Transaction) error
```

RejectTransactions rejects transactions addressed to the receiver address.

<a name="DataBase.RemoveFromBlockchainLocks"></a>
### func \(DataBase\) [RemoveFromBlockchainLocks](<https://github.com/bartossh/Computantis/blob/main/repository/notifier.go#L115>)

```go
func (db DataBase) RemoveFromBlockchainLocks(ctx context.Context, nodeID string) error
```

RemoveFromBlockchainLocks removes blockchain lock from queue.

<a name="DataBase.RunMigration"></a>
### func \(DataBase\) [RunMigration](<https://github.com/bartossh/Computantis/blob/main/repository/migrations.go#L7>)

```go
func (DataBase) RunMigration(_ context.Context) error
```

RunMigration satisfies the RepositoryProvider interface as PostgreSQL migrations are run on when database is created in docker\-compose\-postgresql.yml.

<a name="DataBase.UnregisterNode"></a>
### func \(DataBase\) [UnregisterNode](<https://github.com/bartossh/Computantis/blob/main/repository/node.go#L18>)

```go
func (db DataBase) UnregisterNode(ctx context.Context, n string) error
```

UnregisterNode unregister node from the database.

<a name="DataBase.Write"></a>
### func \(DataBase\) [Write](<https://github.com/bartossh/Computantis/blob/main/repository/logger.go#L12>)

```go
func (db DataBase) Write(p []byte) (n int, err error)
```

Write writes log to the database. p is a marshaled logger.Log.

<a name="DataBase.WriteAddress"></a>
### func \(DataBase\) [WriteAddress](<https://github.com/bartossh/Computantis/blob/main/repository/address.go#L16>)

```go
func (db DataBase) WriteAddress(ctx context.Context, addr string) error
```

WriteAddress writes address to the database.

<a name="DataBase.WriteBlock"></a>
### func \(DataBase\) [WriteBlock](<https://github.com/bartossh/Computantis/blob/main/repository/block.go#L69>)

```go
func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error
```

WriteBlock writes block to the database.

<a name="DataBase.WriteIssuerSignedTransactionForReceiver"></a>
### func \(DataBase\) [WriteIssuerSignedTransactionForReceiver](<https://github.com/bartossh/Computantis/blob/main/repository/transaction.go#L33-L36>)

```go
func (db DataBase) WriteIssuerSignedTransactionForReceiver(ctx context.Context, trx *transaction.Transaction) error
```

WriteIssuerSignedTransactionForReceiver writes transaction to the storage marking it as awaiting.

<a name="DataBase.WriteToken"></a>
### func \(DataBase\) [WriteToken](<https://github.com/bartossh/Computantis/blob/main/repository/token.go#L34>)

```go
func (db DataBase) WriteToken(ctx context.Context, tkn string, expirationDate int64) error
```

WriteToken writes unique token to the database.

<a name="DataBase.WriteValidatorStatus"></a>
### func \(DataBase\) [WriteValidatorStatus](<https://github.com/bartossh/Computantis/blob/main/repository/validator.go#L12>)

```go
func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *validator.Status) error
```

WriteValidatorStatus writes validator status to the database.

<a name="Listener"></a>
## type [Listener](<https://github.com/bartossh/Computantis/blob/main/repository/notifier.go#L37-L39>)

Listener wraps listener for notifications from database. Provides methods for listening and closing.

```go
type Listener struct {
    // contains filtered or unexported fields
}
```

<a name="Listen"></a>
### func [Listen](<https://github.com/bartossh/Computantis/blob/main/repository/notifier.go#L42>)

```go
func Listen(conn string, report func(ev pq.ListenerEventType, err error)) (Listener, error)
```

Listen creates Listener for notifications from database.

<a name="Subscribe"></a>
### func [Subscribe](<https://github.com/bartossh/Computantis/blob/main/repository/repopostgre.go#L49>)

```go
func Subscribe(_ context.Context, cfg DBConfig) (Listener, error)
```

Subscribe subscribes to the database events.

<a name="Listener.Close"></a>
### func \(Listener\) [Close](<https://github.com/bartossh/Computantis/blob/main/repository/notifier.go#L100>)

```go
func (l Listener) Close()
```

Close closes listener.

<a name="Listener.SubscribeToLockBlockchainNotification"></a>
### func \(Listener\) [SubscribeToLockBlockchainNotification](<https://github.com/bartossh/Computantis/blob/main/repository/notifier.go#L53>)

```go
func (l Listener) SubscribeToLockBlockchainNotification(ctx context.Context, c chan<- bool, node string)
```

SubscribeToLockBlockchainNotification listens for blockchain lock. To stop subscription, close channel.

# serializer

```go
import "github.com/bartossh/Computantis/serializer"
```

## Index

- [func Base58Decode\(input \[\]byte\) \(\[\]byte, error\)](<#Base58Decode>)
- [func Base58Encode\(input \[\]byte\) \[\]byte](<#Base58Encode>)


<a name="Base58Decode"></a>
## func [Base58Decode](<https://github.com/bartossh/Computantis/blob/main/serializer/serializer.go#L13>)

```go
func Base58Decode(input []byte) ([]byte, error)
```

Base58Decode decodes base58 string to byte array.

<a name="Base58Encode"></a>
## func [Base58Encode](<https://github.com/bartossh/Computantis/blob/main/serializer/serializer.go#L6>)

```go
func Base58Encode(input []byte) []byte
```

Base58Encode encodes byte array to base58 string.

# server

```go
import "github.com/bartossh/Computantis/server"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [func Run\(ctx context.Context, c Config, repo Repository, bookkeeping Bookkeeper, pv RandomDataProvideValidator, tele providers.HistogramProvider, log logger.Logger, rxBlock ReactiveBlock, rxTrxIssued ReactiveTrxIssued\) error](<#Run>)
- [type AddressReaderWriterModifier](<#AddressReaderWriterModifier>)
- [type AliveResponse](<#AliveResponse>)
- [type ApprovedTransactionsResponse](<#ApprovedTransactionsResponse>)
- [type AwaitedTransactionsResponse](<#AwaitedTransactionsResponse>)
- [type Bookkeeper](<#Bookkeeper>)
- [type Config](<#Config>)
- [type CreateAddressRequest](<#CreateAddressRequest>)
- [type CreateAddressResponse](<#CreateAddressResponse>)
- [type DataToSignRequest](<#DataToSignRequest>)
- [type DataToSignResponse](<#DataToSignResponse>)
- [type DiscoverResponse](<#DiscoverResponse>)
- [type GenerateTokenRequest](<#GenerateTokenRequest>)
- [type GenerateTokenResponse](<#GenerateTokenResponse>)
- [type IssuedTransactionsResponse](<#IssuedTransactionsResponse>)
- [type Message](<#Message>)
- [type RandomDataProvideValidator](<#RandomDataProvideValidator>)
- [type ReactiveBlock](<#ReactiveBlock>)
- [type ReactiveTrxIssued](<#ReactiveTrxIssued>)
- [type Register](<#Register>)
- [type RejectedTransactionsResponse](<#RejectedTransactionsResponse>)
- [type Repository](<#Repository>)
- [type SearchAddressRequest](<#SearchAddressRequest>)
- [type SearchAddressResponse](<#SearchAddressResponse>)
- [type SearchBlockRequest](<#SearchBlockRequest>)
- [type SearchBlockResponse](<#SearchBlockResponse>)
- [type TokenWriteInvalidateChecker](<#TokenWriteInvalidateChecker>)
- [type TransactionConfirmProposeResponse](<#TransactionConfirmProposeResponse>)
- [type TransactionProposeRequest](<#TransactionProposeRequest>)
- [type TransactionsRejectRequest](<#TransactionsRejectRequest>)
- [type TransactionsRejectResponse](<#TransactionsRejectResponse>)
- [type TransactionsRequest](<#TransactionsRequest>)
- [type Verifier](<#Verifier>)


## Constants

<a name="ApiVersion"></a>

```go
const (
    ApiVersion = "1.0.0"
    Header     = "Computantis-Central"
)
```

<a name="MetricsURL"></a>

```go
const (
    MetricsURL              = "/metrics"                        // URL to check service metrics
    AliveURL                = "/alive"                          // URL to check if server is alive and version.
    DiscoverCentralNodesURL = "/discover"                       // URL to discover all running central nodes.
    SearchAddressURL        = searchGroupURL + addressURL       // URL to search for address.
    SearchBlockURL          = searchGroupURL + blockURL         // URL to search for block that contains transaction hash.
    ProposeTransactionURL   = transactionGroupURL + proposeURL  // URL to propose transaction signed by the issuer.
    ConfirmTransactionURL   = transactionGroupURL + confirmURL  // URL to confirm transaction signed by the receiver.
    RejectTransactionURL    = transactionGroupURL + rejectURL   // URL to reject transaction signed only by issuer.
    AwaitedTransactionURL   = transactionGroupURL + awaitedURL  // URL to get awaited transactions for the receiver.
    IssuedTransactionURL    = transactionGroupURL + issuedURL   // URL to get issued transactions for the issuer.
    RejectedTransactionURL  = transactionGroupURL + rejectedURL // URL to get rejected transactions for given address.
    ApprovedTransactionURL  = transactionGroupURL + approvedURL // URL to get approved transactions for given address.
    DataToValidateURL       = validatorGroupURL + dataURL       // URL to get data to validate address by signing rew message.
    CreateAddressURL        = addressGroupURL + createURL       // URL to create new address.
    GenerateTokenURL        = tokenGroupURL + generateURL       // URL to generate new token.
    WsURL                   = "/ws"                             // URL to connect to websocket.
)
```

<a name="CommandEcho"></a>

```go
const (
    CommandEcho         = "echo"
    CommandSocketList   = "socketlist"
    CommandNewBlock     = "command_new_block"
    CommandNewTrxIssued = "command_new_trx_issued"
)
```

## Variables

<a name="ErrWrongPortSpecified"></a>

```go
var (
    ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")
    ErrWrongMessageSize   = errors.New("message size must be between 1024 and 15000000")
)
```

<a name="Run"></a>
## func [Run](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L189-L193>)

```go
func Run(ctx context.Context, c Config, repo Repository, bookkeeping Bookkeeper, pv RandomDataProvideValidator, tele providers.HistogramProvider, log logger.Logger, rxBlock ReactiveBlock, rxTrxIssued ReactiveTrxIssued) error
```

Run initializes routing and runs the server. To stop the server cancel the context. It blocks until the context is canceled.

<a name="AddressReaderWriterModifier"></a>
## type [AddressReaderWriterModifier](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L101-L109>)

AddressReaderWriterModifier abstracts address operations.

```go
type AddressReaderWriterModifier interface {
    FindAddress(ctx context.Context, search string, limit int) ([]string, error)
    CheckAddressExists(ctx context.Context, address string) (bool, error)
    WriteAddress(ctx context.Context, address string) error
    IsAddressSuspended(ctx context.Context, addr string) (bool, error)
    IsAddressStandard(ctx context.Context, addr string) (bool, error)
    IsAddressTrusted(ctx context.Context, addr string) (bool, error)
    IsAddressAdmin(ctx context.Context, addr string) (bool, error)
}
```

<a name="AliveResponse"></a>
## type [AliveResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L14-L18>)

AliveResponse is a response for alive and version check.

```go
type AliveResponse struct {
    APIVersion string `json:"api_version"`
    APIHeader  string `json:"api_header"`
    Alive      bool   `json:"alive"`
}
```

<a name="ApprovedTransactionsResponse"></a>
## type [ApprovedTransactionsResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L445-L448>)

ApprovedTransactionsResponse is a response for approved transactions request.

```go
type ApprovedTransactionsResponse struct {
    ApprovedTransactions []transaction.Transaction `json:"approved_transactions"`
    Success              bool                      `json:"success"`
}
```

<a name="AwaitedTransactionsResponse"></a>
## type [AwaitedTransactionsResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L288-L291>)

AwaitedTransactionsResponse is a response for awaited transactions request.

```go
type AwaitedTransactionsResponse struct {
    AwaitedTransactions []transaction.Transaction `json:"awaited_transactions"`
    Success             bool                      `json:"success"`
}
```

<a name="Bookkeeper"></a>
## type [Bookkeeper](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L140-L145>)

Bookkeeper abstracts methods of the bookkeeping of a blockchain.

```go
type Bookkeeper interface {
    Verifier
    Run(ctx context.Context) error
    WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error
    WriteIssuerSignedTransactionForReceiver(ctx context.Context, trxBlock *transaction.Transaction) error
}
```

<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L169-L173>)

Config contains configuration of the server.

```go
type Config struct {
    WebsocketAddress string `yaml:"websocket_address"` // Address of the websocket server.
    Port             int    `yaml:"port"`              // Port to listen on.
    DataSizeBytes    int    `yaml:"data_size_bytes"`   // Size of the data to be stored in the transaction.
}
```

<a name="CreateAddressRequest"></a>
## type [CreateAddressRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L521-L527>)

CreateAddressRequest is a request to create an address.

```go
type CreateAddressRequest struct {
    Address   string   `json:"address"`
    Token     string   `json:"token"`
    Data      []byte   `json:"data"`
    Signature []byte   `json:"signature"`
    Hash      [32]byte `json:"hash"`
}
```

<a name="CreateAddressResponse"></a>
## type [CreateAddressResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L531-L534>)

Response for address creation request. If Success is true, Address contains created address in base58 format.

```go
type CreateAddressResponse struct {
    Address string `json:"address"`
    Success bool   `json:"success"`
}
```

<a name="DataToSignRequest"></a>
## type [DataToSignRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L497-L499>)

DataToSignRequest is a request to get data to sign for proving identity.

```go
type DataToSignRequest struct {
    Address string `json:"address"`
}
```

<a name="DataToSignResponse"></a>
## type [DataToSignResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L502-L504>)

DataToSignRequest is a response containing data to sign for proving identity.

```go
type DataToSignResponse struct {
    Data []byte `json:"message"`
}
```

<a name="DiscoverResponse"></a>
## type [DiscoverResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L30-L32>)

DiscoverResponse is a response containing all the central node registered in the current system.

```go
type DiscoverResponse struct {
    Sockets []string `json:"sockets"`
}
```

<a name="GenerateTokenRequest"></a>
## type [GenerateTokenRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L588-L594>)

GenerateTokenRequest is a request for token generation.

```go
type GenerateTokenRequest struct {
    Address    string   `json:"address"`
    Data       []byte   `json:"data"`
    Signature  []byte   `json:"signature"`
    Hash       [32]byte `json:"hash"`
    Expiration int64    `json:"expiration"`
}
```

<a name="GenerateTokenResponse"></a>
## type [GenerateTokenResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L597>)

GenerateTokenResponse is a response containing generated token.

```go
type GenerateTokenResponse = token.Token
```

<a name="IssuedTransactionsResponse"></a>
## type [IssuedTransactionsResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L341-L344>)

IssuedTransactionsResponse is a response for issued transactions request.

```go
type IssuedTransactionsResponse struct {
    IssuedTransactions []transaction.Transaction `json:"issued_transactions"`
    Success            bool                      `json:"success"`
}
```

<a name="Message"></a>
## type [Message](<https://github.com/bartossh/Computantis/blob/main/server/ws.go#L40-L46>)

Message is the message that is used to exchange information between the server and the client.

```go
type Message struct {
    Command               string      `json:"command"`                            // Command is the command that refers to the action handler in websocket protocol.
    Error                 string      `json:"error,omitempty"`                    // Error is the error message that is sent to the client.
    IssuedTrxForAddresses []string    `json:"issued_trx_for_addresses,omitempty"` // IssuedTrxForAddresses is the list of addresses that have issued transactions for.
    Sockets               []string    `json:"sockets,omitempty"`                  // sockets is the list of central nodes web-sockets addresses.
    Block                 block.Block `json:"block,omitempty"`                    // Block is the block that is sent to the client.
}
```

<a name="RandomDataProvideValidator"></a>
## type [RandomDataProvideValidator](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L149-L152>)

RandomDataProvideValidator provides random binary data for signing to prove identity and the validator of data being valid and not expired.

```go
type RandomDataProvideValidator interface {
    ProvideData(address string) []byte
    ValidateData(address string, data []byte) bool
}
```

<a name="ReactiveBlock"></a>
## type [ReactiveBlock](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L156-L159>)

ReactiveBlock provides reactive subscription to the blockchain. It allows to listen for the new blocks created by the Ladger.

```go
type ReactiveBlock interface {
    Cancel()
    Channel() <-chan block.Block
}
```

<a name="ReactiveTrxIssued"></a>
## type [ReactiveTrxIssued](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L163-L166>)

ReactiveTrxIssued provides reactive subscription to the issuer address. It allows to listen for the new blocks created by the Ladger.

```go
type ReactiveTrxIssued interface {
    Cancel()
    Channel() <-chan string
}
```

<a name="Register"></a>
## type [Register](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L93-L98>)

Register abstracts node registration operations.

```go
type Register interface {
    RegisterNode(ctx context.Context, n, ws string) error
    UnregisterNode(ctx context.Context, n string) error
    ReadRegisteredNodesAddresses(ctx context.Context) ([]string, error)
    CountRegistered(ctx context.Context) (int, error)
}
```

<a name="RejectedTransactionsResponse"></a>
## type [RejectedTransactionsResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L393-L396>)

RejectedTransactionsResponse is a response for rejected transactions request.

```go
type RejectedTransactionsResponse struct {
    RejectedTransactions []transaction.Transaction `json:"rejected_transactions"`
    Success              bool                      `json:"success"`
}
```

<a name="Repository"></a>
## type [Repository](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L122-L132>)

Repository is the interface that wraps the basic CRUD and Search methods. Repository should be properly indexed to allow for transaction and block hash. as well as address public keys to be and unique and the hash lookup should be fast. Repository holds the blocks and transaction that are part of the blockchain.

```go
type Repository interface {
    Register
    AddressReaderWriterModifier
    TokenWriteInvalidateChecker
    FindTransactionInBlockHash(ctx context.Context, trxBlockHash [32]byte) ([32]byte, error)
    ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadRejectedTransactionsPagginate(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error)
    ReadApprovedTransactions(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error)
    RejectTransactions(ctx context.Context, receiver string, trxs []transaction.Transaction) error
}
```

<a name="SearchAddressRequest"></a>
## type [SearchAddressRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L47-L49>)

SearchAddressRequest is a request to search for address.

```go
type SearchAddressRequest struct {
    Address string `json:"address"`
}
```

<a name="SearchAddressResponse"></a>
## type [SearchAddressResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L52-L54>)

SearchAddressResponse is a response for address search.

```go
type SearchAddressResponse struct {
    Addresses []string `json:"addresses"`
}
```

<a name="SearchBlockRequest"></a>
## type [SearchBlockRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L78-L81>)

SearchBlockRequest is a request to search for block.

```go
type SearchBlockRequest struct {
    Address    string   `json:"address"`
    RawTrxHash [32]byte `json:"raw_trx_hash"`
}
```

<a name="SearchBlockResponse"></a>
## type [SearchBlockResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L84-L86>)

SearchBlockResponse is a response for block search.

```go
type SearchBlockResponse struct {
    RawBlockHash [32]byte `json:"raw_block_hash"`
}
```

<a name="TokenWriteInvalidateChecker"></a>
## type [TokenWriteInvalidateChecker](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L112-L116>)

TokenWriteInvalidateChecker abstracts token operations.

```go
type TokenWriteInvalidateChecker interface {
    WriteToken(ctx context.Context, tkn string, expirationDate int64) error
    CheckToken(ctx context.Context, token string) (bool, error)
    InvalidateToken(ctx context.Context, token string) error
}
```

<a name="TransactionConfirmProposeResponse"></a>
## type [TransactionConfirmProposeResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L126-L129>)

TransactionConfirmProposeResponse is a response for transaction propose.

```go
type TransactionConfirmProposeResponse struct {
    TrxHash [32]byte `json:"trx_hash"`
    Success bool     `json:"success"`
}
```

<a name="TransactionProposeRequest"></a>
## type [TransactionProposeRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L120-L123>)

TransactionProposeRequest is a request to propose a transaction.

```go
type TransactionProposeRequest struct {
    ReceiverAddr string                  `json:"receiver_addr"`
    Transaction  transaction.Transaction `json:"transaction"`
}
```

<a name="TransactionsRejectRequest"></a>
## type [TransactionsRejectRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L212-L218>)

TransactionsRejectRequest is a request to reject a transactions.

```go
type TransactionsRejectRequest struct {
    Address      string                    `json:"address"`
    Transactions []transaction.Transaction `json:"transaction"`
    Data         []byte                    `json:"data"`
    Signature    []byte                    `json:"signature"`
    Hash         [32]byte                  `json:"hash"`
}
```

<a name="TransactionsRejectResponse"></a>
## type [TransactionsRejectResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L221-L224>)

TransactionsRejectResponse is a response for transaction reject.

```go
type TransactionsRejectResponse struct {
    TrxHashes [][32]byte `json:"trx_hash"`
    Success   bool       `json:"success"`
}
```

<a name="TransactionsRequest"></a>
## type [TransactionsRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L278-L285>)

TransactionsRequest is a request to get awaited, issued or rejected transactions for given address. Request contains of Address for which Transactions are requested, Data in binary format, Hash of Data and Signature of the Data to prove that entity doing the request is an Address owner.

```go
type TransactionsRequest struct {
    Address   string   `json:"address"`
    Data      []byte   `json:"data"`
    Signature []byte   `json:"signature"`
    Hash      [32]byte `json:"hash"`
    Offset    int      `json:"offset,omitempty"`
    Limit     int      `json:"limit,omitempty"`
}
```

<a name="Verifier"></a>
## type [Verifier](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L135-L137>)

Verifier provides methods to verify the signature of the message.

```go
type Verifier interface {
    VerifySignature(message, signature []byte, hash [32]byte, address string) error
}
```

# stdoutwriter

```go
import "github.com/bartossh/Computantis/stdoutwriter"
```

## Index

- [type Logger](<#Logger>)
  - [func \(l Logger\) Write\(p \[\]byte\) \(n int, err error\)](<#Logger.Write>)


<a name="Logger"></a>
## type [Logger](<https://github.com/bartossh/Computantis/blob/main/stdoutwriter/stdoutwriter.go#L5>)



```go
type Logger struct{}
```

<a name="Logger.Write"></a>
### func \(Logger\) [Write](<https://github.com/bartossh/Computantis/blob/main/stdoutwriter/stdoutwriter.go#L7>)

```go
func (l Logger) Write(p []byte) (n int, err error)
```



# stress

```go
import "github.com/bartossh/Computantis/stress"
```

Stress is a package that provides a simple way to stress test your code on the full cycle of transaction processing.

## Index



# telemetry

```go
import "github.com/bartossh/Computantis/telemetry"
```

## Index

- [type Measurements](<#Measurements>)
  - [func Run\(ctx context.Context, cancel context.CancelFunc, port int\) \(\*Measurements, error\)](<#Run>)
  - [func \(m \*Measurements\) AddToGauge\(name string, f float64\) bool](<#Measurements.AddToGauge>)
  - [func \(m \*Measurements\) CreateUpdateObservableGauge\(name, description string\)](<#Measurements.CreateUpdateObservableGauge>)
  - [func \(m \*Measurements\) CreateUpdateObservableHistogtram\(name, description string\)](<#Measurements.CreateUpdateObservableHistogtram>)
  - [func \(m \*Measurements\) DecrementGauge\(name string\) bool](<#Measurements.DecrementGauge>)
  - [func \(m \*Measurements\) IncrementGauge\(name string\) bool](<#Measurements.IncrementGauge>)
  - [func \(m \*Measurements\) RecordHistogramTime\(name string, t time.Duration\) bool](<#Measurements.RecordHistogramTime>)
  - [func \(m \*Measurements\) RecordHistogramValue\(name string, f float64\) bool](<#Measurements.RecordHistogramValue>)
  - [func \(m \*Measurements\) RemoveFromGauge\(name string, f float64\) bool](<#Measurements.RemoveFromGauge>)
  - [func \(m \*Measurements\) SetGauge\(name string, f float64\) bool](<#Measurements.SetGauge>)
  - [func \(m \*Measurements\) SetToCurrentTimeGauge\(name string\) bool](<#Measurements.SetToCurrentTimeGauge>)


<a name="Measurements"></a>
## type [Measurements](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L20-L23>)

Measurements collects measurements for prometheus.

```go
type Measurements struct {
    // contains filtered or unexported fields
}
```

<a name="Run"></a>
### func [Run](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L121>)

```go
func Run(ctx context.Context, cancel context.CancelFunc, port int) (*Measurements, error)
```

Run starts collecting metrics and server with prometheus telemetry endpoint. Returns Measurements structure if successfully started or cancels context otherwise. Default port of 2112 is used if port value is set to 0.

<a name="Measurements.AddToGauge"></a>
### func \(\*Measurements\) [AddToGauge](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L65>)

```go
func (m *Measurements) AddToGauge(name string, f float64) bool
```

AddToGeuge adds to gauge the value if entity with given name exists.

<a name="Measurements.CreateUpdateObservableGauge"></a>
### func \(\*Measurements\) [CreateUpdateObservableGauge](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L55>)

```go
func (m *Measurements) CreateUpdateObservableGauge(name, description string)
```

CreateUpdateObservableGauge creats or updates observable gauge.

<a name="Measurements.CreateUpdateObservableHistogtram"></a>
### func \(\*Measurements\) [CreateUpdateObservableHistogtram](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L26>)

```go
func (m *Measurements) CreateUpdateObservableHistogtram(name, description string)
```

CreateUpdateObservableHistogtram creats or updates observable histogram.

<a name="Measurements.DecrementGauge"></a>
### func \(\*Measurements\) [DecrementGauge](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L92>)

```go
func (m *Measurements) DecrementGauge(name string) bool
```

DecrementGeuge decrements gauge the value if entity with given name exists.

<a name="Measurements.IncrementGauge"></a>
### func \(\*Measurements\) [IncrementGauge](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L83>)

```go
func (m *Measurements) IncrementGauge(name string) bool
```

IncrementGeuge increments gauge the value if entity with given name exists.

<a name="Measurements.RecordHistogramTime"></a>
### func \(\*Measurements\) [RecordHistogramTime](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L36>)

```go
func (m *Measurements) RecordHistogramTime(name string, t time.Duration) bool
```

RecordHistogramTime records histogram time if entity with given name exists.

<a name="Measurements.RecordHistogramValue"></a>
### func \(\*Measurements\) [RecordHistogramValue](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L46>)

```go
func (m *Measurements) RecordHistogramValue(name string, f float64) bool
```

RecordHistogramValue records histogram value if entity with given name exists.

<a name="Measurements.RemoveFromGauge"></a>
### func \(\*Measurements\) [RemoveFromGauge](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L74>)

```go
func (m *Measurements) RemoveFromGauge(name string, f float64) bool
```

SubstractFromGeuge substracts from gauge the value if entity with given name exists.

<a name="Measurements.SetGauge"></a>
### func \(\*Measurements\) [SetGauge](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L101>)

```go
func (m *Measurements) SetGauge(name string, f float64) bool
```

SetGeuge sets the gauge to the value if entity with given name exists.

<a name="Measurements.SetToCurrentTimeGauge"></a>
### func \(\*Measurements\) [SetToCurrentTimeGauge](<https://github.com/bartossh/Computantis/blob/main/telemetry/telemetry.go#L110>)

```go
func (m *Measurements) SetToCurrentTimeGauge(name string) bool
```

SetToCurrentTimeGeuge sets the gauge to the current time if entity with given name exists.

# token

```go
import "github.com/bartossh/Computantis/token"
```

## Index

- [type Token](<#Token>)
  - [func New\(expiration int64\) \(Token, error\)](<#New>)


<a name="Token"></a>
## type [Token](<https://github.com/bartossh/Computantis/blob/main/token/token.go#L22-L27>)

Token holds information about unique token. Token is a way of proving to the REST API of the central server that the request is valid and comes from the client that is allowed to use the API.

```go
type Token struct {
    ID             any    `json:"-"               bson:"_id,omitempty"   db:"id"`
    Token          string `json:"token"           bson:"token"           db:"token"`
    Valid          bool   `json:"valid"           bson:"valid"           db:"valid"`
    ExpirationDate int64  `json:"expiration_date" bson:"expiration_date" db:"expiration_date"`
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/token/token.go#L30>)

```go
func New(expiration int64) (Token, error)
```

New creates new token.

# transaction

```go
import "github.com/bartossh/Computantis/transaction"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [type Signer](<#Signer>)
- [type Transaction](<#Transaction>)
  - [func New\(subject string, data \[\]byte, receiverAddress string, issuer Signer\) \(Transaction, error\)](<#New>)
  - [func \(t \*Transaction\) GetMessage\(\) \[\]byte](<#Transaction.GetMessage>)
  - [func \(t \*Transaction\) Sign\(receiver Signer, v Verifier\) \(\[32\]byte, error\)](<#Transaction.Sign>)
- [type TransactionAwaitingReceiverSignature](<#TransactionAwaitingReceiverSignature>)
- [type TransactionInBlock](<#TransactionInBlock>)
- [type Verifier](<#Verifier>)


## Constants

<a name="ExpirationTimeInDays"></a>

```go
const (
    ExpirationTimeInDays = 7 // transaction validity expiration time in days. TODO: move to config
)
```

## Variables

<a name="ErrTransactionHasAFutureTime"></a>

```go
var (
    ErrTransactionHasAFutureTime        = errors.New("transaction has a future time")
    ErrExpiredTransaction               = errors.New("transaction has expired")
    ErrTransactionHashIsInvalid         = errors.New("transaction hash is invalid")
    ErrSignatureNotValidOrDataCorrupted = errors.New("signature not valid or data are corrupted")
    ErrSubjectIsEmpty                   = errors.New("subject cannot be empty")
    ErrAddressIsInvalid                 = errors.New("address is invalid")
)
```

<a name="Signer"></a>
## type [Signer](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L27-L30>)

Signer provides signing and address methods.

```go
type Signer interface {
    Sign(message []byte) (digest [32]byte, signature []byte)
    Address() string
}
```

<a name="Transaction"></a>
## type [Transaction](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L44-L54>)

Transaction contains transaction information, subject type, subject data, signatures and public keys. Transaction is valid for a week from being issued. Subject represents an information how to read the Data and / or how to decode them. Data is not validated by the computantis server, Ladger ior block. What is stored in Data is not important for the whole Computantis system. It is only important that the data are signed by the issuer and the receiver and both parties agreed on them.

```go
type Transaction struct {
    ID                any       `json:"-"                  bson:"_id"                db:"id"`
    CreatedAt         time.Time `json:"created_at"         bson:"created_at"         db:"created_at"`
    IssuerAddress     string    `json:"issuer_address"     bson:"issuer_address"     db:"issuer_address"`
    ReceiverAddress   string    `json:"receiver_address"   bson:"receiver_address"   db:"receiver_address"`
    Subject           string    `json:"subject"            bson:"subject"            db:"subject"`
    Data              []byte    `json:"data"               bson:"data"               db:"data"`
    IssuerSignature   []byte    `json:"issuer_signature"   bson:"issuer_signature"   db:"issuer_signature"`
    ReceiverSignature []byte    `json:"receiver_signature" bson:"receiver_signature" db:"receiver_signature"`
    Hash              [32]byte  `json:"hash"               bson:"hash"               db:"hash"`
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L57>)

```go
func New(subject string, data []byte, receiverAddress string, issuer Signer) (Transaction, error)
```

New creates new transaction signed by the issuer.

<a name="Transaction.GetMessage"></a>
### func \(\*Transaction\) [GetMessage](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L123>)

```go
func (t *Transaction) GetMessage() []byte
```

GeMessage returns message used for signature validation.

<a name="Transaction.Sign"></a>
### func \(\*Transaction\) [Sign](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L90>)

```go
func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error)
```

Sign verifies issuer signature and signs Transaction by the receiver.

<a name="TransactionAwaitingReceiverSignature"></a>
## type [TransactionAwaitingReceiverSignature](<https://github.com/bartossh/Computantis/blob/main/transaction/entities.go#L13-L19>)

TransactionAwaitingReceiverSignature represents transaction awaiting receiver signature. It is as well the entity of all issued transactions that has not been signed by receiver yet.

```go
type TransactionAwaitingReceiverSignature struct {
    ID              any         `json:"-"                bson:"_id,omitempty"    db:"id"`
    ReceiverAddress string      `json:"receiver_address" bson:"receiver_address" db:"receiver_address"`
    IssuerAddress   string      `json:"issuer_address"   bson:"issuer_address"   db:"issuer_address"`
    Transaction     Transaction `json:"transaction"      bson:"transaction"      db:"-"`
    TransactionHash [32]byte    `json:"transaction_hash" bson:"transaction_hash" db:"hash"`
}
```

<a name="TransactionInBlock"></a>
## type [TransactionInBlock](<https://github.com/bartossh/Computantis/blob/main/transaction/entities.go#L5-L9>)

TransactionInBlock stores relation between Transaction and Block to which Transaction was added. It is stored for fast lookup only to allow to find Block hash in which Transaction was added.

```go
type TransactionInBlock struct {
    ID              any      `json:"-" bson:"_id,omitempty"    db:"id"`
    BlockHash       [32]byte `json:"-" bson:"block_hash"       db:"block_hash"`
    TransactionHash [32]byte `json:"-" bson:"transaction_hash" db:"transaction_hash"`
}
```

<a name="Verifier"></a>
## type [Verifier](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L33-L35>)

Verifier provides signature verification method.

```go
type Verifier interface {
    Verify(message, signature []byte, hash [32]byte, issuer string) error
}
```

# validator

```go
import "github.com/bartossh/Computantis/validator"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [func Run\(ctx context.Context, cfg Config, srw StatusReadWriter, log logger.Logger, ver Verifier, wh WebhookCreateRemovePoster, wallet \*wallet.Wallet, rdp server.RandomDataProvideValidator\) error](<#Run>)
- [type Config](<#Config>)
- [type CreateRemoveUpdateHookRequest](<#CreateRemoveUpdateHookRequest>)
- [type CreateRemoveUpdateHookResponse](<#CreateRemoveUpdateHookResponse>)
- [type Status](<#Status>)
- [type StatusReadWriter](<#StatusReadWriter>)
- [type Verifier](<#Verifier>)
- [type WebhookCreateRemovePoster](<#WebhookCreateRemovePoster>)


## Constants

<a name="AliveURL"></a>

```go
const (
    AliveURL           = server.AliveURL          // URL to check is service alive
    MetricsURL         = server.MetricsURL        // URL to serve service metrics over http.
    DataEndpointURL    = server.DataToValidateURL // URL to serve data to sign to prove identity.
    BloclHookURL       = "/block/new"             // URL allows to create block hook.
    TransactionHookURL = "/transaction/new"       // URL allows to create transaction hook.
)
```

<a name="Header"></a>

```go
const (
    Header = "Computantis-Validator"
)
```

## Variables

<a name="ErrProofBlockIsInvalid"></a>

```go
var (
    ErrProofBlockIsInvalid    = fmt.Errorf("block proof is invalid")
    ErrBlockIndexIsInvalid    = fmt.Errorf("block index is invalid")
    ErrBlockPrevHashIsInvalid = fmt.Errorf("block previous hash is invalid")
)
```

<a name="Run"></a>
## func [Run](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L99-L104>)

```go
func Run(ctx context.Context, cfg Config, srw StatusReadWriter, log logger.Logger, ver Verifier, wh WebhookCreateRemovePoster, wallet *wallet.Wallet, rdp server.RandomDataProvideValidator) error
```

Run initializes routing and runs the validator. To stop the validator cancel the context. Validator connects to the central server via websocket and listens for new blocks. It will block until the context is canceled.

<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L76-L80>)

Config contains configuration of the validator.

```go
type Config struct {
    Token              string `yaml:"token"`                // token is used to authenticate validator in the central server
    CentralNodeAddress string `yaml:"central_node_address"` // address of the central server
    Port               int    `yaml:"port"`                 // port on which validator will listen for http requests
}
```

<a name="CreateRemoveUpdateHookRequest"></a>
## type [CreateRemoveUpdateHookRequest](<https://github.com/bartossh/Computantis/blob/main/validator/webhook.go#L4-L10>)

CreateRemoveUpdateHookRequest is the request send to create, remove or update the webhook.

```go
type CreateRemoveUpdateHookRequest struct {
    URL       string   `json:"address"`        // URL is a url  of the webhook.
    Address   string   `json:"wallet_address"` // Address is the address of the wallet that is used to sign the webhook.
    Data      []byte   `json:"data"`           // Data is the data is a subject of the signature. It is signed by the wallet address.
    Signature []byte   `json:"signature"`      // Signature is the signature of the data. It is used to verify that the data is not changed.
    Digest    [32]byte `json:"digest"`         // Digest is the digest of the data. It is used to verify that the data is not changed.
}
```

<a name="CreateRemoveUpdateHookResponse"></a>
## type [CreateRemoveUpdateHookResponse](<https://github.com/bartossh/Computantis/blob/main/validator/webhook.go#L13-L16>)

CreateRemoveUpdateHookResponse is the response send back to the webhook creator.

```go
type CreateRemoveUpdateHookResponse struct {
    Err string `json:"error"`
    Ok  bool   `json:"ok"`
}
```

<a name="Status"></a>
## type [Status](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L48-L54>)

Status is a status of each received block by the validator. It keeps track of invalid blocks in case of blockchain corruption.

```go
type Status struct {
    ID        any         `json:"-"          bson:"_id,omitempty" db:"id"`
    CreatedAt time.Time   `json:"created_at" bson:"created_at"    db:"created_at"`
    Block     block.Block `json:"block"      bson:"block"         db:"-"`
    Index     int64       `json:"index"      bson:"index"         db:"index"`
    Valid     bool        `json:"valid"      bson:"valid"         db:"valid"`
}
```

<a name="StatusReadWriter"></a>
## type [StatusReadWriter](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L57-L60>)

StatusReadWriter provides methods to bulk read and single write validator status.

```go
type StatusReadWriter interface {
    WriteValidatorStatus(ctx context.Context, vs *Status) error
    ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]Status, error)
}
```

<a name="Verifier"></a>
## type [Verifier](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L71-L73>)

Verifier provides methods to verify the signature of the message.

```go
type Verifier interface {
    Verify(message, signature []byte, hash [32]byte, address string) error
}
```

<a name="WebhookCreateRemovePoster"></a>
## type [WebhookCreateRemovePoster](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L63-L68>)

WebhookCreateRemovePoster provides methods to create, remove webhooks and post messages to webhooks.

```go
type WebhookCreateRemovePoster interface {
    CreateWebhook(trigger byte, address string, h webhooks.Hook) error
    RemoveWebhook(trigger byte, address string, h webhooks.Hook) error
    PostWebhookBlock(blc *block.Block)
    PostWebhookNewTransaction(publicAddresses []string)
}
```

# wallet

```go
import "github.com/bartossh/Computantis/wallet"
```

## Index

- [type Helper](<#Helper>)
  - [func NewVerifier\(\) Helper](<#NewVerifier>)
  - [func \(h Helper\) AddressToPubKey\(address string\) \(ed25519.PublicKey, error\)](<#Helper.AddressToPubKey>)
  - [func \(h Helper\) Verify\(message, signature \[\]byte, hash \[32\]byte, address string\) error](<#Helper.Verify>)
- [type Wallet](<#Wallet>)
  - [func DecodeGOBWallet\(data \[\]byte\) \(Wallet, error\)](<#DecodeGOBWallet>)
  - [func New\(\) \(Wallet, error\)](<#New>)
  - [func \(w \*Wallet\) Address\(\) string](<#Wallet.Address>)
  - [func \(w \*Wallet\) ChecksumLength\(\) int](<#Wallet.ChecksumLength>)
  - [func \(w \*Wallet\) EncodeGOB\(\) \(\[\]byte, error\)](<#Wallet.EncodeGOB>)
  - [func \(w \*Wallet\) Sign\(message \[\]byte\) \(digest \[32\]byte, signature \[\]byte\)](<#Wallet.Sign>)
  - [func \(w \*Wallet\) Verify\(message, signature \[\]byte, hash \[32\]byte\) bool](<#Wallet.Verify>)
  - [func \(w \*Wallet\) Version\(\) byte](<#Wallet.Version>)


<a name="Helper"></a>
## type [Helper](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L13>)

Helper provides wallet helper functionalities without knowing about wallet private and public keys.

```go
type Helper struct{}
```

<a name="NewVerifier"></a>
### func [NewVerifier](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L16>)

```go
func NewVerifier() Helper
```

NewVerifier creates new wallet Helper verifier.

<a name="Helper.AddressToPubKey"></a>
### func \(Helper\) [AddressToPubKey](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L21>)

```go
func (h Helper) AddressToPubKey(address string) (ed25519.PublicKey, error)
```

AddressToPubKey creates ED25519 public key from address, or returns error otherwise.

<a name="Helper.Verify"></a>
### func \(Helper\) [Verify](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L42>)

```go
func (h Helper) Verify(message, signature []byte, hash [32]byte, address string) error
```

Verify verifies if message is signed by given key and hash is equal.

<a name="Wallet"></a>
## type [Wallet](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L20-L23>)

Wallet holds public and private key of the wallet owner.

```go
type Wallet struct {
    Private ed25519.PrivateKey `json:"private" bson:"private"`
    Public  ed25519.PublicKey  `json:"public" bson:"public"`
}
```

<a name="DecodeGOBWallet"></a>
### func [DecodeGOBWallet](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L35>)

```go
func DecodeGOBWallet(data []byte) (Wallet, error)
```

DecodeGOBWallet tries to decode Wallet from gob representation or returns error otherwise.

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L26>)

```go
func New() (Wallet, error)
```

New tries to creates a new Wallet or returns error otherwise.

<a name="Wallet.Address"></a>
### func \(\*Wallet\) [Address](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L70>)

```go
func (w *Wallet) Address() string
```

Address creates address from the public key that contains wallet version and checksum.

<a name="Wallet.ChecksumLength"></a>
### func \(\*Wallet\) [ChecksumLength](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L60>)

```go
func (w *Wallet) ChecksumLength() int
```

ChecksumLength returns checksum length.

<a name="Wallet.EncodeGOB"></a>
### func \(\*Wallet\) [EncodeGOB](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L47>)

```go
func (w *Wallet) EncodeGOB() ([]byte, error)
```

EncodeGOB tries to encodes Wallet in to the gob representation or returns error otherwise.

<a name="Wallet.Sign"></a>
### func \(\*Wallet\) [Sign](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L82>)

```go
func (w *Wallet) Sign(message []byte) (digest [32]byte, signature []byte)
```

Sign signs the message with Ed25519 signature. Returns digest hash sha256 and signature.

<a name="Wallet.Verify"></a>
### func \(\*Wallet\) [Verify](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L90>)

```go
func (w *Wallet) Verify(message, signature []byte, hash [32]byte) bool
```

Verify verifies message ED25519 signature and hash. Uses hashing sha256.

<a name="Wallet.Version"></a>
### func \(\*Wallet\) [Version](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L65>)

```go
func (w *Wallet) Version() byte
```

Version returns wallet version.

# walletapi

```go
import "github.com/bartossh/Computantis/walletapi"
```

## Index

- [Constants](<#constants>)
- [func Run\(ctx context.Context, cfg Config, log logger.Logger, timeout time.Duration, fw transaction.Verifier, wrs walletmiddleware.WalletReadSaver, walletCreator walletmiddleware.NewSignValidatorCreator\) error](<#Run>)
- [type AddressResponse](<#AddressResponse>)
- [type AliveResponse](<#AliveResponse>)
- [type ApprovedTransactionResponse](<#ApprovedTransactionResponse>)
- [type Config](<#Config>)
- [type ConfirmTransactionRequest](<#ConfirmTransactionRequest>)
- [type ConfirmTransactionResponse](<#ConfirmTransactionResponse>)
- [type CreateWalletRequest](<#CreateWalletRequest>)
- [type CreateWalletResponse](<#CreateWalletResponse>)
- [type CreateWebHookRequest](<#CreateWebHookRequest>)
- [type CreateWebhookResponse](<#CreateWebhookResponse>)
- [type IssueTransactionRequest](<#IssueTransactionRequest>)
- [type IssueTransactionResponse](<#IssueTransactionResponse>)
- [type IssuedTransactionResponse](<#IssuedTransactionResponse>)
- [type ReadWalletPublicAddressResponse](<#ReadWalletPublicAddressResponse>)
- [type ReceivedTransactionResponse](<#ReceivedTransactionResponse>)
- [type RejectTransactionsRequest](<#RejectTransactionsRequest>)
- [type RejectTransactionsResponse](<#RejectTransactionsResponse>)
- [type RejectedTransactionResponse](<#RejectedTransactionResponse>)


## Constants

<a name="MetricsURL"></a>

```go
const (
    MetricsURL              = server.MetricsURL                       // URL serves service metrics.
    Alive                   = server.AliveURL                         // URL allows to check if server is alive and if sign service is of the same version.
    Address                 = "/address"                              // URL allows to check wallet public address
    IssueTransaction        = "/transactions/issue"                   // URL allows to issue transaction signed by the issuer.
    ConfirmTransaction      = "/transaction/sign"                     // URL allows to sign transaction received by the receiver.
    RejectTransactions      = "/transactions/reject"                  // URL allows to reject transactions received by the receiver.
    GetIssuedTransactions   = "/transactions/issued"                  // URL allows to get issued transactions for the issuer.
    GetReceivedTransactions = "/transactions/received"                // URL allows to get received transactions for the receiver.
    GetRejectedTransactions = "/transactions/rejected/:offset/:limit" // URL allows to get rejected transactions with pagination.
    GetApprovedTransactions = "/transactions/approved/:offset/:limit" // URL allows to get approved transactions with pagination.
    CreateWallet            = "/wallet/create"                        // URL allows to create new wallet.
    CreateUpdateWebhook     = "/webhook/create"                       // URL allows to creatre webhook
    ReadWalletPublicAddress = "/wallet/address"                       // URL allows to read public address of the wallet.
    GetOneDayToken          = "token/day"                             // URL allows to get one day token.
    GetOneWeekToken         = "token/week"                            // URL allows to get one week token.
)
```

<a name="Run"></a>
## func [Run](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L58-L59>)

```go
func Run(ctx context.Context, cfg Config, log logger.Logger, timeout time.Duration, fw transaction.Verifier, wrs walletmiddleware.WalletReadSaver, walletCreator walletmiddleware.NewSignValidatorCreator) error
```

Run runs the service application that exposes the API for creating, validating and signing transactions. This blocks until the context is canceled.

<a name="AddressResponse"></a>
## type [AddressResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L141-L143>)

AddressResponse is wallet public address response.

```go
type AddressResponse struct {
    Address string `json:"address"`
}
```

<a name="AliveResponse"></a>
## type [AliveResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L126>)

AliveResponse is containing server alive data such as ApiVersion and APIHeader.

```go
type AliveResponse server.AliveResponse
```

<a name="ApprovedTransactionResponse"></a>
## type [ApprovedTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L314-L318>)

ApprovedTransactionResponse is a response of approved transactions.

```go
type ApprovedTransactionResponse struct {
    Ok           bool                      `json:"ok"`
    Err          string                    `json:"err"`
    Transactions []transaction.Transaction `json:"transactions"`
}
```

<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L21-L25>)

Config is the configuration for the server

```go
type Config struct {
    Port             string `yaml:"port"`
    CentralNodeURL   string `yaml:"central_node_url"`
    ValidatorNodeURL string `yaml:"validator_node_url"`
}
```

<a name="ConfirmTransactionRequest"></a>
## type [ConfirmTransactionRequest](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L190-L192>)

ConfirmTransactionRequest is a request to confirm transaction.

```go
type ConfirmTransactionRequest struct {
    Transaction transaction.Transaction `json:"transaction"`
}
```

<a name="ConfirmTransactionResponse"></a>
## type [ConfirmTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L195-L198>)

ConfirmTransactionResponse is response of confirming transaction.

```go
type ConfirmTransactionResponse struct {
    Ok  bool   `json:"ok"`
    Err string `json:"err"`
}
```

<a name="CreateWalletRequest"></a>
## type [CreateWalletRequest](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L345-L347>)

CreateWalletRequest is a request to create wallet.

```go
type CreateWalletRequest struct {
    Token string `json:"token"`
}
```

<a name="CreateWalletResponse"></a>
## type [CreateWalletResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L350-L353>)

CreateWalletResponse is response to create wallet.

```go
type CreateWalletResponse struct {
    Ok  bool   `json:"ok"`
    Err string `json:"err"`
}
```

<a name="CreateWebHookRequest"></a>
## type [CreateWebHookRequest](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L379-L381>)

CreateWebHookRequest is a request to create a web hook

```go
type CreateWebHookRequest struct {
    URL string `json:"url"`
}
```

<a name="CreateWebhookResponse"></a>
## type [CreateWebhookResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L384-L387>)

CreateWebhookResponse is a response describing effect of creating a web hook

```go
type CreateWebhookResponse struct {
    Ok  bool   `json:"ok"`
    Err string `json:"error"`
}
```

<a name="IssueTransactionRequest"></a>
## type [IssueTransactionRequest](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L161-L165>)

IssueTransactionRequest is a request message that contains data and subject of the transaction to be issued.

```go
type IssueTransactionRequest struct {
    ReceiverAddress string `json:"receiver_address"`
    Subject         string `json:"subject"`
    Data            []byte `json:"data"`
}
```

<a name="IssueTransactionResponse"></a>
## type [IssueTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L168-L171>)

IssueTransactionResponse is response to issued transaction.

```go
type IssueTransactionResponse struct {
    Ok  bool   `json:"ok"`
    Err string `json:"err"`
}
```

<a name="IssuedTransactionResponse"></a>
## type [IssuedTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L249-L253>)

IssuedTransactionResponse is a response of issued transactions.

```go
type IssuedTransactionResponse struct {
    Ok           bool                      `json:"ok"`
    Err          string                    `json:"err"`
    Transactions []transaction.Transaction `json:"transactions"`
}
```

<a name="ReadWalletPublicAddressResponse"></a>
## type [ReadWalletPublicAddressResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L409-L413>)

ReadWalletPublicAddressResponse is a response to read wallet public address.

```go
type ReadWalletPublicAddressResponse struct {
    Ok      bool   `json:"ok"`
    Err     string `json:"err"`
    Address string `json:"address"`
}
```

<a name="ReceivedTransactionResponse"></a>
## type [ReceivedTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L266-L270>)

ReceivedTransactionResponse is a response of issued transactions.

```go
type ReceivedTransactionResponse struct {
    Ok           bool                      `json:"ok"`
    Err          string                    `json:"err"`
    Transactions []transaction.Transaction `json:"transactions"`
}
```

<a name="RejectTransactionsRequest"></a>
## type [RejectTransactionsRequest](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L218-L220>)

RejectTransactionsRequest is a request to reject transactions.

```go
type RejectTransactionsRequest struct {
    Transactions []transaction.Transaction `json:"transactions"`
}
```

<a name="RejectTransactionsResponse"></a>
## type [RejectTransactionsResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L223-L227>)

RejectTransactionsResponse is response of rejecting transactions.

```go
type RejectTransactionsResponse struct {
    Ok         bool       `json:"ok"`
    Err        string     `json:"err"`
    TrxsHashes [][32]byte `json:"trxs_hashes"`
}
```

<a name="RejectedTransactionResponse"></a>
## type [RejectedTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/walletapi/walletapi.go#L283-L287>)

RejectedTransactionResponse is a response of rejected transactions.

```go
type RejectedTransactionResponse struct {
    Ok           bool                      `json:"ok"`
    Err          string                    `json:"err"`
    Transactions []transaction.Transaction `json:"transactions"`
}
```

# walletmiddleware

```go
import "github.com/bartossh/Computantis/walletmiddleware"
```

## Index

- [type Client](<#Client>)
  - [func NewClient\(apiRoot string, timeout time.Duration, fw transaction.Verifier, wrs WalletReadSaver, walletCreator NewSignValidatorCreator\) \*Client](<#NewClient>)
  - [func \(c \*Client\) Address\(\) \(string, error\)](<#Client.Address>)
  - [func \(c \*Client\) ConfirmTransaction\(trx \*transaction.Transaction\) error](<#Client.ConfirmTransaction>)
  - [func \(c \*Client\) CreateWebhook\(webHookURL string\) error](<#Client.CreateWebhook>)
  - [func \(c \*Client\) DataToSign\(\) \(server.DataToSignResponse, error\)](<#Client.DataToSign>)
  - [func \(c \*Client\) FlushWalletFromMemory\(\)](<#Client.FlushWalletFromMemory>)
  - [func \(c \*Client\) GenerateToken\(t time.Time\) \(token.Token, error\)](<#Client.GenerateToken>)
  - [func \(c \*Client\) NewWallet\(token string\) error](<#Client.NewWallet>)
  - [func \(c \*Client\) ProposeTransaction\(receiverAddr string, subject string, data \[\]byte\) error](<#Client.ProposeTransaction>)
  - [func \(c \*Client\) ReadApprovedTransactions\(offset, limit int\) \(\[\]transaction.Transaction, error\)](<#Client.ReadApprovedTransactions>)
  - [func \(c \*Client\) ReadIssuedTransactions\(\) \(\[\]transaction.Transaction, error\)](<#Client.ReadIssuedTransactions>)
  - [func \(c \*Client\) ReadRejectedTransactions\(offset, limit int\) \(\[\]transaction.Transaction, error\)](<#Client.ReadRejectedTransactions>)
  - [func \(c \*Client\) ReadWaitingTransactions\(\) \(\[\]transaction.Transaction, error\)](<#Client.ReadWaitingTransactions>)
  - [func \(c \*Client\) ReadWalletFromFile\(\) error](<#Client.ReadWalletFromFile>)
  - [func \(c \*Client\) RejectTransactions\(trxs \[\]transaction.Transaction\) \(\[\]\[32\]byte, error\)](<#Client.RejectTransactions>)
  - [func \(c \*Client\) SaveWalletToFile\(\) error](<#Client.SaveWalletToFile>)
  - [func \(c \*Client\) Sign\(d \[\]byte\) \(digest \[32\]byte, signature \[\]byte, err error\)](<#Client.Sign>)
  - [func \(c \*Client\) ValidateApiVersion\(\) error](<#Client.ValidateApiVersion>)
- [type NewSignValidatorCreator](<#NewSignValidatorCreator>)
- [type WalletReadSaver](<#WalletReadSaver>)


<a name="Client"></a>
## type [Client](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L35-L43>)

Client is a rest client for the API. It provides methods to communicate with the API server and is designed to serve as a easy way of building client applications that uses the REST API of the central node.

```go
type Client struct {
    // contains filtered or unexported fields
}
```

<a name="NewClient"></a>
### func [NewClient](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L46-L49>)

```go
func NewClient(apiRoot string, timeout time.Duration, fw transaction.Verifier, wrs WalletReadSaver, walletCreator NewSignValidatorCreator) *Client
```

NewClient creates a new rest client.

<a name="Client.Address"></a>
### func \(\*Client\) [Address](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L131>)

```go
func (c *Client) Address() (string, error)
```

Address reads the wallet address. Address is a string representation of wallet public key.

<a name="Client.ConfirmTransaction"></a>
### func \(\*Client\) [ConfirmTransaction](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L177>)

```go
func (c *Client) ConfirmTransaction(trx *transaction.Transaction) error
```

ConfirmTransaction confirms transaction by signing it with the wallet and then sending it to the API server.

<a name="Client.CreateWebhook"></a>
### func \(\*Client\) [CreateWebhook](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L447>)

```go
func (c *Client) CreateWebhook(webHookURL string) error
```



<a name="Client.DataToSign"></a>
### func \(\*Client\) [DataToSign](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L420>)

```go
func (c *Client) DataToSign() (server.DataToSignResponse, error)
```

DataToSign returns data to sign for the current wallet. Data to sign are randomly generated bytes by the server and stored in pair with the address. Signing this data is a proof that the signing public address is the owner of the wallet a making request.

<a name="Client.FlushWalletFromMemory"></a>
### func \(\*Client\) [FlushWalletFromMemory](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L495>)

```go
func (c *Client) FlushWalletFromMemory()
```

FlushWalletFromMemory flushes the wallet from the memory. Do it after you have saved the wallet to the file. It is recommended to use this just before logging out from the UI or closing the front end app that.

<a name="Client.GenerateToken"></a>
### func \(\*Client\) [GenerateToken](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L366>)

```go
func (c *Client) GenerateToken(t time.Time) (token.Token, error)
```

GenerateToken generates a token for the given time in the central node repository. It is only permitted to generate a token if wallet has admin permissions in the central node.

<a name="Client.NewWallet"></a>
### func \(\*Client\) [NewWallet](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L75>)

```go
func (c *Client) NewWallet(token string) error
```

NewWallet creates a new wallet and sends a request to the API server to validate the wallet.

<a name="Client.ProposeTransaction"></a>
### func \(\*Client\) [ProposeTransaction](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L144>)

```go
func (c *Client) ProposeTransaction(receiverAddr string, subject string, data []byte) error
```

ProposeTransaction sends a Transaction proposal to the API server for provided receiver address. Subject describes how to read the data from the transaction. For example, if the subject is "json", then the data can by decoded to map\[sting\]any, when subject "pdf" than it should be decoded by proper pdf decoder, when "csv" then it should be decoded by proper csv decoder. Client is not responsible for decoding the data, it is only responsible for sending the data to the API server.

<a name="Client.ReadApprovedTransactions"></a>
### func \(\*Client\) [ReadApprovedTransactions](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L333>)

```go
func (c *Client) ReadApprovedTransactions(offset, limit int) ([]transaction.Transaction, error)
```

ReadApprovedTransactions reads approved transactions belonging to current wallet from the API server. Method allows for paggination with offset and limit.

<a name="Client.ReadIssuedTransactions"></a>
### func \(\*Client\) [ReadIssuedTransactions](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L269>)

```go
func (c *Client) ReadIssuedTransactions() ([]transaction.Transaction, error)
```

ReadIssuedTransactions reads all issued transactions belonging to current wallet from the API server.

<a name="Client.ReadRejectedTransactions"></a>
### func \(\*Client\) [ReadRejectedTransactions](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L300>)

```go
func (c *Client) ReadRejectedTransactions(offset, limit int) ([]transaction.Transaction, error)
```

ReadRejectedTransactions reads rejected transactions belonging to current wallet from the API server. Method allows for paggination with offset and limit.

<a name="Client.ReadWaitingTransactions"></a>
### func \(\*Client\) [ReadWaitingTransactions](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L239>)

```go
func (c *Client) ReadWaitingTransactions() ([]transaction.Transaction, error)
```

ReadWaitingTransactions reads all waiting transactions belonging to current wallet from the API server.

<a name="Client.ReadWalletFromFile"></a>
### func \(\*Client\) [ReadWalletFromFile](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L407>)

```go
func (c *Client) ReadWalletFromFile() error
```

ReadWalletFromFile reads the wallet from the file in the path.

<a name="Client.RejectTransactions"></a>
### func \(\*Client\) [RejectTransactions](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L206>)

```go
func (c *Client) RejectTransactions(trxs []transaction.Transaction) ([][32]byte, error)
```

RejectTransactions rejects given transactions. Transaction will be rejected if the transaction receiver is a given wellet public address. Returns hashes of all the rejected transactions or error otherwise.

<a name="Client.SaveWalletToFile"></a>
### func \(\*Client\) [SaveWalletToFile](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L398>)

```go
func (c *Client) SaveWalletToFile() error
```

SaveWalletToFile saves the wallet to the file in the path.

<a name="Client.Sign"></a>
### func \(\*Client\) [Sign](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L439>)

```go
func (c *Client) Sign(d []byte) (digest [32]byte, signature []byte, err error)
```

Sign signs the given data with the wallet and returns digest and signature or error otherwise. This process creates a proof for the API server that requesting client is the owner of the wallet.

<a name="Client.ValidateApiVersion"></a>
### func \(\*Client\) [ValidateApiVersion](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L56>)

```go
func (c *Client) ValidateApiVersion() error
```

ValidateApiVersion makes a call to the API server and validates client and server API versions and header correctness. If API version not much it is returning an error as accessing the API server with different API version may lead to unexpected results.

<a name="NewSignValidatorCreator"></a>
## type [NewSignValidatorCreator](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L29>)

NewWalletCreator is a function that creates a new SignValidator.

```go
type NewSignValidatorCreator func() (wallet.Wallet, error)
```

<a name="WalletReadSaver"></a>
## type [WalletReadSaver](<https://github.com/bartossh/Computantis/blob/main/walletmiddleware/walletmiddleware.go#L23-L26>)

WalletReadSaver allows to read and save the wallet.

```go
type WalletReadSaver interface {
    ReadWallet() (wallet.Wallet, error)
    SaveWallet(w wallet.Wallet) error
}
```

# webhooks

```go
import "github.com/bartossh/Computantis/webhooks"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [type Hook](<#Hook>)
- [type NewTransactionMessage](<#NewTransactionMessage>)
- [type Service](<#Service>)
  - [func New\(l logger.Logger\) \*Service](<#New>)
  - [func \(s \*Service\) CreateWebhook\(trigger byte, publicAddress string, h Hook\) error](<#Service.CreateWebhook>)
  - [func \(s \*Service\) PostWebhookBlock\(blc \*block.Block\)](<#Service.PostWebhookBlock>)
  - [func \(s \*Service\) PostWebhookNewTransaction\(publicAddresses \[\]string\)](<#Service.PostWebhookNewTransaction>)
  - [func \(s \*Service\) RemoveWebhook\(trigger byte, publicAddress string, h Hook\) error](<#Service.RemoveWebhook>)
- [type WebHookNewBlockMessage](<#WebHookNewBlockMessage>)


## Constants

<a name="TriggerNewBlock"></a>

```go
const (
    TriggerNewBlock       byte = iota // TriggerNewBlock is the trigger for new block. It is triggered when a new block is forged.
    TriggerNewTransaction             // TriggerNewTransaction is a trigger for new transaction. It is triggered when a new transaction is received.
)
```

<a name="StateIssued"></a>

```go
const (
    StateIssued      byte = 0 // StateIssued is state of the transaction meaning it is only signed by the issuer.
    StateAcknowleged          // StateAcknowledged is a state ot the transaction meaning it is acknowledged and signed by the receiver.
)
```

## Variables

<a name="ErrorHookNotImplemented"></a>

```go
var ErrorHookNotImplemented = errors.New("hook not implemented")
```

<a name="Hook"></a>
## type [Hook](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L41-L44>)

Hook is the hook that is used to trigger the webhook.

```go
type Hook struct {
    URL   string `json:"address"` // URL is a url  of the webhook.
    Token string `json:"token"`   // Token is the token added to the webhook to verify that the message comes from the valid source.
}
```

<a name="NewTransactionMessage"></a>
## type [NewTransactionMessage](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L34-L38>)

NewTransactionMessage is the message send to the webhook url about new transaction for given wallet address.

```go
type NewTransactionMessage struct {
    Token string    `json:"token"`
    Time  time.Time `json:"time"`
    State byte      `json:"state"`
}
```

<a name="Service"></a>
## type [Service](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L49-L53>)

Service provide webhook service that is used to create, remove and update webhooks.

```go
type Service struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L56>)

```go
func New(l logger.Logger) *Service
```

New creates new instance of the webhook service.

<a name="Service.CreateWebhook"></a>
### func \(\*Service\) [CreateWebhook](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L65>)

```go
func (s *Service) CreateWebhook(trigger byte, publicAddress string, h Hook) error
```

CreateWebhook creates new webhook or or updates existing one for given trigger.

<a name="Service.PostWebhookBlock"></a>
### func \(\*Service\) [PostWebhookBlock](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L87>)

```go
func (s *Service) PostWebhookBlock(blc *block.Block)
```

PostWebhookBlock posts block to all webhooks that are subscribed to the new block trigger.

<a name="Service.PostWebhookNewTransaction"></a>
### func \(\*Service\) [PostWebhookNewTransaction](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L109>)

```go
func (s *Service) PostWebhookNewTransaction(publicAddresses []string)
```

PostWebhookNewTransaction posts information to the corresponding public address.

<a name="Service.RemoveWebhook"></a>
### func \(\*Service\) [RemoveWebhook](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L76>)

```go
func (s *Service) RemoveWebhook(trigger byte, publicAddress string, h Hook) error
```

RemoveWebhook removes webhook for given trigger and Hook URL.

<a name="WebHookNewBlockMessage"></a>
## type [WebHookNewBlockMessage](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L22-L26>)

WebHookNewBlockMessage is the message send to the webhook url about new forged block.

```go
type WebHookNewBlockMessage struct {
    Token string      `json:"token"` // Token given to the webhook by the webhooks creator to validate the message source.
    Block block.Block `json:"block"` // Block is the block that was mined.
    Valid bool        `json:"valid"` // Valid is the flag that indicates if the block is valid.
}
```

# zincaddapter

```go
import "github.com/bartossh/Computantis/zincaddapter"
```

## Index

- [Variables](<#variables>)
- [type Config](<#Config>)
- [type ZincClient](<#ZincClient>)
  - [func New\(cfg Config\) \(ZincClient, error\)](<#New>)
  - [func \(z \*ZincClient\) Write\(p \[\]byte\) \(n int, err error\)](<#ZincClient.Write>)


## Variables

<a name="ErrZincServerNotResponding"></a>

```go
var (
    ErrZincServerNotResponding = errors.New("zinc server not responding on given address")
    ErrZincServerWriteFailed   = errors.New("zinc server write failed")
)
```

<a name="Config"></a>
## type [Config](<https://github.com/bartossh/Computantis/blob/main/zincaddapter/zincaddapter.go#L24-L28>)

LoggerConfig contains configuration for logger back\-end

```go
type Config struct {
    Address string `yaml:"address"` // logger back-end server address
    Index   string `yaml:"index"`   // unique index per service to easy search for logs by the service
    Token   string `yaml:"token"`   // Authentication token i n format [ Basic some-auth-token-base64 ]
}
```

<a name="ZincClient"></a>
## type [ZincClient](<https://github.com/bartossh/Computantis/blob/main/zincaddapter/zincaddapter.go#L37-L41>)

ZincClient provides a client that sends logs to the zincsearch backend

```go
type ZincClient struct {
    // contains filtered or unexported fields
}
```

<a name="New"></a>
### func [New](<https://github.com/bartossh/Computantis/blob/main/zincaddapter/zincaddapter.go#L44>)

```go
func New(cfg Config) (ZincClient, error)
```

New creates a new ZincClient.

<a name="ZincClient.Write"></a>
### func \(\*ZincClient\) [Write](<https://github.com/bartossh/Computantis/blob/main/zincaddapter/zincaddapter.go#L52>)

```go
func (z *ZincClient) Write(p []byte) (n int, err error)
```

Write satisfies io.Writer abstraction.

# central

```go
import "github.com/bartossh/Computantis/cmd/central"
```

## Index



# client

```go
import "github.com/bartossh/Computantis/cmd/client"
```

## Index



# emulator

```go
import "github.com/bartossh/Computantis/cmd/emulator"
```

## Index



# generator

```go
import "github.com/bartossh/Computantis/cmd/generator"
```

## Index



# validator

```go
import "github.com/bartossh/Computantis/cmd/validator"
```

## Index



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
