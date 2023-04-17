---
layout: page
title: Documentation.
subtitle: Package and the REST API ersver.
---

# block

```go
import "github.com/bartossh/The-Accountant/block"
```

## Index

- [type Block](<#type-block>)
  - [func NewBlock(difficulty, next uint64, prevHash [32]byte, trxHashes [][32]byte) Block](<#func-newblock>)
  - [func (b *Block) Validate(trxHashes [][32]byte) bool](<#func-block-validate>)


## type [Block](<https://github.com/bartossh/The-Accountant/blob/main/block/block.go#L17-L26>)

Block holds block information.

```go
type Block struct {
    ID         primitive.ObjectID `json:"-"          bson:"_id"`
    Index      uint64             `json:"index"      bson:"index"`
    Timestamp  uint64             `json:"timestamp"  bson:"timestamp"`
    Nonce      uint64             `json:"nonce"      bson:"nonce"`
    Difficulty uint64             `json:"difficulty" bson:"difficulty"`
    Hash       [32]byte           `json:"hash"       bson:"hash"`
    PrevHash   [32]byte           `json:"prev_hash"  bson:"prev_hash"`
    TrxHashes  [][32]byte         `json:"trx_hashes" bson:"trx_hashes"`
}
```

### func [NewBlock](<https://github.com/bartossh/The-Accountant/blob/main/block/block.go#L32>)

```go
func NewBlock(difficulty, next uint64, prevHash [32]byte, trxHashes [][32]byte) Block
```

NewBlock creates a new Block hashing it with given difficulty. Higher difficulty requires more computations to happen to find possible target hash. Difficulty is stored inside the Block and is a part of a hashed data. Transactions hashes are prehashed before calculating the Block hash with merkle tree.

### func \(\*Block\) [Validate](<https://github.com/bartossh/The-Accountant/blob/main/block/block.go#L57>)

```go
func (b *Block) Validate(trxHashes [][32]byte) bool
```

Validate validates the Block. Validations goes in the same order like Block hashing allgorithm, just the proof of work part is not required as Nonce is arleady known.

# blockchain

```go
import "github.com/bartossh/The-Accountant/blockchain"
```

## Index

- [Variables](<#variables>)
- [type BlockReadWriter](<#type-blockreadwriter>)
- [type BlockReader](<#type-blockreader>)
- [type BlockWriter](<#type-blockwriter>)
- [type Blockchain](<#type-blockchain>)
  - [func NewBlockchain(ctx context.Context, rw BlockReadWriter) (*Blockchain, error)](<#func-newblockchain>)
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

## type [BlockReadWriter](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L27-L30>)

```go
type BlockReadWriter interface {
    BlockReader
    BlockWriter
}
```

## type [BlockReader](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L18-L21>)

```go
type BlockReader interface {
    LastBlock(ctx context.Context) (block.Block, error)
    ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
}
```

## type [BlockWriter](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L23-L25>)

```go
type BlockWriter interface {
    WriteBlock(ctx context.Context, block block.Block) error
}
```

## type [Blockchain](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L33-L38>)

Blockchain keeps track of the blocks.

```go
type Blockchain struct {
    // contains filtered or unexported fields
}
```

### func [NewBlockchain](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L41>)

```go
func NewBlockchain(ctx context.Context, rw BlockReadWriter) (*Blockchain, error)
```

NewChaion creates a new Blockchain that has access to the blockchain stored in the repository.

### func \(\*Blockchain\) [LastBlockHashIndex](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L56>)

```go
func (c *Blockchain) LastBlockHashIndex() ([32]byte, uint64)
```

LastBlockHashIndex returns last block hash and index.

### func \(\*Blockchain\) [ReadBlocksFromIndex](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L83>)

```go
func (c *Blockchain) ReadBlocksFromIndex(ctx context.Context, idx uint64) ([]block.Block, error)
```

ReadBlocksFromIndex reads all blocks from given index till the current block.

### func \(\*Blockchain\) [ReadLastNBlocks](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L63>)

```go
func (c *Blockchain) ReadLastNBlocks(ctx context.Context, n int) ([]block.Block, error)
```

ReadLastNBlocks reads the last n blocks.

### func \(\*Blockchain\) [WriteBlock](<https://github.com/bartossh/The-Accountant/blob/main/blockchain/blockchain.go#L107>)

```go
func (c *Blockchain) WriteBlock(ctx context.Context, block block.Block) error
```

WriteBlock writes block in to the blockchain repository.

# bookkeeping

```go
import "github.com/bartossh/The-Accountant/bookkeeping"
```

## Index

- [Variables](<#variables>)
- [type AddressChecker](<#type-addresschecker>)
- [type BlockFinder](<#type-blockfinder>)
- [type BlockReadWriter](<#type-blockreadwriter>)
- [type BlockReader](<#type-blockreader>)
- [type BlockWriter](<#type-blockwriter>)
- [type Config](<#type-config>)
  - [func (c Config) Validate() error](<#func-config-validate>)
- [type Ledger](<#type-ledger>)
  - [func NewLedger(config Config, bc BlockReadWriter, tx TrxWriteMover, ac AddressChecker, vr SignatureVerifier, tf BlockFinder) (*Ledger, error)](<#func-newledger>)
  - [func (l *Ledger) Run(ctx context.Context)](<#func-ledger-run>)
  - [func (l *Ledger) WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error](<#func-ledger-writecandidatetransaction>)
- [type SignatureVerifier](<#type-signatureverifier>)
- [type TrxWriteMover](<#type-trxwritemover>)


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

## type [AddressChecker](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L51-L53>)

```go
type AddressChecker interface {
    CheckAddressExists(ctx context.Context, address string) (bool, error)
}
```

## type [BlockFinder](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L59-L62>)

```go
type BlockFinder interface {
    WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
    FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
}
```

## type [BlockReadWriter](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L46-L49>)

```go
type BlockReadWriter interface {
    BlockReader
    BlockWriter
}
```

## type [BlockReader](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L38-L40>)

```go
type BlockReader interface {
    LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error)
}
```

## type [BlockWriter](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L42-L44>)

```go
type BlockWriter interface {
    WriteBlock(ctx context.Context, block block.Block) error
}
```

## type [Config](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L64-L68>)

```go
type Config struct {
    Difficulty            uint64        `json:"difficulty"              bson:"difficulty"              yaml:"difficulty"`
    BlockWriteTimestamp   time.Duration `json:"block_write_timestamp"   bson:"block_write_timestamp"   yaml:"block_write_timestamp"`
    BlockTransactionsSize int           `json:"block_transactions_size" bson:"block_transactions_size" yaml:"block_transactions_size"`
}
```

### func \(Config\) [Validate](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L70>)

```go
func (c Config) Validate() error
```

## type [Ledger](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L87-L96>)

Ledger is a collection of ledger functionality to perform bookkeeping.

```go
type Ledger struct {
    // contains filtered or unexported fields
}
```

### func [NewLedger](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L99-L106>)

```go
func NewLedger(config Config, bc BlockReadWriter, tx TrxWriteMover, ac AddressChecker, vr SignatureVerifier, tf BlockFinder) (*Ledger, error)
```

NewLedger creates new Ledger if config is valid or returns error otherwise.

### func \(\*Ledger\) [Run](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L124>)

```go
func (l *Ledger) Run(ctx context.Context)
```

Run runs the Ladger engine that writes blocks to the blockchain repository. Run starts a goroutine and can be stopped by cancelling the context.

### func \(\*Ledger\) [WriteCandidateTransaction](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L149>)

```go
func (l *Ledger) WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error
```

WriteCandidateTransaction validates and writes a transaction to the repository. Transaction is not yet a part of the blockchain.

## type [SignatureVerifier](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L55-L57>)

```go
type SignatureVerifier interface {
    Verify(message, signature []byte, hash [32]byte, address string) error
}
```

## type [TrxWriteMover](<https://github.com/bartossh/The-Accountant/blob/main/bookkeeping/bookkeeping.go#L33-L36>)

```go
type TrxWriteMover interface {
    WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
    MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
}
```

# repo

```go
import "github.com/bartossh/The-Accountant/repo"
```

## Index

- [type Address](<#type-address>)
- [type DataBase](<#type-database>)
  - [func Connect(ctx context.Context, connStr, databaseName string) (*DataBase, error)](<#func-connect>)
  - [func (db DataBase) CheckAddressExists(ctx context.Context, address string) (bool, error)](<#func-database-checkaddressexists>)
  - [func (c DataBase) Disconnect(ctx context.Context) error](<#func-database-disconnect>)
  - [func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)](<#func-database-findtransactioninblockhash>)
  - [func (db DataBase) LastBlock(ctx context.Context) (block.Block, error)](<#func-database-lastblock>)
  - [func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error](<#func-database-movetransactionsfromtemporarytopermanent>)
  - [func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)](<#func-database-readblockbyhash>)
  - [func (c DataBase) RunMigration(ctx context.Context) error](<#func-database-runmigration>)
  - [func (db DataBase) WriteAddress(ctx context.Context, address Address) error](<#func-database-writeaddress>)
  - [func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error](<#func-database-writeblock>)
  - [func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error](<#func-database-writetemporarytransaction>)
  - [func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error](<#func-database-writetransactionsinblock>)
- [type Migration](<#type-migration>)
- [type TransactionInBlock](<#type-transactioninblock>)


## type [Address](<https://github.com/bartossh/The-Accountant/blob/main/repo/address.go#L13-L16>)

Address holds information about unique PublicKey.

```go
type Address struct {
    ID        primitive.ObjectID `json:"-"          bson:"_id,omitempty"`
    PublicKey string             `json:"public_key" bson:"public_key"`
}
```

## type [DataBase](<https://github.com/bartossh/The-Accountant/blob/main/repo/repo.go#L22-L24>)

Database provides database access for read, write and delete of repository entities.

```go
type DataBase struct {
    // contains filtered or unexported fields
}
```

### func [Connect](<https://github.com/bartossh/The-Accountant/blob/main/repo/repo.go#L27>)

```go
func Connect(ctx context.Context, connStr, databaseName string) (*DataBase, error)
```

Connect creates new connection to the playableassets repository and returns pointer to that user instance

### func \(DataBase\) [CheckAddressExists](<https://github.com/bartossh/The-Accountant/blob/main/repo/address.go#L32>)

```go
func (db DataBase) CheckAddressExists(ctx context.Context, address string) (bool, error)
```

CheckAddressExists checks if address exists in the database.

### func \(DataBase\) [Disconnect](<https://github.com/bartossh/The-Accountant/blob/main/repo/repo.go#L43>)

```go
func (c DataBase) Disconnect(ctx context.Context) error
```

Disconnect disconnects user from database

### func \(DataBase\) [FindTransactionInBlockHash](<https://github.com/bartossh/The-Accountant/blob/main/repo/search.go#L32>)

```go
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
```

FindTransactionInBlockHash finds Block hash in to which Transaction with given hash was added.

### func \(DataBase\) [LastBlock](<https://github.com/bartossh/The-Accountant/blob/main/repo/block.go#L13>)

```go
func (db DataBase) LastBlock(ctx context.Context) (block.Block, error)
```

LastBlock returns last block from the database.

### func \(DataBase\) [MoveTransactionsFromTemporaryToPermanent](<https://github.com/bartossh/The-Accountant/blob/main/repo/transaction.go#L21>)

```go
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
```

MoveTransactionsFromTemporaryToPermanent moves transactions from temporary storage to permanent.

### func \(DataBase\) [ReadBlockByHash](<https://github.com/bartossh/The-Accountant/blob/main/repo/block.go#L35>)

```go
func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
```

ReadBlockByHash returns block with given hash.

### func \(DataBase\) [RunMigration](<https://github.com/bartossh/The-Accountant/blob/main/repo/migrations.go#L190>)

```go
func (c DataBase) RunMigration(ctx context.Context) error
```

RunMigrationUp runs all the migrations

### func \(DataBase\) [WriteAddress](<https://github.com/bartossh/The-Accountant/blob/main/repo/address.go#L19>)

```go
func (db DataBase) WriteAddress(ctx context.Context, address Address) error
```

WriteAddress writes unique address to the database.

### func \(DataBase\) [WriteBlock](<https://github.com/bartossh/The-Accountant/blob/main/repo/block.go#L44>)

```go
func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error
```

WriteBlock writes block to the database.

### func \(DataBase\) [WriteTemporaryTransaction](<https://github.com/bartossh/The-Accountant/blob/main/repo/transaction.go#L13>)

```go
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
```

WriteTemporaryTransaction writes transaction to the temporary storage.

### func \(DataBase\) [WriteTransactionsInBlock](<https://github.com/bartossh/The-Accountant/blob/main/repo/search.go#L19>)

```go
func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
```

WrirteTransactionInBlock stores relation between Transaction and Block to which Transaction was added.

## type [Migration](<https://github.com/bartossh/The-Accountant/blob/main/repo/migrations.go#L24-L26>)

Migration describes migration that is made in the repository database.

```go
type Migration struct {
    Name string `json:"name" bson:"name"`
}
```

## type [TransactionInBlock](<https://github.com/bartossh/The-Accountant/blob/main/repo/search.go#L12-L16>)

TransactionInBlock stores relation between Transaction and Block to which Transaction was added. It is tored for fast lookup only.

```go
type TransactionInBlock struct {
    ID              primitive.ObjectID `json:"-" bson:"_id,omitempty"`
    BlockHash       [32]byte           `json:"-" bson:"block_hash"`
    TransactionHash [32]byte           `json:"-" bson:"transaction_hash"`
}
```

# serializer

```go
import "github.com/bartossh/The-Accountant/serializer"
```

## Index

- [func Base58Decode(input []byte) ([]byte, error)](<#func-base58decode>)
- [func Base58Encode(input []byte) []byte](<#func-base58encode>)


## func [Base58Decode](<https://github.com/bartossh/The-Accountant/blob/main/serializer/serializer.go#L11>)

```go
func Base58Decode(input []byte) ([]byte, error)
```

## func [Base58Encode](<https://github.com/bartossh/The-Accountant/blob/main/serializer/serializer.go#L5>)

```go
func Base58Encode(input []byte) []byte
```

# transaction

```go
import "github.com/bartossh/The-Accountant/transaction"
```

## Index

- [type Signer](<#type-signer>)
- [type Transaction](<#type-transaction>)
  - [func New(subject string, message []byte, issuer Signer) (Transaction, error)](<#func-new>)
  - [func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error)](<#func-transaction-sign>)
- [type Verifier](<#type-verifier>)


## type [Signer](<https://github.com/bartossh/The-Accountant/blob/main/transaction/transaction.go#L14-L17>)

```go
type Signer interface {
    Sign(message []byte) (digest [32]byte, signature []byte)
    Address() string
}
```

## type [Transaction](<https://github.com/bartossh/The-Accountant/blob/main/transaction/transaction.go#L24-L34>)

Transaction contains transaction information, subject type, subject data, signatues and public keys.

```go
type Transaction struct {
    ID                primitive.ObjectID `json:"-"                  bson:"_id"`
    CreatedAt         time.Time          `json:"created_at"         bson:"created_at"`
    Hash              [32]byte           `json:"hash"               bson:"hash"`
    IssuerAddress     string             `json:"issuer_address"     bson:"issuer_address"`
    ReceiverAddress   string             `json:"receiver_address"   bson:"receiver_address"`
    Subject           string             `json:"subject"            bson:"subcject"`
    Data              []byte             `json:"data"               bson:"data"`
    IssuerSignature   []byte             `json:"issuer_signature"   bson:"issuer_signature"`
    ReceiverSignature []byte             `json:"receiver_signature" bson:"receiver_signature"`
}
```

### func [New](<https://github.com/bartossh/The-Accountant/blob/main/transaction/transaction.go#L37>)

```go
func New(subject string, message []byte, issuer Signer) (Transaction, error)
```

New creates new transaction signed by issuer.

### func \(\*Transaction\) [Sign](<https://github.com/bartossh/The-Accountant/blob/main/transaction/transaction.go#L58>)

```go
func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error)
```

Sign signs Transaction by receiver.

## type [Verifier](<https://github.com/bartossh/The-Accountant/blob/main/transaction/transaction.go#L19-L21>)

```go
type Verifier interface {
    Verify(message, signature []byte, hash [32]byte, address string) error
}
```

# wallet

```go
import "github.com/bartossh/The-Accountant/wallet"
```

## Index

- [type Helper](<#type-helper>)
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


## type [Helper](<https://github.com/bartossh/The-Accountant/blob/main/wallet/verifier.go#L13>)

Helper provides wallet helper functionalities without knowing about wallet private and public keys.

```go
type Helper struct{}
```

### func \(Helper\) [AddressToPubKey](<https://github.com/bartossh/The-Accountant/blob/main/wallet/verifier.go#L16>)

```go
func (h Helper) AddressToPubKey(address string) (ed25519.PublicKey, error)
```

AddressToPubKey creates ED25519 public key from address, or returns error otherwise.

### func \(Helper\) [Verify](<https://github.com/bartossh/The-Accountant/blob/main/wallet/verifier.go#L37>)

```go
func (h Helper) Verify(message, signature []byte, hash [32]byte, address string) error
```

Verify verifies if message is signed by given key and hash is equal.

## type [Wallet](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L20-L23>)

Wallet holds public and private key of the wallet owner.

```go
type Wallet struct {
    Private ed25519.PrivateKey
    Public  ed25519.PublicKey
}
```

### func [DecodeGOBWallet](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L35>)

```go
func DecodeGOBWallet(data []byte) (Wallet, error)
```

DecodeGOBWallet tries to decode Wallet from gob representation or returns error otherwise.

### func [New](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L26>)

```go
func New() (Wallet, error)
```

New tries to creates a new Wallet or returns error otherwise.

### func \(\*Wallet\) [Address](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L70>)

```go
func (w *Wallet) Address() string
```

Address creates address from the public key that contains wallet version and checksum.

### func \(\*Wallet\) [ChecksumLength](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L60>)

```go
func (w *Wallet) ChecksumLength() int
```

ChecksumLength returns checksum length.

### func \(\*Wallet\) [EncodeGOB](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L47>)

```go
func (w *Wallet) EncodeGOB() ([]byte, error)
```

EncodeGOB tries to encodes Wallet in to the gob representation or returns error otherwise.

### func \(\*Wallet\) [Sign](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L82>)

```go
func (w *Wallet) Sign(message []byte) (digest [32]byte, signature []byte)
```

Signe signs the message with Ed25519 signature. Returns digest hash sha256 and signature.

### func \(\*Wallet\) [Verify](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L90>)

```go
func (w *Wallet) Verify(message, signature []byte, hash [32]byte) bool
```

Verify verifies message ED25519 signature and hash. Uses hashing sha256.

### func \(\*Wallet\) [Version](<https://github.com/bartossh/The-Accountant/blob/main/wallet/wallet.go#L65>)

```go
func (w *Wallet) Version() byte
```

Version returns wallet version.


