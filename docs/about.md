---
layout: page
title: About The Project
subtitle: ...and a bit about technology.
---

My name is Bartosz Lenart. I build the IoT, Edge and Cloud back-ends and I want to introduce to you “The Accountant” solution.
The Accountant is distributed software allowing for reliable and corruption-resilient accounting and transaction validation.
The Accountant software takes care of transactions between the issuer and receiver. Only issuers and receivers that have public keys in the crated-in system can take part in the transaction. The system doesn’t hold the issuer or receiver's personal information. This anonymises the transaction and keeps the issuer and receiver information private. The transaction requires to be signed cryptographically first by the issuer and then transferred to the receiver. When signed cryptographically by the receiver it goes to the backend and both signatures are validated. Anyone claiming to have a valid document or contract between him and the other person providing that document can be easily validated by checking the document hash. If a transaction with a given hash exists then the document is legitimate.
The Accountant uses hash SHA 256. The signature uses ED25519 elliptic curve asymmetric keys. The public keys of the issuer and the receiver are part of the transaction, whereas the private keys are not known to the accountant's backend and are kept secret by the issuer and the receiver wallets respectively.
When the transaction is validated with success it is saved in a temporary repository collection and hashes are sent as candidates for the next block to be forged. When the block is forged and validators accept the new block (and corresponding transactions) then all the transactions whose hashes belong to the new block are moved into the final transaction collection.
Blockchain and transactions are duplicated along all the validators and stake nodes. But the Accountant system may work as a single node or single stake node with validators.

### Technology

1. The programming language The Accountant software is written in is the [Go](https://go.dev/). The choice of programming language was made based on these arguments:
    - Need for cryptography packages to be part of the standard library, so that it is well tested, maintained, reviewed and will not be dropped or forgotten.
     - Need for relatively fast language, best if compiled to multiple architectures and systems, without an interpreter or virtual machine. Go can be compiled on IoT devices and microcontrollers thanks to [TinyGo](https://tinygo.org/) compiler.
     - Language that can be compiled to [WebAssembly](https://webassembly.org/) to create a  safe wallet for web applications. The TinyGo and Go compilers allow for that.
    - Language that is pragmatic and easy to maintain.
    - Safe language, good if being garbage collectored  or having other safety feature helping with memory control.
    - Language that scales well and has good support for executing concurrent code.
    - A testing feature built into the language.
    - Paradigm-independent language that will not force OOP or functional programming.
    - Good control over memory layout.
    - Good speed of development.
    - Simple and pragmatic composition and packaging system.
 
2. The repository part of the software is abstracted away so any database may be used. Nevertheless, the Accountant is using the [MongoDB](https://www.mongodb.com/) database and is optimised for this database usage.
3. Networking is a very important part of backend solutions. To maintain a speed of execution and development the [Fiber](https://docs.gofiber.io/) framework is used to build the REST API and WebSocket networking.
4. The mem cache is used for a caching mechanism to avoid unnecessary network I/O calls and allowing for multithreaded access.


### Motto

The project motto is the one from a quote by Steve Wozniak: "Wherever smart people work, doors are unlocked."


