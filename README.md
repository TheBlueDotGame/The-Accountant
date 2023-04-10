# The-Accountant

The accountant is a service that keeps track of transactions between wallets.
Each wallet has its own independent history of transactions. There is a set of rules allowing for transactions to happen. 

## Transaction rules

0. Transaction may happen only when signed by issuer wallet and receiver wallet.
1. Transaction is unique per wallet owner.
2. Transactions are stored per wallet owner in blocks. 

## Blocks rules

0. Block are stored in the blockchain that is wallet dependent.
1. Block may store more then one transaction. The max stored transactions limit per block is ajustable.
2. One block per wallet owner may be cleated per 15 minutes (this value is ajustable).


## Wallet rules

0. Owner may have only one valid wallet.
1. Public key is added to the Owner repository.
2. All historical public keys are stored in the Owner repository.
3. Wallet is not stored in the repository and is kept by the Owner independently from the system.

## The Accountant System

The Accountant system consists of two separate services:

- Backend REST API - keeps track of transactions, validates and stores transactions. It stores private keys per Owner and keeps Owners accounts. Backend ensures that lost wallet is not valid anymore.
- Client Application - comunicates with the backend and signs the transactions, keeps user Wallet private, allows to regenerate the Wallet in case of losing it. 


