# The Accountant

The accountant is a service that keeps track of transactions between wallets.
Each wallet has its own independent history of transactions. There is a set of rules allowing for transactions to happen.
The accountant is not keeping track of all transactions in a single blockchain but rather allows to keep transactions signed by an authority. A signed transaction is valid transaction only if the issuer and receiver of the transaction are existing within the system.

## Transaction rules

0. The transaction may happen only when signed by the issuer wallet and receiver wallet.
1. Transaction is unique per whole blockchain.
2. Transactions are stored per wallet owner in blocks.
3. There is a limit of transactions issuer can create per given timespan (configurable).
4. Transaction has expiration time and cannot be signed after expiration time has passed.

## Blocks rules

0. Block is stored in the blockchain.
1. Block may store more than one transaction. The max stored transactions limit per block is adjustable.
2. Blocks can only be added to the blockchain, and the blockchain cannot be updated.
3. Transactions are stored in the block as transactions hash.

## Wallet rules

0. The owner may have only one valid wallet.
1. Public key is added to the Owner repository.
2. All historical public keys are stored in the Owner repository.
3. Wallet is not stored in the repository and is kept by the Owner independently from the system.

## The Accountant System

The Accountant system consists of two separate services:

- Backend REST API - keeps track of transactions, validates and stores transactions per user block-chain, and stores users' addresses.
- Client Application - communicates with the Backend REST API, signs the transactions, keeps user Wallet private, allows to regenerate the Wallet in case of Wallet being lost.

