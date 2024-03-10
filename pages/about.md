---
layout: page
title: About The Project
subtitle: ...and a bit about the technology.
---

The Computantis is a set of cloud/edge services that keeps track of transactions between wallets.
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

### Technology

1. The programming language Computantis software is written in is the [Go](https://go.dev/). 
First it was considered to use [RUST](https://www.rust-lang.org/) programming language, but Go features for servers development,
and very good cryptographic library (part of standard library), as well as great concurrency model and performance that
in real life case benchmarks matches the RUST or is not far apart from RUST, convinced me to use the Go language.
2. The repository part of the software is abstracted away, so any database may be used. Computantis first was using the [MongoDB](https://www.mongodb.com/) database but overtime I moved all the logic to use [PostgreSQL](https://www.postgresql.org/).
The reason behind this choice is to keep all the transaction ACID even sacrificing the performance a little. 
Probably the change will be beneficial for blockchain and transaction lookups but I wasn't benchmarking for that so it is a guess.
3. Networking is a very important part of backend solutions. To maintain a speed of execution and development the [Fiber](https://docs.gofiber.io/) framework is used to build the REST API and WebSocket networking.

### Motto

The project motto is the one from a quote by Steve Wozniak: "Wherever smart people work, doors are unlocked."

