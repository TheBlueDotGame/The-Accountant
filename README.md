# The Accountant

The accountant is a service that keeps track of transactions between wallets.
Each wallet has its own independent history of transactions. There is a set of rules allowing for transactions to happen.
The accountant is not keeping track of all transactions in a single blockchain but rather allows to keep transactions signed by an authority. A signed transaction is valid transaction only if the issuer and receiver of the transaction are existing within the system.

## Transaction rules

0. The transaction may happen only when signed by the issuer wallet and receiver wallet.
1. Transaction is unique per wallet owner.
2. Transactions are stored per wallet owner in blocks.

## Blocks rules

0. Block is stored in the blockchain that is wallet dependent.
1. Block may store more than one transaction. The max stored transactions limit per block is adjustable.
2. One block per wallet owner may be cleated per 15 minutes (this value is adjustable).
3. Blocks can only be added to the blockchain, and the blockchain cannot be updated.

## Wallet rules

0. The owner may have only one valid wallet.
1. Public key is added to the Owner repository.
2. All historical public keys are stored in the Owner repository.
3. Wallet is not stored in the repository and is kept by the Owner independently from the system.

## The Accountant System

The Accountant system consists of two separate services:

- Backend REST API - keeps track of transactions, validates and stores transactions per user block-chain, and stores users' addresses.
- Client Application - communicates with the Backend REST API, signs the transactions, keeps user Wallet private, allows to regenerate the Wallet in case of Wallet being lost.

