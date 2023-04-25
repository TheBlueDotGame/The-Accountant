---
layout: page
title: About The Project
subtitle: ...and a bit about technology.
---
Hi.
My name is Bartosz Lenart. I build the IoT, Edge and Cloud back-ends and I want to introduce to you “Computantis” solution.


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

### Technology

1. The programming language Computantis software is written in is the [Go](https://go.dev/). 

The choice of programming language was made based on these arguments:
   - Well maintained and trusted cryptography library. In Go it is part of the standard library.
   - Relatively fast language. It is fastest than Java and slower than Rust.
   - Compiled language on many architectures, no virtual machine. It uses LLVM as a compiler backend allowing for a large spectrum of architectures to be run on.
   - Language that is pragmatic and easy to maintain. It is C with GC and abstractions. 
   - Safe language. It has GC, powerful error handling and a great typing system.
   - Language that scales well and has good support for executing concurrent code. It is known to be the language of the cloud and can run thousands of goroutines without performance issues.
   - Tests built into the language. It has it in the standard library. 
   - Paradigm-independent language that will not force OOP or functional programming. Not forcing developers to any style.
   - Good control over memory layout, you can do things like in C, but you can be safer than in Java.
   - Good speed of development. It is known for being very efficient to produce software.
   - Simple and pragmatic composition and packaging system. Great standard library so you do not have to use third-party solutions.


2. The repository part of the software is abstracted away so any database may be used. Nevertheless, Computantis is using the [MongoDB](https://www.mongodb.com/) database and is optimized for this database usage.
3. Networking is a very important part of backend solutions. To maintain a speed of execution and development the [Fiber](https://docs.gofiber.io/) framework is used to build the REST API and WebSocket networking.
4. The mem cache is used for a caching mechanism to avoid unnecessary network I/O calls and allowing for multithreaded access.

### Motto

The project motto is the one from a quote by Steve Wozniak: "Wherever smart people work, doors are unlocked."


