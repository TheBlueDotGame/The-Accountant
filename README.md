# Computantis

[![CodeQL](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/github-code-scanning/codeql)
[![pages-build-deployment](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/bartossh/Computantis/actions/workflows/pages/pages-build-deployment)


The Computantis is a set of services that keeps track of transactions between wallets.
Transactions are not transferring any tokens between wallets but it might be the case if someone wants to use it this way. Just this set of services isn’t designed to track token exchange. Instead, transactions are entities holding data that the transaction issuer and transaction receiver agreed upon. Each wallet has its own independent history of transactions. There is a set of strict rules allowing for transactions to happen:

The central server is private to the corporation, government or agency. It is trusted by the above entity and participants. This solution isn’t proposing the distributed system of transactions as this is not the case. It is ensuring that the transaction is issued and signed and received and signed to confirm its validity. Blockchain keeps transactions history immutable so the validators can be sure that no one will corrupt the transactions. 
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


## Execute the server

0. Run database `docker compose up`.
1. Build the server `go build -o path/to/bin/central cmd/central/main.go`.
2. Create `server_settings.yaml` according to `server_settings_example.yaml` file in `path/to/bin/` folder.
3. Run `./path/to/bin/central`.

## Run for development

0. Run mongo database `docker compose -f docker-compose-mongo.yaml up -d` or postgresql `docker compose -f docker-compose-postgresql.yaml up -d`.
1. Create `server_settings.yaml` according to `server_settings_example.yaml` in the repo root folder.
2. Run `make run` or `go run cmd/central/main.go`.

## Stress test

Directory `stress/` contains central node REST API performance tests.
 - Testing performance on MacBook with M2 arm64 chip, 24GB RAM with central node, validator node and MongoDB running in docker container with 1CPU and 1GB RAM, 25 transactions per block, full cycle of creating 1000 transactions took 3.75 sec.
 - Testing performance on MacBook with M2 arm64 chip, 24GB RAM with central node, validator node and PostgreSQL running in docker container with 1CPU and 1GB RAM, 25 transactions per block, full cycle of creating 1000 transactions took 3.91 sec.
  - Testing performance on MacBook with M2 arm64 chip, 24GB RAM with central node, validator node and PostgreSQL running in docker container with 1CPU and 1GB RAM, 25 transactions per block, full cycle of creating 10000 transactions took 37.35 sec. This allows to fully process 267 transactions per second, which means: 267 times per second reding proposed transaction by issuer with proper validation, sending it to receiver, reding signed confirmation from receiver with proper validation, forging blocks by permanently adding transactions and sending blocks to the validator node which validates the forging process.

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

# GO Documentation

<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# address

```go
import "github.com/bartossh/Computantis/address"
```

## Index

- [type Address](<#type-address>)


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
- [type Helper](<#type-helper>)
  - [func New() Helper](<#func-new>)
  - [func (h Helper) Decrypt(key, data []byte) ([]byte, error)](<#func-helper-decrypt>)
  - [func (h Helper) Encrypt(key, data []byte) ([]byte, error)](<#func-helper-encrypt>)


## Variables

```go
var (
    ErrInvalidKeyLength   = errors.New("invalid key length, must be longer then 32 bytes")
    ErrCipherFailure      = errors.New("cipher creation failure")
    ErrGCMFailure         = errors.New("gcm creation failure")
    ErrRandomNonceFailure = errors.New("random nonce creation failure")
    ErrOpenDataFailure    = errors.New("open data failure, cannot decrypt data")
)
```

## type [Helper](<https://github.com/bartossh/Computantis/blob/main/aeswrapper/aeswrapper.go#L25>)

Helper wraps eas encryption and decryption. Uses Galois Counter Mode \(GCM\) for encryption and decryption.

```go
type Helper struct{}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/aeswrapper/aeswrapper.go#L28>)

```go
func New() Helper
```

Creates a new Helper.

### func \(Helper\) [Decrypt](<https://github.com/bartossh/Computantis/blob/main/aeswrapper/aeswrapper.go#L61>)

```go
func (h Helper) Decrypt(key, data []byte) ([]byte, error)
```

Decrypt decrypts data with key. Key must be at least 32 bytes long.

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

- [type Block](<#type-block>)
  - [func New(difficulty, next uint64, prevHash [32]byte, trxHashes [][32]byte) Block](<#func-new>)
  - [func (b *Block) Validate(trxHashes [][32]byte) bool](<#func-block-validate>)


## type [Block](<https://github.com/bartossh/Computantis/blob/main/block/block.go#L20-L29>)

Block holds block information. Block is a part of a blockchain assuring immutability of the data. Block mining difficulty may change if needed and is a part of a hash digest. Block ensures that transactions hashes are valid and match the transactions stored in the repository.

```go
type Block struct {
    ID         any        `json:"-"          bson:"_id"        db:"id"`
    Index      uint64     `json:"index"      bson:"index"      db:"index"`
    Timestamp  uint64     `json:"timestamp"  bson:"timestamp"  db:"timestamp"`
    Nonce      uint64     `json:"nonce"      bson:"nonce"      db:"nonce"`
    Difficulty uint64     `json:"difficulty" bson:"difficulty" db:"difficulty"`
    Hash       [32]byte   `json:"hash"       bson:"hash"       db:"hash"`
    PrevHash   [32]byte   `json:"prev_hash"  bson:"prev_hash"  db:"prev_hash"`
    TrxHashes  [][32]byte `json:"trx_hashes" bson:"trx_hashes" db:"trx_hashes"`
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/block/block.go#L35>)

```go
func New(difficulty, next uint64, prevHash [32]byte, trxHashes [][32]byte) Block
```

New creates a new Block hashing it with given difficulty. Higher difficulty requires more computations to happen to find possible target hash. Difficulty is stored inside the Block and is a part of a hashed data. Transactions hashes are prehashed before calculating the Block hash with merkle tree.

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
- [func GenesisBlock(ctx context.Context, rw BlockReadWriter) error](<#func-genesisblock>)
- [type BlockReadWriter](<#type-blockreadwriter>)
- [type BlockReader](<#type-blockreader>)
- [type BlockWriter](<#type-blockwriter>)
- [type Blockchain](<#type-blockchain>)
  - [func New(ctx context.Context, rw BlockReadWriter) (*Blockchain, error)](<#func-new>)
  - [func (c *Blockchain) LastBlockHashIndex() ([32]byte, uint64)](<#func-blockchain-lastblockhashindex>)
  - [func (c *Blockchain) ReadBlocksFromIndex(ctx context.Context, idx uint64) ([]block.Block, error)](<#func-blockchain-readblocksfromindex>)
  - [func (c *Blockchain) ReadLastNBlocks(ctx context.Context, n int) ([]block.Block, error)](<#func-blockchain-readlastnblocks>)
  - [func (c *Blockchain) WriteBlock(ctx context.Context, block block.Block) error](<#func-blockchain-writeblock>)


## Variables

```go
var (
    ErrBlockNotFound        = errors.New("block not found")
    ErrInvalidBlockPrevHash = errors.New("block prev hash is invalid")
    ErrInvalidBlockHash     = errors.New("block hash is invalid")
    ErrInvalidBlockIndex    = errors.New("block index is invalid")
)
```

## func [GenesisBlock](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L49>)

```go
func GenesisBlock(ctx context.Context, rw BlockReadWriter) error
```

GenesisBlock creates a genesis block. It is a first block in the blockchain. The genesis block is created only if there is no other block in the repository. Otherwise returning an error.

## type [BlockReadWriter](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L31-L34>)

BlockReadWriter provides read and write access to the blockchain repository.

```go
type BlockReadWriter interface {
    BlockReader
    BlockWriter
}
```

## type [BlockReader](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L20-L23>)

BlockReader provides read access to the blockchain repository.

```go
type BlockReader interface {
    LastBlock(ctx context.Context) (block.Block, error)
    ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
}
```

## type [BlockWriter](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L26-L28>)

BlockWriter provides write access to the blockchain repository.

```go
type BlockWriter interface {
    WriteBlock(ctx context.Context, block block.Block) error
}
```

## type [Blockchain](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L39-L44>)

Blockchain keeps track of the blocks creating immutable chain of data. Blockchain is stored in repository as separate blocks that relates to each other based on the hash of the previous block.

```go
type Blockchain struct {
    // contains filtered or unexported fields
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L62>)

```go
func New(ctx context.Context, rw BlockReadWriter) (*Blockchain, error)
```

New creates a new Blockchain that has access to the blockchain stored in the repository. The access to the repository is injected via BlockReadWriter interface. You can use any implementation of repository that implements BlockReadWriter interface and ensures unique indexing for Block Hash, PrevHash and Index.

### func \(\*Blockchain\) [LastBlockHashIndex](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L77>)

```go
func (c *Blockchain) LastBlockHashIndex() ([32]byte, uint64)
```

LastBlockHashIndex returns last block hash and index.

### func \(\*Blockchain\) [ReadBlocksFromIndex](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L104>)

```go
func (c *Blockchain) ReadBlocksFromIndex(ctx context.Context, idx uint64) ([]block.Block, error)
```

ReadBlocksFromIndex reads all blocks from given index till the current block in consecutive order.

### func \(\*Blockchain\) [ReadLastNBlocks](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L84>)

```go
func (c *Blockchain) ReadLastNBlocks(ctx context.Context, n int) ([]block.Block, error)
```

ReadLastNBlocks reads the last n blocks in reverse consecutive order.

### func \(\*Blockchain\) [WriteBlock](<https://github.com/bartossh/Computantis/blob/main/blockchain/blockchain.go#L128>)

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
- [type AddressChecker](<#type-addresschecker>)
- [type BlockFindWriter](<#type-blockfindwriter>)
- [type BlockReadWriter](<#type-blockreadwriter>)
- [type BlockReader](<#type-blockreader>)
- [type BlockSubscription](<#type-blocksubscription>)
- [type BlockWriter](<#type-blockwriter>)
- [type Config](<#type-config>)
  - [func (c Config) Validate() error](<#func-config-validate>)
- [type Ledger](<#type-ledger>)
  - [func New(config Config, bc BlockReadWriter, tx TrxWriteReadMover, ac AddressChecker, vr SignatureVerifier, tf BlockFindWriter, log logger.Logger, blcSub BlockSubscription) (*Ledger, error)](<#func-new>)
  - [func (l *Ledger) Run(ctx context.Context)](<#func-ledger-run>)
  - [func (l *Ledger) VerifySignature(message, signature []byte, hash [32]byte, address string) error](<#func-ledger-verifysignature>)
  - [func (l *Ledger) WriteCandidateTransaction(ctx context.Context, trx *transaction.Transaction) error](<#func-ledger-writecandidatetransaction>)
  - [func (l *Ledger) WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error](<#func-ledger-writeissuersignedtransactionforreceiver>)
- [type SignatureVerifier](<#type-signatureverifier>)
- [type TrxWriteReadMover](<#type-trxwritereadmover>)


## Variables

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

## type [AddressChecker](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L67-L69>)

AddressChecker provides address existence check method. If you use other repository than addresses repository, you can implement this interface but address should be uniquely indexed in your repository implementation.

```go
type AddressChecker interface {
    CheckAddressExists(ctx context.Context, address string) (bool, error)
}
```

## type [BlockFindWriter](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L77-L80>)

BlockFindWriter provides block find and write method.

```go
type BlockFindWriter interface {
    WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
    FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
}
```

## type [BlockReadWriter](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L59-L62>)

BlockReadWriter provides block read and write methods.

```go
type BlockReadWriter interface {
    BlockReader
    BlockWriter
}
```

## type [BlockReader](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L49-L51>)

BlockReader provides block read methods.

```go
type BlockReader interface {
    LastBlockHashIndex() ([32]byte, uint64)
}
```

## type [BlockSubscription](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L85-L87>)

BlockSubscription provides block publishing method. It uses reactive package. It you are using your own implementation of reactive package take care of Publish method to be non\-blocking.

```go
type BlockSubscription interface {
    Publish(block.Block)
}
```

## type [BlockWriter](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L54-L56>)

BlockWriter provides block write methods.

```go
type BlockWriter interface {
    WriteBlock(ctx context.Context, block block.Block) error
}
```

## type [Config](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L90-L94>)

Config is a configuration of the Ledger.

```go
type Config struct {
    Difficulty            uint64 `json:"difficulty"              bson:"difficulty"              yaml:"difficulty"`
    BlockWriteTimestamp   uint64 `json:"block_write_timestamp"   bson:"block_write_timestamp"   yaml:"block_write_timestamp"`
    BlockTransactionsSize int    `json:"block_transactions_size" bson:"block_transactions_size" yaml:"block_transactions_size"`
}
```

### func \(Config\) [Validate](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L97>)

```go
func (c Config) Validate() error
```

Validate validates the Ledger configuration.

## type [Ledger](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L117-L128>)

Ledger is a collection of ledger functionality to perform bookkeeping. It performs all the actions on the transactions and blockchain. Ladger seals all the transaction actions in the blockchain.

```go
type Ledger struct {
    // contains filtered or unexported fields
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L131-L140>)

```go
func New(config Config, bc BlockReadWriter, tx TrxWriteReadMover, ac AddressChecker, vr SignatureVerifier, tf BlockFindWriter, log logger.Logger, blcSub BlockSubscription) (*Ledger, error)
```

New creates new Ledger if config is valid or returns error otherwise.

### func \(\*Ledger\) [Run](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L161>)

```go
func (l *Ledger) Run(ctx context.Context)
```

Run runs the Ladger engine that writes blocks to the blockchain repository. Run starts a goroutine and can be stopped by cancelling the context. It is non\-blocking and concurrent safe.

### func \(\*Ledger\) [VerifySignature](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L229>)

```go
func (l *Ledger) VerifySignature(message, signature []byte, hash [32]byte, address string) error
```

VerifySignature verifies signature of the message.

### func \(\*Ledger\) [WriteCandidateTransaction](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L211>)

```go
func (l *Ledger) WriteCandidateTransaction(ctx context.Context, trx *transaction.Transaction) error
```

WriteCandidateTransaction validates and writes a transaction to the repository. Transaction is not yet a part of the blockchain at this point. Ladger will perform all the necessary checks and validations before writing it to the repository. The candidate needs to be signed by the receiver later in the process  to be placed as a candidate in the blockchain.

### func \(\*Ledger\) [WriteIssuerSignedTransactionForReceiver](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L191-L195>)

```go
func (l *Ledger) WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
```

WriteIssuerSignedTransactionForReceiver validates issuer signature and writes a transaction to the repository for receiver.

## type [SignatureVerifier](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L72-L74>)

SignatureVerifier provides signature verification method.

```go
type SignatureVerifier interface {
    Verify(message, signature []byte, hash [32]byte, address string) error
}
```

## type [TrxWriteReadMover](<https://github.com/bartossh/Computantis/blob/main/bookkeeping/bookkeeping.go#L38-L46>)

TrxWriteReadMover provides transactions write, read and move methods. It allows to access temporary, permanent and awaiting transactions.

```go
type TrxWriteReadMover interface {
    WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
    WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
    MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
    RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error
    ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error)
}
```

# client

```go
import "github.com/bartossh/Computantis/client"
```

## Index

- [Variables](<#variables>)
- [type Client](<#type-client>)
  - [func NewClient(apiRoot string, timeout time.Duration, fw transaction.Verifier, wrs WalletReadSaver, walletCreator NewSignValidatorCreator) *Client](<#func-newclient>)
  - [func (c *Client) Address() (string, error)](<#func-client-address>)
  - [func (c *Client) ConfirmTransaction(trx *transaction.Transaction) error](<#func-client-confirmtransaction>)
  - [func (c *Client) DataToSign(address string) (server.DataToSignResponse, error)](<#func-client-datatosign>)
  - [func (c *Client) FlushWalletFromMemory()](<#func-client-flushwalletfrommemory>)
  - [func (c *Client) NewWallet(token string) error](<#func-client-newwallet>)
  - [func (c *Client) PostWebhookBlock(url string, token string, block *block.Block) error](<#func-client-postwebhookblock>)
  - [func (c *Client) ProposeTransaction(receiverAddr string, subject string, data []byte) error](<#func-client-proposetransaction>)
  - [func (c *Client) ReadIssuedTransactions() ([]transaction.Transaction, error)](<#func-client-readissuedtransactions>)
  - [func (c *Client) ReadWaitingTransactions() ([]transaction.Transaction, error)](<#func-client-readwaitingtransactions>)
  - [func (c *Client) ReadWalletFromFile(passwd, path string) error](<#func-client-readwalletfromfile>)
  - [func (c *Client) SaveWalletToFile() error](<#func-client-savewallettofile>)
  - [func (c *Client) Sign(d []byte) (digest [32]byte, signature []byte, err error)](<#func-client-sign>)
  - [func (c *Client) ValidateApiVersion() error](<#func-client-validateapiversion>)
- [type NewSignValidatorCreator](<#type-newsignvalidatorcreator>)
- [type WalletReadSaver](<#type-walletreadsaver>)


## Variables

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

## type [Client](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L49-L57>)

Client is a rest client for the API. It provides methods to communicate with the API server and is designed to serve as a easy way of building client applications that uses the REST API of the central node.

```go
type Client struct {
    // contains filtered or unexported fields
}
```

### func [NewClient](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L60-L63>)

```go
func NewClient(apiRoot string, timeout time.Duration, fw transaction.Verifier, wrs WalletReadSaver, walletCreator NewSignValidatorCreator) *Client
```

NewClient creates a new rest client.

### func \(\*Client\) [Address](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L140>)

```go
func (c *Client) Address() (string, error)
```

Address reads the wallet address. Address is a string representation of wallet public key.

### func \(\*Client\) [ConfirmTransaction](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L186>)

```go
func (c *Client) ConfirmTransaction(trx *transaction.Transaction) error
```

ConfirmTransaction confirms transaction by signing it with the wallet and then sending it to the API server.

### func \(\*Client\) [DataToSign](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L296>)

```go
func (c *Client) DataToSign(address string) (server.DataToSignResponse, error)
```

DataToSign returns data to sign for the given address. Data to sign are randomly generated bytes by the server and stored in pair with the address. Signing this data is a proof that the signing public address is the owner of the wallet a making request.

### func \(\*Client\) [FlushWalletFromMemory](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L332>)

```go
func (c *Client) FlushWalletFromMemory()
```

FlushWalletFromMemory flushes the wallet from the memory. Do it after you have saved the wallet to the file. It is recommended to use this just before logging out from the UI or closing the front end app that.

### func \(\*Client\) [NewWallet](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L88>)

```go
func (c *Client) NewWallet(token string) error
```

NewWallet creates a new wallet and sends a request to the API server to validate the wallet.

### func \(\*Client\) [PostWebhookBlock](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L319>)

```go
func (c *Client) PostWebhookBlock(url string, token string, block *block.Block) error
```

PostWebhookBlock posts validator.WebHookNewBlockMessage to given url.

### func \(\*Client\) [ProposeTransaction](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L153>)

```go
func (c *Client) ProposeTransaction(receiverAddr string, subject string, data []byte) error
```

ProposeTransaction sends a Transaction proposal to the API server for provided receiver address. Subject describes how to read the data from the transaction. For example, if the subject is "json", then the data can by decoded to map\[sting\]any, when subject "pdf" than it should be decoded by proper pdf decoder, when "csv" then it should be decoded by proper csv decoder. Client is not responsible for decoding the data, it is only responsible for sending the data to the API server.

### func \(\*Client\) [ReadIssuedTransactions](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L244>)

```go
func (c *Client) ReadIssuedTransactions() ([]transaction.Transaction, error)
```

ReadIssuedTransactions reads all issued transactions belonging to current wallet from the API server.

### func \(\*Client\) [ReadWaitingTransactions](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L213>)

```go
func (c *Client) ReadWaitingTransactions() ([]transaction.Transaction, error)
```

ReadWaitingTransactions reads all waiting transactions belonging to current wallet from the API server.

### func \(\*Client\) [ReadWalletFromFile](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L283>)

```go
func (c *Client) ReadWalletFromFile(passwd, path string) error
```

ReadWalletFromFile reads the wallet from the file in the path.

### func \(\*Client\) [SaveWalletToFile](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L274>)

```go
func (c *Client) SaveWalletToFile() error
```

SaveWalletToFile saves the wallet to the file in the path.

### func \(\*Client\) [Sign](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L310>)

```go
func (c *Client) Sign(d []byte) (digest [32]byte, signature []byte, err error)
```

Sign signs the given data with the wallet and returns digest and signature or error otherwise. This process creates a proof for the API server that requesting client is the owner of the wallet.

### func \(\*Client\) [ValidateApiVersion](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L70>)

```go
func (c *Client) ValidateApiVersion() error
```

ValidateApiVersion makes a call to the API server and validates client and server API versions and header correctness. If API version not much it is returning an error as accessing the API server with different API version may lead to unexpected results.

## type [NewSignValidatorCreator](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L43>)

NewWalletCreator is a function that creates a new SignValidator.

```go
type NewSignValidatorCreator func() (wallet.Wallet, error)
```

## type [WalletReadSaver](<https://github.com/bartossh/Computantis/blob/main/client/client.go#L37-L40>)

WalletReadSaver allows to read and save the wallet.

```go
type WalletReadSaver interface {
    ReadWallet() (wallet.Wallet, error)
    SaveWallet(w wallet.Wallet) error
}
```

# configuration

```go
import "github.com/bartossh/Computantis/configuration"
```

## Index

- [type Configuration](<#type-configuration>)
  - [func Read(path string) (Configuration, error)](<#func-read>)


## type [Configuration](<https://github.com/bartossh/Computantis/blob/main/configuration/configuration.go#L18-L25>)

Configuration is the main configuration of the application that corresponds to the \*.yaml file that holds the configuration.

```go
type Configuration struct {
    Bookkeeper   bookkeeping.Config    `yaml:"bookkeeper"`
    Server       server.Config         `yaml:"server"`
    Database     repohelper.DBConfig   `yaml:"database"`
    DataProvider dataprovider.Config   `yaml:"data_provider"`
    Validator    validator.Config      `yaml:"validator"`
    FileOperator fileoperations.Config `yaml:"file_operator"`
}
```

### func [Read](<https://github.com/bartossh/Computantis/blob/main/configuration/configuration.go#L28>)

```go
func Read(path string) (Configuration, error)
```

Read reads the configuration from the file and returns the Configuration with set fields according to the yaml setup.

# dataprovider

```go
import "github.com/bartossh/Computantis/dataprovider"
```

## Index

- [type Cache](<#type-cache>)
  - [func New(ctx context.Context, cfg Config) *Cache](<#func-new>)
  - [func (c *Cache) ProvideData(address string) []byte](<#func-cache-providedata>)
  - [func (c *Cache) ValidateData(address string, data []byte) bool](<#func-cache-validatedata>)
- [type Config](<#type-config>)


## type [Cache](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L22-L26>)

Cache is a simple in\-memory cache for storing generated data.

```go
type Cache struct {
    // contains filtered or unexported fields
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L29>)

```go
func New(ctx context.Context, cfg Config) *Cache
```

New creates new Cache and runs the cleaner.

### func \(\*Cache\) [ProvideData](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L65>)

```go
func (c *Cache) ProvideData(address string) []byte
```

ProvideData generates data and stores it referring to given address.

### func \(\*Cache\) [ValidateData](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L80>)

```go
func (c *Cache) ValidateData(address string, data []byte) bool
```

ValidateData checks if data is stored for given address and is not expired.

## type [Config](<https://github.com/bartossh/Computantis/blob/main/dataprovider/dataprovider.go#L12-L14>)

Config holds configuration for Cache.

```go
type Config struct {
    Longevity uint64 `yaml:"longevity"` // Data longevity in seconds.
}
```

# fileoperations

```go
import "github.com/bartossh/Computantis/fileoperations"
```

## Index

- [type Config](<#type-config>)
- [type Helper](<#type-helper>)
  - [func New(cfg Config, s Sealer) Helper](<#func-new>)
  - [func (h Helper) ReadWallet() (wallet.Wallet, error)](<#func-helper-readwallet>)
  - [func (h Helper) SaveWallet(w wallet.Wallet) error](<#func-helper-savewallet>)
- [type Sealer](<#type-sealer>)


## type [Config](<https://github.com/bartossh/Computantis/blob/main/fileoperations/fileoperations.go#L4-L7>)

Config holds configuration of the file operator Helper.

```go
type Config struct {
    WalletPath   string `yaml:"wallet_path"`   // wallet path to the wallet file
    WalletPasswd string `yaml:"wallet_passwd"` // wallet password to the wallet file in hex format
}
```

## type [Helper](<https://github.com/bartossh/Computantis/blob/main/fileoperations/fileoperations.go#L10-L13>)

Helper holds all file operation methods.

```go
type Helper struct {
    // contains filtered or unexported fields
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/fileoperations/fileoperations.go#L16>)

```go
func New(cfg Config, s Sealer) Helper
```

New creates new Helper.

### func \(Helper\) [ReadWallet](<https://github.com/bartossh/Computantis/blob/main/fileoperations/wallet.go#L16>)

```go
func (h Helper) ReadWallet() (wallet.Wallet, error)
```

RereadWallet reads wallet from the file.

### func \(Helper\) [SaveWallet](<https://github.com/bartossh/Computantis/blob/main/fileoperations/wallet.go#L40>)

```go
func (h Helper) SaveWallet(w wallet.Wallet) error
```

SaveWallet saves wallet to the file.

## type [Sealer](<https://github.com/bartossh/Computantis/blob/main/fileoperations/wallet.go#L10-L13>)

```go
type Sealer interface {
    Encrypt(key, data []byte) ([]byte, error)
    Decrypt(key, data []byte) ([]byte, error)
}
```

# logger

```go
import "github.com/bartossh/Computantis/logger"
```

## Index

- [type Log](<#type-log>)
- [type Logger](<#type-logger>)


## type [Log](<https://github.com/bartossh/Computantis/blob/main/logger/logger.go#L8-L13>)

Log is log marshaled and written in to the io.Writer of the helper implementing Logger abstraction.

```go
type Log struct {
    ID        any       `json:"_id"        bson:"_id"        db:"id"`
    Level     string    `jon:"level"       bson:"level"      db:"level"`
    Msg       string    `json:"msg"        bson:"msg"        db:"msg"`
    CreatedAt time.Time `json:"created_at" bson:"created_at" db:"created_at"`
}
```

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

- [type Helper](<#type-helper>)
  - [func New(callOnWriteLogErr, callOnFatal func(error), writers ...io.Writer) Helper](<#func-new>)
  - [func (h Helper) Debug(msg string)](<#func-helper-debug>)
  - [func (h Helper) Error(msg string)](<#func-helper-error>)
  - [func (h Helper) Fatal(msg string)](<#func-helper-fatal>)
  - [func (h Helper) Info(msg string)](<#func-helper-info>)
  - [func (h Helper) Warn(msg string)](<#func-helper-warn>)


## type [Helper](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L16-L20>)

Helper helps with writing logs to io.Writers. Helper implements logger.Logger interface. Writing is done concurrently with out blocking the current thread.

```go
type Helper struct {
    // contains filtered or unexported fields
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L23>)

```go
func New(callOnWriteLogErr, callOnFatal func(error), writers ...io.Writer) Helper
```

New creates new Helper.

### func \(Helper\) [Debug](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L28>)

```go
func (h Helper) Debug(msg string)
```

Debug writes debug log.

### func \(Helper\) [Error](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L61>)

```go
func (h Helper) Error(msg string)
```

Error writes error log.

### func \(Helper\) [Fatal](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L72>)

```go
func (h Helper) Fatal(msg string)
```

Fatal writes fatal log.

### func \(Helper\) [Info](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L39>)

```go
func (h Helper) Info(msg string)
```

Info writes info log.

### func \(Helper\) [Warn](<https://github.com/bartossh/Computantis/blob/main/logging/logging.go#L50>)

```go
func (h Helper) Warn(msg string)
```

Warn writes warning log.

# reactive

```go
import "github.com/bartossh/Computantis/reactive"
```

## Index

- [type Observable](<#type-observable>)
  - [func New[T any](size int) *Observable[T]](<#func-new>)
  - [func (o *Observable[T]) Publish(v T)](<#func-observablet-publish>)
  - [func (o *Observable[T]) Subscribe() *subscriber[T]](<#func-observablet-subscribe>)


## type [Observable](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L25-L29>)

Observable creates a container for subscribers. This works in single producer multiple consumer pattern.

```go
type Observable[T any] struct {
    // contains filtered or unexported fields
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L33>)

```go
func New[T any](size int) *Observable[T]
```

New creates Observable container that holds channels for all subscribers. size is the buffer size of each channel.

### func \(\*Observable\[T\]\) [Publish](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L54>)

```go
func (o *Observable[T]) Publish(v T)
```

Publish publishes value to all subscribers.

### func \(\*Observable\[T\]\) [Subscribe](<https://github.com/bartossh/Computantis/blob/main/reactive/reactive.go#L42>)

```go
func (o *Observable[T]) Subscribe() *subscriber[T]
```

Subscribe subscribes to the container.

# repohelper

```go
import "github.com/bartossh/Computantis/repohelper"
```

## Index

- [Variables](<#variables>)
- [type AddressWriteFindChecker](<#type-addresswritefindchecker>)
- [type BlockReadWriter](<#type-blockreadwriter>)
- [type ConnectionCloser](<#type-connectioncloser>)
- [type DBConfig](<#type-dbconfig>)
  - [func (cfg DBConfig) Connect(ctx context.Context) (RepositoryProvider, error)](<#func-dbconfig-connect>)
- [type Migrator](<#type-migrator>)
- [type RepositoryProvider](<#type-repositoryprovider>)
- [type TokenWriteCheckInvalidator](<#type-tokenwritecheckinvalidator>)
- [type TransactionOperator](<#type-transactionoperator>)
- [type ValidatorStatusReader](<#type-validatorstatusreader>)


## Variables

```go
var (
    ErrDatabaseNotSupported = fmt.Errorf("database not supported")
)
```

## type [AddressWriteFindChecker](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L21-L25>)

AddressWriteFindChecker abstracts address operations.

```go
type AddressWriteFindChecker interface {
    WriteAddress(ctx context.Context, addr string) error
    CheckAddressExists(ctx context.Context, addr string) (bool, error)
    FindAddress(ctx context.Context, search string, limit int) ([]string, error)
}
```

## type [BlockReadWriter](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L28-L32>)

BlockReadWriter abstracts block operations.

```go
type BlockReadWriter interface {
    LastBlock(ctx context.Context) (block.Block, error)
    ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
    WriteBlock(ctx context.Context, block block.Block) error
}
```

## type [ConnectionCloser](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L66-L68>)

ConnectionCloser abstracts connection closing operations.

```go
type ConnectionCloser interface {
    Disconnect(ctx context.Context) error
}
```

## type [DBConfig](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L83-L88>)

Config contains configuration for the database.

```go
type DBConfig struct {
    ConnStr      string `yaml:"conn_str"`         // ConnStr is the connection string to the database.
    DatabaseName string `yaml:"database_name"`    // DatabaseName is the name of the database.
    Token        string `yaml:"token"`            // Token is the token that is used to confirm api clients access.
    TokenExpire  int64  `yaml:"token_expiration"` // TokenExpire is the number of seconds after which token expires.
}
```

### func \(DBConfig\) [Connect](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L91>)

```go
func (cfg DBConfig) Connect(ctx context.Context) (RepositoryProvider, error)
```

Connect connects to the proper database and returns that connection.

## type [Migrator](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L35-L37>)

MigrationRunner abstracts migration operations.

```go
type Migrator interface {
    RunMigration(ctx context.Context) error
}
```

## type [RepositoryProvider](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L71-L80>)

RepositoryProvider is an interface that ensures that all required methods to run computantis are implemented.

```go
type RepositoryProvider interface {
    AddressWriteFindChecker
    BlockReadWriter
    io.Writer
    Migrator
    TokenWriteCheckInvalidator
    TransactionOperator
    ValidatorStatusReader
    ConnectionCloser
}
```

## type [TokenWriteCheckInvalidator](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L40-L44>)

TokenWriteCheckInvalidator abstracts token operations.

```go
type TokenWriteCheckInvalidator interface {
    CheckToken(ctx context.Context, tkn string) (bool, error)
    WriteToken(ctx context.Context, tkn string, expirationDate int64) error
    InvalidateToken(ctx context.Context, token string) error
}
```

## type [TransactionOperator](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L47-L57>)

TransactionOperator abstracts transaction operations.

```go
type TransactionOperator interface {
    WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
    FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
    WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
    RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error
    WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
    ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
    MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
    ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error)
}
```

## type [ValidatorStatusReader](<https://github.com/bartossh/Computantis/blob/main/repohelper/repohelper.go#L60-L63>)

ValidatorStatusReader abstracts validator status operations.

```go
type ValidatorStatusReader interface {
    ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error)
    WriteValidatorStatus(ctx context.Context, vs *validator.Status) error
}
```

# repomongo

```go
import "github.com/bartossh/Computantis/repomongo"
```

## Index

- [type DataBase](<#type-database>)
  - [func Connect(ctx context.Context, conn, database string) (*DataBase, error)](<#func-connect>)
  - [func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error)](<#func-database-checkaddressexists>)
  - [func (db DataBase) CheckToken(ctx context.Context, tkn string) (bool, error)](<#func-database-checktoken>)
  - [func (c DataBase) Disconnect(ctx context.Context) error](<#func-database-disconnect>)
  - [func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error)](<#func-database-findaddress>)
  - [func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)](<#func-database-findtransactioninblockhash>)
  - [func (db DataBase) InvalidateToken(ctx context.Context, token string) error](<#func-database-invalidatetoken>)
  - [func (db DataBase) LastBlock(ctx context.Context) (block.Block, error)](<#func-database-lastblock>)
  - [func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error](<#func-database-movetransactionsfromtemporarytopermanent>)
  - [func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)](<#func-database-readawaitingtransactionsbyissuer>)
  - [func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)](<#func-database-readawaitingtransactionsbyreceiver>)
  - [func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)](<#func-database-readblockbyhash>)
  - [func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error)](<#func-database-readlastnvalidatorstatuses>)
  - [func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error)](<#func-database-readtemporarytransactions>)
  - [func (db DataBase) RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error](<#func-database-removeawaitingtransaction>)
  - [func (c DataBase) RunMigration(ctx context.Context) error](<#func-database-runmigration>)
  - [func (db DataBase) Write(p []byte) (n int, err error)](<#func-database-write>)
  - [func (db DataBase) WriteAddress(ctx context.Context, addr string) error](<#func-database-writeaddress>)
  - [func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error](<#func-database-writeblock>)
  - [func (db DataBase) WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error](<#func-database-writeissuersignedtransactionforreceiver>)
  - [func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error](<#func-database-writetemporarytransaction>)
  - [func (db DataBase) WriteToken(ctx context.Context, tkn string, expirationDate int64) error](<#func-database-writetoken>)
  - [func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error](<#func-database-writetransactionsinblock>)
  - [func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *validator.Status) error](<#func-database-writevalidatorstatus>)
- [type Migration](<#type-migration>)


## type [DataBase](<https://github.com/bartossh/Computantis/blob/main/repomongo/mongorepo.go#L26-L28>)

Database provides database access for read, write and delete of repository entities.

```go
type DataBase struct {
    // contains filtered or unexported fields
}
```

### func [Connect](<https://github.com/bartossh/Computantis/blob/main/repomongo/mongorepo.go#L31>)

```go
func Connect(ctx context.Context, conn, database string) (*DataBase, error)
```

Connect creates new connection to the repository and returns pointer to the DataBase.

### func \(DataBase\) [CheckAddressExists](<https://github.com/bartossh/Computantis/blob/main/repomongo/address.go#L32>)

```go
func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error)
```

CheckAddressExists checks if address exists in the database. Returns true if exists and error if database error different from ErrNoDocuments.

### func \(DataBase\) [CheckToken](<https://github.com/bartossh/Computantis/blob/main/repomongo/token.go#L14>)

```go
func (db DataBase) CheckToken(ctx context.Context, tkn string) (bool, error)
```

CheckToken checks if token exists in the database is valid and didn't expire.

### func \(DataBase\) [Disconnect](<https://github.com/bartossh/Computantis/blob/main/repomongo/mongorepo.go#L47>)

```go
func (c DataBase) Disconnect(ctx context.Context) error
```

Disconnect disconnects user from database

### func \(DataBase\) [FindAddress](<https://github.com/bartossh/Computantis/blob/main/repomongo/search.go#L38>)

```go
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error)
```

FindAddress looks for matching address in the addresses repository and returns limited slice of matching addresses. If limit is set to 0 or above the 1000 which is maximum then search is limited to 1000.

### func \(DataBase\) [FindTransactionInBlockHash](<https://github.com/bartossh/Computantis/blob/main/repomongo/search.go#L28>)

```go
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
```

FindTransactionInBlockHash finds Block hash in to which Transaction with given hash was added.

### func \(DataBase\) [InvalidateToken](<https://github.com/bartossh/Computantis/blob/main/repomongo/token.go#L46>)

```go
func (db DataBase) InvalidateToken(ctx context.Context, token string) error
```

InvalidateToken invalidates token.

### func \(DataBase\) [LastBlock](<https://github.com/bartossh/Computantis/blob/main/repomongo/block.go#L14>)

```go
func (db DataBase) LastBlock(ctx context.Context) (block.Block, error)
```

LastBlock returns last block from the database.

### func \(DataBase\) [MoveTransactionsFromTemporaryToPermanent](<https://github.com/bartossh/Computantis/blob/main/repomongo/transaction.go#L82>)

```go
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
```

MoveTransactionsFromTemporaryToPermanent moves transactions from temporary storage to permanent.

### func \(DataBase\) [ReadAwaitingTransactionsByIssuer](<https://github.com/bartossh/Computantis/blob/main/repomongo/transaction.go#L63>)

```go
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
```

ReadAwaitingTransactionsByReceiver reads all transactions paired with given issuer address.

### func \(DataBase\) [ReadAwaitingTransactionsByReceiver](<https://github.com/bartossh/Computantis/blob/main/repomongo/transaction.go#L44>)

```go
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
```

ReadAwaitingTransactionsByReceiver reads all transactions paired with given receiver address.

### func \(DataBase\) [ReadBlockByHash](<https://github.com/bartossh/Computantis/blob/main/repomongo/block.go#L36>)

```go
func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
```

ReadBlockByHash returns block with given hash.

### func \(DataBase\) [ReadLastNValidatorStatuses](<https://github.com/bartossh/Computantis/blob/main/repomongo/validator.go#L18>)

```go
func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error)
```

ReadLastNValidatorStatuses reads last validator statuses from the database.

### func \(DataBase\) [ReadTemporaryTransactions](<https://github.com/bartossh/Computantis/blob/main/repomongo/transaction.go#L113>)

```go
func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error)
```

ReadTemporaryTransactions reads transactions from the temporary storage.

### func \(DataBase\) [RemoveAwaitingTransaction](<https://github.com/bartossh/Computantis/blob/main/repomongo/transaction.go#L21>)

```go
func (db DataBase) RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error
```

RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.

### func \(DataBase\) [RunMigration](<https://github.com/bartossh/Computantis/blob/main/repomongo/migrations.go#L289>)

```go
func (c DataBase) RunMigration(ctx context.Context) error
```

RunMigrationUp runs all the migrations

### func \(DataBase\) [Write](<https://github.com/bartossh/Computantis/blob/main/repomongo/logger.go#L13>)

```go
func (db DataBase) Write(p []byte) (n int, err error)
```

Write writes log to the database. p is a marshaled logger.Log.

### func \(DataBase\) [WriteAddress](<https://github.com/bartossh/Computantis/blob/main/repomongo/address.go#L14>)

```go
func (db DataBase) WriteAddress(ctx context.Context, addr string) error
```

WriteAddress writes unique address to the database.

### func \(DataBase\) [WriteBlock](<https://github.com/bartossh/Computantis/blob/main/repomongo/block.go#L45>)

```go
func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error
```

WriteBlock writes block to the database.

### func \(DataBase\) [WriteIssuerSignedTransactionForReceiver](<https://github.com/bartossh/Computantis/blob/main/repomongo/transaction.go#L27-L31>)

```go
func (db DataBase) WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
```

WriteIssuerSignedTransactionForReceiver writes transaction to the awaiting transaction storage paired with given receiver.

### func \(DataBase\) [WriteTemporaryTransaction](<https://github.com/bartossh/Computantis/blob/main/repomongo/transaction.go#L14>)

```go
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
```

WriteTemporaryTransaction writes transaction to the temporary storage.

### func \(DataBase\) [WriteToken](<https://github.com/bartossh/Computantis/blob/main/repomongo/token.go#L32>)

```go
func (db DataBase) WriteToken(ctx context.Context, tkn string, expirationDate int64) error
```

WriteToken writes unique token to the database.

### func \(DataBase\) [WriteTransactionsInBlock](<https://github.com/bartossh/Computantis/blob/main/repomongo/search.go#L14>)

```go
func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
```

WriteTransactionsInBlock stores relation between Transaction and Block to which Transaction was added.

### func \(DataBase\) [WriteValidatorStatus](<https://github.com/bartossh/Computantis/blob/main/repomongo/validator.go#L12>)

```go
func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *validator.Status) error
```

WriteValidatorStatus writes validator status to the database.

## type [Migration](<https://github.com/bartossh/Computantis/blob/main/repomongo/migrations.go#L24-L26>)

Migration describes migration that is made in the repository database.

```go
type Migration struct {
    Name string `json:"name" bson:"name"`
}
```

# repopostgre

```go
import "github.com/bartossh/Computantis/repopostgre"
```

## Index

- [Variables](<#variables>)
- [type DataBase](<#type-database>)
  - [func Connect(ctx context.Context, conn, database string) (*DataBase, error)](<#func-connect>)
  - [func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error)](<#func-database-checkaddressexists>)
  - [func (db DataBase) CheckToken(ctx context.Context, tkn string) (bool, error)](<#func-database-checktoken>)
  - [func (db DataBase) Disconnect(ctx context.Context) error](<#func-database-disconnect>)
  - [func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error)](<#func-database-findaddress>)
  - [func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)](<#func-database-findtransactioninblockhash>)
  - [func (db DataBase) InvalidateToken(ctx context.Context, token string) error](<#func-database-invalidatetoken>)
  - [func (db DataBase) LastBlock(ctx context.Context) (block.Block, error)](<#func-database-lastblock>)
  - [func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error](<#func-database-movetransactionsfromtemporarytopermanent>)
  - [func (db DataBase) Ping(ctx context.Context) error](<#func-database-ping>)
  - [func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)](<#func-database-readawaitingtransactionsbyissuer>)
  - [func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)](<#func-database-readawaitingtransactionsbyreceiver>)
  - [func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)](<#func-database-readblockbyhash>)
  - [func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error)](<#func-database-readlastnvalidatorstatuses>)
  - [func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error)](<#func-database-readtemporarytransactions>)
  - [func (db DataBase) RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error](<#func-database-removeawaitingtransaction>)
  - [func (DataBase) RunMigration(_ context.Context) error](<#func-database-runmigration>)
  - [func (db DataBase) Write(p []byte) (n int, err error)](<#func-database-write>)
  - [func (db DataBase) WriteAddress(ctx context.Context, addr string) error](<#func-database-writeaddress>)
  - [func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error](<#func-database-writeblock>)
  - [func (db DataBase) WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error](<#func-database-writeissuersignedtransactionforreceiver>)
  - [func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error](<#func-database-writetemporarytransaction>)
  - [func (db DataBase) WriteToken(ctx context.Context, tkn string, expirationDate int64) error](<#func-database-writetoken>)
  - [func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error](<#func-database-writetransactionsinblock>)
  - [func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *validator.Status) error](<#func-database-writevalidatorstatus>)


## Variables

```go
var (
    ErrInsertFailed    = fmt.Errorf("insert failed")
    ErrRemoveFailed    = fmt.Errorf("remove failed")
    ErrSelectFailed    = fmt.Errorf("select failed")
    ErrMoveFailed      = fmt.Errorf("move failed")
    ErrScanFailed      = fmt.Errorf("scan failed")
    ErrUnmarshalFailed = fmt.Errorf("unmarshal failed")
    ErrCommitFailed    = fmt.Errorf("transaction commit failed")
)
```

## type [DataBase](<https://github.com/bartossh/Computantis/blob/main/repopostgre/repopostgre.go#L23-L25>)

Database provides database access for read, write and delete of repository entities.

```go
type DataBase struct {
    // contains filtered or unexported fields
}
```

### func [Connect](<https://github.com/bartossh/Computantis/blob/main/repopostgre/repopostgre.go#L28>)

```go
func Connect(ctx context.Context, conn, database string) (*DataBase, error)
```

Connect creates new connection to the repository and returns pointer to the DataBase.

### func \(DataBase\) [CheckAddressExists](<https://github.com/bartossh/Computantis/blob/main/repopostgre/address.go#L18>)

```go
func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error)
```

CheckAddressExists checks if address exists in the database.

### func \(DataBase\) [CheckToken](<https://github.com/bartossh/Computantis/blob/main/repopostgre/token.go#L14>)

```go
func (db DataBase) CheckToken(ctx context.Context, tkn string) (bool, error)
```

CheckToken checks if token exists in the database is valid and didn't expire.

### func \(DataBase\) [Disconnect](<https://github.com/bartossh/Computantis/blob/main/repopostgre/repopostgre.go#L38>)

```go
func (db DataBase) Disconnect(ctx context.Context) error
```

Disconnect disconnects user from database

### func \(DataBase\) [FindAddress](<https://github.com/bartossh/Computantis/blob/main/repopostgre/search.go#L12>)

```go
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error)
```

FindAddress finds address in the database.

### func \(DataBase\) [FindTransactionInBlockHash](<https://github.com/bartossh/Computantis/blob/main/repopostgre/search.go#L68>)

```go
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
```

FindTransactionInBlockHash finds Block hash in to which Transaction with given hash was added.

### func \(DataBase\) [InvalidateToken](<https://github.com/bartossh/Computantis/blob/main/repopostgre/token.go#L44>)

```go
func (db DataBase) InvalidateToken(ctx context.Context, token string) error
```

InvalidateToken invalidates token.

### func \(DataBase\) [LastBlock](<https://github.com/bartossh/Computantis/blob/main/repopostgre/block.go#L12>)

```go
func (db DataBase) LastBlock(ctx context.Context) (block.Block, error)
```

LastBlock returns last block from the database.

### func \(DataBase\) [MoveTransactionsFromTemporaryToPermanent](<https://github.com/bartossh/Computantis/blob/main/repopostgre/transaction.go#L116>)

```go
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
```

MoveTransactionsFromTemporaryToPermanent moves transactions from temporary storage to permanent storage.

### func \(DataBase\) [Ping](<https://github.com/bartossh/Computantis/blob/main/repopostgre/repopostgre.go#L43>)

```go
func (db DataBase) Ping(ctx context.Context) error
```

Ping checks if the connection to the database is still alive.

### func \(DataBase\) [ReadAwaitingTransactionsByIssuer](<https://github.com/bartossh/Computantis/blob/main/repopostgre/transaction.go#L89>)

```go
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
```

RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.

### func \(DataBase\) [ReadAwaitingTransactionsByReceiver](<https://github.com/bartossh/Computantis/blob/main/repopostgre/transaction.go#L62>)

```go
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
```

ReadAwaitingTransactionsByReceiver reads all transactions paired with given receiver address.

### func \(DataBase\) [ReadBlockByHash](<https://github.com/bartossh/Computantis/blob/main/repopostgre/block.go#L41>)

```go
func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
```

ReadBlockByHash returns block with given hash.

### func \(DataBase\) [ReadLastNValidatorStatuses](<https://github.com/bartossh/Computantis/blob/main/repopostgre/validator.go#L25>)

```go
func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error)
```

ReadLastNValidatorStatuses reads last validator statuses from the database.

### func \(DataBase\) [ReadTemporaryTransactions](<https://github.com/bartossh/Computantis/blob/main/repopostgre/transaction.go#L167>)

```go
func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error)
```

ReadTemporaryTransactions reads all transactions from the temporary storage.

### func \(DataBase\) [RemoveAwaitingTransaction](<https://github.com/bartossh/Computantis/blob/main/repopostgre/transaction.go#L31>)

```go
func (db DataBase) RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error
```

RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.

### func \(DataBase\) [RunMigration](<https://github.com/bartossh/Computantis/blob/main/repopostgre/migrations.go#L7>)

```go
func (DataBase) RunMigration(_ context.Context) error
```

RunMigration satisfies the RepositoryProvider interface as PostgreSQL migrations are run on when database is created in docker\-compose\-postgresql.yml.

### func \(DataBase\) [Write](<https://github.com/bartossh/Computantis/blob/main/repopostgre/logger.go#L12>)

```go
func (db DataBase) Write(p []byte) (n int, err error)
```

Write writes log to the database. p is a marshaled logger.Log.

### func \(DataBase\) [WriteAddress](<https://github.com/bartossh/Computantis/blob/main/repopostgre/address.go#L9>)

```go
func (db DataBase) WriteAddress(ctx context.Context, addr string) error
```

WriteAddress writes address to the database.

### func \(DataBase\) [WriteBlock](<https://github.com/bartossh/Computantis/blob/main/repopostgre/block.go#L69>)

```go
func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error
```

WriteBlock writes block to the database.

### func \(DataBase\) [WriteIssuerSignedTransactionForReceiver](<https://github.com/bartossh/Computantis/blob/main/repopostgre/transaction.go#L40-L44>)

```go
func (db DataBase) WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
```

WriteIssuerSignedTransactionForReceiver writes transaction to the awaiting transaction storage paired with given receiver.

### func \(DataBase\) [WriteTemporaryTransaction](<https://github.com/bartossh/Computantis/blob/main/repopostgre/transaction.go#L13>)

```go
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
```

WriteTemporaryTransaction writes transaction to the temporary storage.

### func \(DataBase\) [WriteToken](<https://github.com/bartossh/Computantis/blob/main/repopostgre/token.go#L34>)

```go
func (db DataBase) WriteToken(ctx context.Context, tkn string, expirationDate int64) error
```

WriteToken writes unique token to the database.

### func \(DataBase\) [WriteTransactionsInBlock](<https://github.com/bartossh/Computantis/blob/main/repopostgre/search.go#L35>)

```go
func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
```

WriteTransactionsInBlock stores relation between Transaction and Block to which Transaction was added.

### func \(DataBase\) [WriteValidatorStatus](<https://github.com/bartossh/Computantis/blob/main/repopostgre/validator.go#L12>)

```go
func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *validator.Status) error
```

WriteValidatorStatus writes validator status to the database.

# serializer

```go
import "github.com/bartossh/Computantis/serializer"
```

## Index

- [func Base58Decode(input []byte) ([]byte, error)](<#func-base58decode>)
- [func Base58Encode(input []byte) []byte](<#func-base58encode>)


## func [Base58Decode](<https://github.com/bartossh/Computantis/blob/main/serializer/serializer.go#L13>)

```go
func Base58Decode(input []byte) ([]byte, error)
```

Base58Decode decodes base58 string to byte array.

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
- [func Run(ctx context.Context, c Config, repo Repository, bookkeeping Bookkeeper, pv RandomDataProvideValidator, log logger.Logger, rx ReactiveSubscriberProvider) error](<#func-run>)
- [type AliveResponse](<#type-aliveresponse>)
- [type AwaitedIssuedTransactionRequest](<#type-awaitedissuedtransactionrequest>)
- [type AwaitedTransactionResponse](<#type-awaitedtransactionresponse>)
- [type Bookkeeper](<#type-bookkeeper>)
- [type Config](<#type-config>)
- [type CreateAddressRequest](<#type-createaddressrequest>)
- [type CreateAddressResponse](<#type-createaddressresponse>)
- [type DataToSignRequest](<#type-datatosignrequest>)
- [type DataToSignResponse](<#type-datatosignresponse>)
- [type IssuedTransactionResponse](<#type-issuedtransactionresponse>)
- [type Message](<#type-message>)
- [type RandomDataProvideValidator](<#type-randomdataprovidevalidator>)
- [type ReactiveSubscriberProvider](<#type-reactivesubscriberprovider>)
- [type Repository](<#type-repository>)
- [type SearchAddressRequest](<#type-searchaddressrequest>)
- [type SearchAddressResponse](<#type-searchaddressresponse>)
- [type SearchBlockRequest](<#type-searchblockrequest>)
- [type SearchBlockResponse](<#type-searchblockresponse>)
- [type TransactionConfirmProposeResponse](<#type-transactionconfirmproposeresponse>)
- [type TransactionProposeRequest](<#type-transactionproposerequest>)
- [type Verifier](<#type-verifier>)


## Constants

```go
const (
    ApiVersion = "1.0.0"
    Header     = "Computantis-Central"
)
```

```go
const (
    AliveURL              = "/alive"                         // URL to check if server is alive and version.
    SearchAddressURL      = searchGroupURL + addressURL      // URL to search for address.
    SearchBlockURL        = searchGroupURL + blockURL        // URL to search for block that contains transaction hash.
    ProposeTransactionURL = transactionGroupURL + proposeURL // URL to propose transaction signed by the issuer.
    ConfirmTransactionURL = transactionGroupURL + confirmURL // URL to confirm transaction signed by the receiver.
    AwaitedTransactionURL = transactionGroupURL + awaitedURL // URL to get awaited transactions for the receiver.
    IssuedTransactionURL  = transactionGroupURL + issuedURL  // URL to get issued transactions for the issuer.
    DataToValidateURL     = validatorGroupURL + dataURL      // URL to get data to validate address by signing rew message.
    CreateAddressURL      = addressGroupURL + createURL      // URL to create new address.
    WsURL                 = "/ws"                            // URL to connect to websocket.
)
```

```go
const (
    CommandNewBlock       = "command_new_block"
    CommandNewTransaction = "command_new_transaction"
)
```

## Variables

```go
var (
    ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")
    ErrWrongMessageSize   = errors.New("message size must be between 1024 and 15000000")
)
```

## func [Run](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L117-L121>)

```go
func Run(ctx context.Context, c Config, repo Repository, bookkeeping Bookkeeper, pv RandomDataProvideValidator, log logger.Logger, rx ReactiveSubscriberProvider) error
```

Run initializes routing and runs the server. To stop the server cancel the context. It blocks until the context is canceled.

## type [AliveResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L11-L15>)

AliveResponse is a response for alive and version check.

```go
type AliveResponse struct {
    Alive      bool   `json:"alive"`
    APIVersion string `json:"api_version"`
    APIHeader  string `json:"api_header"`
}
```

## type [AwaitedIssuedTransactionRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L150-L155>)

AwaitedIssuedTransactionRequest is a request to get awaited or issued transactions for given address. Request contains of Address for which Transactions are requested, Data in binary format, Hash of Data and Signature of the Data to prove that entity doing the request is an Address owner.

```go
type AwaitedIssuedTransactionRequest struct {
    Address   string   `json:"address"`
    Data      []byte   `json:"data"`
    Hash      [32]byte `json:"hash"`
    Signature []byte   `json:"signature"`
}
```

## type [AwaitedTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L158-L161>)

AwaitedTransactionResponse is a response for awaited transactions request.

```go
type AwaitedTransactionResponse struct {
    Success             bool                      `json:"success"`
    AwaitedTransactions []transaction.Transaction `json:"awaited_transactions"`
}
```

## type [Bookkeeper](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L78-L83>)

Bookkeeper abstracts methods of the bookkeeping of a blockchain.

```go
type Bookkeeper interface {
    Verifier
    Run(ctx context.Context)
    WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error
    WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
}
```

## type [Config](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L100-L103>)

Config contains configuration of the server.

```go
type Config struct {
    Port          int `yaml:"port"`            // Port to listen on.
    DataSizeBytes int `yaml:"data_size_bytes"` // Size of the data to be stored in the transaction.
}
```

## type [CreateAddressRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L259-L265>)

CreateAddressRequest is a request to create an address.

```go
type CreateAddressRequest struct {
    Address   string   `json:"address"`
    Token     string   `json:"token"`
    Data      []byte   `json:"data"`
    Hash      [32]byte `json:"hash"`
    Signature []byte   `json:"signature"`
}
```

## type [CreateAddressResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L269-L272>)

Response for address creation request. If Success is true, Address contains created address in base58 format.

```go
type CreateAddressResponse struct {
    Success bool   `json:"success"`
    Address string `json:"address"`
}
```

## type [DataToSignRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L239-L241>)

DataToSignRequest is a request to get data to sign for proving identity.

```go
type DataToSignRequest struct {
    Address string `json:"address"`
}
```

## type [DataToSignResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L244-L246>)

DataToSignRequest is a response containing data to sign for proving identity.

```go
type DataToSignResponse struct {
    Data []byte `json:"message"`
}
```

## type [IssuedTransactionResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L199-L202>)

AwaitedTransactionResponse is a response for issued transactions request.

```go
type IssuedTransactionResponse struct {
    Success            bool                      `json:"success"`
    IssuedTransactions []transaction.Transaction `json:"issued_transactions"`
}
```

## type [Message](<https://github.com/bartossh/Computantis/blob/main/server/ws.go#L41-L46>)

Message is the message that is used to exchange information between the server and the client.

```go
type Message struct {
    Command     string                  `json:"command"`     // Command is the command that refers to the action handler in websocket protocol.
    Error       string                  `json:"error"`       // Error is the error message that is sent to the client.
    Block       block.Block             `json:"block"`       // Block is the block that is sent to the client.
    Transaction transaction.Transaction `json:"transaction"` // Transaction is the transaction validated by the central server and will be added to the next block.
}
```

## type [RandomDataProvideValidator](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L87-L90>)

RandomDataProvideValidator provides random binary data for signing to prove identity and the validator of data being valid and not expired.

```go
type RandomDataProvideValidator interface {
    ProvideData(address string) []byte
    ValidateData(address string, data []byte) bool
}
```

## type [ReactiveSubscriberProvider](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L94-L97>)

ReactiveSubscriberProvider provides reactive subscription to the blockchain. It allows to listen for the new blocks created by the Ladger.

```go
type ReactiveSubscriberProvider interface {
    Cancel()
    Channel() <-chan block.Block
}
```

## type [Repository](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L59-L70>)

Repository is the interface that wraps the basic CRUD and Search methods. Repository should be properly indexed to allow for transaction and block hash. as well as address public keys to be and unique and the hash lookup should be fast. Repository holds the blocks and transaction that are part of the blockchain.

```go
type Repository interface {
    Disconnect(ctx context.Context) error
    RunMigration(ctx context.Context) error
    FindAddress(ctx context.Context, search string, limit int) ([]string, error)
    CheckAddressExists(ctx context.Context, address string) (bool, error)
    WriteAddress(ctx context.Context, address string) error
    FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
    CheckToken(ctx context.Context, token string) (bool, error)
    InvalidateToken(ctx context.Context, token string) error
    ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
    ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
}
```

## type [SearchAddressRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L27-L29>)

SearchAddressRequest is a request to search for address.

```go
type SearchAddressRequest struct {
    Address string `json:"address"`
}
```

## type [SearchAddressResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L32-L34>)

SearchAddressResponse is a response for address search.

```go
type SearchAddressResponse struct {
    Addresses []string `json:"addresses"`
}
```

## type [SearchBlockRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L55-L57>)

SearchBlockRequest is a request to search for block.

```go
type SearchBlockRequest struct {
    RawTrxHash [32]byte `json:"raw_trx_hash"`
}
```

## type [SearchBlockResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L60-L62>)

SearchBlockResponse is a response for block search.

```go
type SearchBlockResponse struct {
    RawBlockHash [32]byte `json:"raw_block_hash"`
}
```

## type [TransactionConfirmProposeResponse](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L90-L93>)

TransactionConfirmProposeResponse is a response for transaction propose.

```go
type TransactionConfirmProposeResponse struct {
    Success bool     `json:"success"`
    TrxHash [32]byte `json:"trx_hash"`
}
```

## type [TransactionProposeRequest](<https://github.com/bartossh/Computantis/blob/main/server/rest.go#L84-L87>)

TransactionProposeRequest is a request to propose a transaction.

```go
type TransactionProposeRequest struct {
    ReceiverAddr string                  `json:"receiver_addr"`
    Transaction  transaction.Transaction `json:"transaction"`
}
```

## type [Verifier](<https://github.com/bartossh/Computantis/blob/main/server/server.go#L73-L75>)

Verifier provides methods to verify the signature of the message.

```go
type Verifier interface {
    VerifySignature(message, signature []byte, hash [32]byte, address string) error
}
```

# stress

```go
import "github.com/bartossh/Computantis/stress"
```

Stress is a package that provides a simple way to stress test your code on the full cycle of transaction processing.

## Index



# token

```go
import "github.com/bartossh/Computantis/token"
```

## Index

- [type Token](<#type-token>)


## type [Token](<https://github.com/bartossh/Computantis/blob/main/token/token.go#L6-L11>)

Token holds information about unique token. Token is a way of proving to the REST API of the central server that the request is valid and comes from the client that is allowed to use the API.

```go
type Token struct {
    ID             any    `json:"-"               bson:"_id,omitempty"   db:"id"`
    Token          string `json:"token"           bson:"token"           db:"token"`
    Valid          bool   `json:"valid"           bson:"valid"           db:"valid"`
    ExpirationDate int64  `json:"expiration_date" bson:"expiration_date" db:"expiration_date"`
}
```

# transaction

```go
import "github.com/bartossh/Computantis/transaction"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [type Signer](<#type-signer>)
- [type Transaction](<#type-transaction>)
  - [func New(subject string, message []byte, issuer Signer) (Transaction, error)](<#func-new>)
  - [func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error)](<#func-transaction-sign>)
- [type TransactionAwaitingReceiverSignature](<#type-transactionawaitingreceiversignature>)
- [type TransactionInBlock](<#type-transactioninblock>)
- [type Verifier](<#type-verifier>)


## Constants

```go
const ExpirationTimeInDays = 7 // transaction validity expiration time in days. TODO: move to config
```

## Variables

```go
var (
    ErrTransactionHasAFutureTime        = errors.New("transaction has a future time")
    ErrExpiredTransaction               = errors.New("transaction has expired")
    ErrTransactionHashIsInvalid         = errors.New("transaction hash is invalid")
    ErrSignatureNotValidOrDataCorrupted = errors.New("signature not valid or data are corrupted")
)
```

## type [Signer](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L22-L25>)

Signer provides signing and address methods.

```go
type Signer interface {
    Sign(message []byte) (digest [32]byte, signature []byte)
    Address() string
}
```

## type [Transaction](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L39-L49>)

Transaction contains transaction information, subject type, subject data, signatures and public keys. Transaction is valid for a week from being issued. Subject represents an information how to read the Data and / or how to decode them. Data is not validated by the computantis server, Ladger ior block. What is stored in Data is not important for the whole Computantis system. It is only important that the data are signed by the issuer and the receiver and both parties agreed on them.

```go
type Transaction struct {
    ID                any       `json:"-"                  bson:"_id"                db:"id"`
    CreatedAt         time.Time `json:"created_at"         bson:"created_at"         db:"created_at"`
    Hash              [32]byte  `json:"hash"               bson:"hash"               db:"hash"`
    IssuerAddress     string    `json:"issuer_address"     bson:"issuer_address"     db:"issuer_address"`
    ReceiverAddress   string    `json:"receiver_address"   bson:"receiver_address"   db:"receiver_address"`
    Subject           string    `json:"subject"            bson:"subject"            db:"subject"`
    Data              []byte    `json:"data"               bson:"data"               db:"data"`
    IssuerSignature   []byte    `json:"issuer_signature"   bson:"issuer_signature"   db:"issuer_signature"`
    ReceiverSignature []byte    `json:"receiver_signature" bson:"receiver_signature" db:"receiver_signature"`
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L52>)

```go
func New(subject string, message []byte, issuer Signer) (Transaction, error)
```

New creates new transaction signed by the issuer.

### func \(\*Transaction\) [Sign](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L73>)

```go
func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error)
```

Sign verifies issuer signature and signs Transaction by the receiver.

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

## type [TransactionInBlock](<https://github.com/bartossh/Computantis/blob/main/transaction/entities.go#L5-L9>)

TransactionInBlock stores relation between Transaction and Block to which Transaction was added. It is stored for fast lookup only to allow to find Block hash in which Transaction was added.

```go
type TransactionInBlock struct {
    ID              any      `json:"-" bson:"_id,omitempty"    db:"id"`
    BlockHash       [32]byte `json:"-" bson:"block_hash"       db:"block_hash"`
    TransactionHash [32]byte `json:"-" bson:"transaction_hash" db:"transaction_hash"`
}
```

## type [Verifier](<https://github.com/bartossh/Computantis/blob/main/transaction/transaction.go#L28-L30>)

Verifier provides signature verification method.

```go
type Verifier interface {
    Verify(message, signature []byte, hash [32]byte, address string) error
}
```

# validator

```go
import "github.com/bartossh/Computantis/validator"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [func Run(ctx context.Context, cfg Config, srw StatusReadWriter, log logger.Logger, ver Verifier, wh WebhookCreateRemovePoster, wallet *wallet.Wallet) error](<#func-run>)
- [type Config](<#type-config>)
- [type CreateRemoveUpdateHookRequest](<#type-createremoveupdatehookrequest>)
- [type Status](<#type-status>)
- [type StatusReadWriter](<#type-statusreadwriter>)
- [type Verifier](<#type-verifier>)
- [type WebHookNewBlockMessage](<#type-webhooknewblockmessage>)
- [type WebhookCreateRemovePoster](<#type-webhookcreateremoveposter>)


## Constants

```go
const (
    Header = "Computantis-Validator"
)
```

## Variables

```go
var (
    ErrProofBlockIsInvalid    = fmt.Errorf("block proof is invalid")
    ErrBlockIndexIsInvalid    = fmt.Errorf("block index is invalid")
    ErrBlockPrevHashIsInvalid = fmt.Errorf("block previous hash is invalid")
)
```

## func [Run](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L84>)

```go
func Run(ctx context.Context, cfg Config, srw StatusReadWriter, log logger.Logger, ver Verifier, wh WebhookCreateRemovePoster, wallet *wallet.Wallet) error
```

Run initializes routing and runs the validator. To stop the validator cancel the context. Validator connects to the central server via websocket and listens for new blocks. It will block until the context is canceled.

## type [Config](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L61-L65>)

Config contains configuration of the validator.

```go
type Config struct {
    Token     string `yaml:"token"`     // token is used to authenticate validator in the central server
    Websocket string `yaml:"websocket"` // websocket address of the central server
    Port      int    `yaml:"port"`      // port on which validator will listen for http requests
}
```

## type [CreateRemoveUpdateHookRequest](<https://github.com/bartossh/Computantis/blob/main/validator/webhook.go#L6-L14>)

CreateRemoveUpdateHookRequest is the request sent to create, remove or update the webhook.

```go
type CreateRemoveUpdateHookRequest struct {
    URL       string `json:"address"`        // URL is a url  of the webhook.
    Hook      string `json:"hook"`           // Hook is a type of the webhook. It describes on what event the webhook is triggered.
    Address   string `json:"wallet_address"` // Address is the address of the wallet that is used to sign the webhook.
    Token     string `json:"token"`          // Token is the token added to the webhook to verify that the message comes from the valid source.
    Data      []byte `json:"data"`           // Data is the data is a subject of the signature. It is signed by the wallet address.
    Digest    []byte `json:"digest"`         // Digest is the digest of the data. It is used to verify that the data is not changed.
    Signature []byte `json:"signature"`      // Signature is the signature of the data. It is used to verify that the data is not changed.
}
```

## type [Status](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L34-L40>)

Status is a status of each received block by the validator. It keeps track of invalid blocks in case of blockchain corruption.

```go
type Status struct {
    ID        any         `json:"-"          bson:"_id,omitempty" db:"id"`
    Index     int64       `json:"index"      bson:"index"         db:"index"`
    Block     block.Block `json:"block"      bson:"block"         db:"-"`
    Valid     bool        `json:"valid"      bson:"valid"         db:"valid"`
    CreatedAt time.Time   `json:"created_at" bson:"created_at"    db:"created_at"`
}
```

## type [StatusReadWriter](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L43-L46>)

StatusReadWriter provides methods to bulk read and single write validator status.

```go
type StatusReadWriter interface {
    WriteValidatorStatus(ctx context.Context, vs *Status) error
    ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]Status, error)
}
```

## type [Verifier](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L56-L58>)

Verifier provides methods to verify the signature of the message.

```go
type Verifier interface {
    Verify(message, signature []byte, hash [32]byte, address string) error
}
```

## type [WebHookNewBlockMessage](<https://github.com/bartossh/Computantis/blob/main/validator/webhook.go#L17-L21>)

WebHookNewBlockMessage is the message sent to the webhook url that was created.

```go
type WebHookNewBlockMessage struct {
    Token string      `json:"token"` // Token given to the webhook by the webhooks creator to validate the message source.
    Block block.Block `json:"block"` // Block is the block that was mined.
    Valid bool        `json:"valid"` // Valid is the flag that indicates if the block is valid.
}
```

## type [WebhookCreateRemovePoster](<https://github.com/bartossh/Computantis/blob/main/validator/validator.go#L49-L53>)

WebhookCreateRemovePoster provides methods to create, remove webhooks and post messages to webhooks.

```go
type WebhookCreateRemovePoster interface {
    CreateWebhook(trigger string, h webhooks.Hook) error
    RemoveWebhook(trigger string, h webhooks.Hook) error
    PostWebhookBlock(blc *block.Block)
}
```

# wallet

```go
import "github.com/bartossh/Computantis/wallet"
```

## Index

- [type Helper](<#type-helper>)
  - [func NewVerifier() Helper](<#func-newverifier>)
  - [func (h Helper) AddressToPubKey(address string) (ed25519.PublicKey, error)](<#func-helper-addresstopubkey>)
  - [func (h Helper) Verify(message, signature []byte, hash [32]byte, address string) error](<#func-helper-verify>)
- [type Wallet](<#type-wallet>)
  - [func DecodeGOBWallet(data []byte) (Wallet, error)](<#func-decodegobwallet>)
  - [func New() (Wallet, error)](<#func-new>)
  - [func (w *Wallet) Address() string](<#func-wallet-address>)
  - [func (w *Wallet) ChecksumLength() int](<#func-wallet-checksumlength>)
  - [func (w *Wallet) EncodeGOB() ([]byte, error)](<#func-wallet-encodegob>)
  - [func (w *Wallet) Sign(message []byte) (digest [32]byte, signature []byte)](<#func-wallet-sign>)
  - [func (w *Wallet) Verify(message, signature []byte, hash [32]byte) bool](<#func-wallet-verify>)
  - [func (w *Wallet) Version() byte](<#func-wallet-version>)


## type [Helper](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L13>)

Helper provides wallet helper functionalities without knowing about wallet private and public keys.

```go
type Helper struct{}
```

### func [NewVerifier](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L16>)

```go
func NewVerifier() Helper
```

NewVerifier creates new wallet Helper verifier.

### func \(Helper\) [AddressToPubKey](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L21>)

```go
func (h Helper) AddressToPubKey(address string) (ed25519.PublicKey, error)
```

AddressToPubKey creates ED25519 public key from address, or returns error otherwise.

### func \(Helper\) [Verify](<https://github.com/bartossh/Computantis/blob/main/wallet/verifier.go#L42>)

```go
func (h Helper) Verify(message, signature []byte, hash [32]byte, address string) error
```

Verify verifies if message is signed by given key and hash is equal.

## type [Wallet](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L20-L23>)

Wallet holds public and private key of the wallet owner.

```go
type Wallet struct {
    Private ed25519.PrivateKey `json:"private" bson:"private"`
    Public  ed25519.PublicKey  `json:"public" bson:"public"`
}
```

### func [DecodeGOBWallet](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L35>)

```go
func DecodeGOBWallet(data []byte) (Wallet, error)
```

DecodeGOBWallet tries to decode Wallet from gob representation or returns error otherwise.

### func [New](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L26>)

```go
func New() (Wallet, error)
```

New tries to creates a new Wallet or returns error otherwise.

### func \(\*Wallet\) [Address](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L70>)

```go
func (w *Wallet) Address() string
```

Address creates address from the public key that contains wallet version and checksum.

### func \(\*Wallet\) [ChecksumLength](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L60>)

```go
func (w *Wallet) ChecksumLength() int
```

ChecksumLength returns checksum length.

### func \(\*Wallet\) [EncodeGOB](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L47>)

```go
func (w *Wallet) EncodeGOB() ([]byte, error)
```

EncodeGOB tries to encodes Wallet in to the gob representation or returns error otherwise.

### func \(\*Wallet\) [Sign](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L82>)

```go
func (w *Wallet) Sign(message []byte) (digest [32]byte, signature []byte)
```

Sign signs the message with Ed25519 signature. Returns digest hash sha256 and signature.

### func \(\*Wallet\) [Verify](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L90>)

```go
func (w *Wallet) Verify(message, signature []byte, hash [32]byte) bool
```

Verify verifies message ED25519 signature and hash. Uses hashing sha256.

### func \(\*Wallet\) [Version](<https://github.com/bartossh/Computantis/blob/main/wallet/wallet.go#L65>)

```go
func (w *Wallet) Version() byte
```

Version returns wallet version.

# webhooks

```go
import "github.com/bartossh/Computantis/webhooks"
```

## Index

- [Constants](<#constants>)
- [Variables](<#variables>)
- [type Hook](<#type-hook>)
- [type HookRequestHTTPPoster](<#type-hookrequesthttpposter>)
- [type Service](<#type-service>)
  - [func New(client HookRequestHTTPPoster, l logger.Logger) *Service](<#func-new>)
  - [func (s *Service) CreateWebhook(trigger string, h Hook) error](<#func-service-createwebhook>)
  - [func (s *Service) PostWebhookBlock(blc *block.Block)](<#func-service-postwebhookblock>)
  - [func (s *Service) RemoveWebhook(trigger string, h Hook) error](<#func-service-removewebhook>)


## Constants

```go
const (
    TriggerNewBlock = "trigger_new_block" // TriggerNewBlock is the trigger for new block. It is triggered when a new block is forged.
)
```

## Variables

```go
var (
    ErrorHookNotImplemented = errors.New("hook not implemented")
)
```

## type [Hook](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L21-L24>)

Hook is the hook that is used to trigger the webhook.

```go
type Hook struct {
    URL   string `json:"address"` // URL is a url  of the webhook.
    Token string `json:"token"`   // Token is the token added to the webhook to verify that the message comes from the valid source.
}
```

## type [HookRequestHTTPPoster](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L29-L31>)

HookRequestHTTPPoster provides PostWebhookBlock method that allows to post new forged block to the webhook url over HTTP protocol.

```go
type HookRequestHTTPPoster interface {
    PostWebhookBlock(url string, token string, block *block.Block) error
}
```

## type [Service](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L34-L39>)

Service provide webhook service that is used to create, remove and update webhooks.

```go
type Service struct {
    // contains filtered or unexported fields
}
```

### func [New](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L42>)

```go
func New(client HookRequestHTTPPoster, l logger.Logger) *Service
```

New creates new instance of the webhook service.

### func \(\*Service\) [CreateWebhook](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L52>)

```go
func (s *Service) CreateWebhook(trigger string, h Hook) error
```

CreateWebhook creates new webhook or or updates existing one for given trigger.

### func \(\*Service\) [PostWebhookBlock](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L74>)

```go
func (s *Service) PostWebhookBlock(blc *block.Block)
```

PostWebhookBlock posts block to all webhooks that are subscribed to the new block trigger.

### func \(\*Service\) [RemoveWebhook](<https://github.com/bartossh/Computantis/blob/main/webhooks/webhooks.go#L63>)

```go
func (s *Service) RemoveWebhook(trigger string, h Hook) error
```

RemoveWebhook removes webhook for given trigger and Hook URL.

# central

```go
import "github.com/bartossh/Computantis/cmd/central"
```

## Index



# validator

```go
import "github.com/bartossh/Computantis/cmd/validator"
```

## Index





Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
