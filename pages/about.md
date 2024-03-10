---
layout: page
title: About The Computantis Project
subtitle: Brief description.
---

# Description 

Computantis is a cloud/edge service designed to track, validate, and facilitate secure token transfers between wallets within a closed ecosystem. It offers a scalable and efficient alternative to traditional blockchains for specific use cases, particularly for organizations that require a more controlled environment.

# Key Features

- Low hardware expectations, run on 1vCPU with 256MB RAM.
- High throughput of 200+ TPS.
- Single node can run with no external dependencies.
- Centralized Trust: Operates on a private, trusted central server ideal for corporations, governments, or agencies.
- Secure Token Transfers: Enables secure and efficient transfer of tokens between authorized participants.
- Data-Centric Transactions (Optional): Tracks data agreed upon by sender and receiver in addition to tokens (optional feature).
- Independent Wallets: Each wallet maintains its own independent transaction history.
- Byzantine Fault Tolerance (BFT) is satisfied. There is no central authority to decide on block forging and instead of hashing signature encapsulations are used.
- Data secured via cryptographic signature of encapsulated data in Direct Acyclic Graph (DAG). Vertex encapsulates transactions, pointing to two parents. The Vertex hash is signed. The transaction is signed by the issuer for token-only transfer and additionally, by the receiver it contains data.
- No GPU, no expensive hashing, nodes are not competing but cooperating.
 
# Transaction Process (data transfer)

- Transaction Creation: The issuer creates a transaction specifying the recipient's address, token amount, and (optionally) additional data. The issuer signs the transaction with their private key and attaches their public key.
- Central Server Validation: The server verifies issuer address, signature, data digest, expiration date, and sufficient token balance.
- Awaiting Transactions: If valid, the transaction goes to a repository awaiting receiver's signature.
- Receiver Signs: The receiver retrieves awaiting transactions, proving their address, and signs with their private key.
- Receiver Approval: If the signature is valid, all awaiting transactions are transferred to the receiver.
- Receiver Signs Approved: The receiver signs approved transactions (including token transfers) and sends them back to the server.
- Validate Transaction against DAG and if correct save in the DAG.
- Gossip Vertex to other nodes.
- Gossip receivers may or may not accept the vertex.

# Transaction Process (data transfer)
- Transaction Creation: The issuer creates a transaction specifying the recipient's address, token amount, and (optionally) additional data. The issuer signs the transaction with their private key and attaches their public key.
- Validate Transaction against DAG and if correct save in the DAG.
- Gossip Vertex to other nodes.
- Gossip receivers may or may not accept the vertex.

# Technology

- Programming Languages: Go (chosen for rich server development features, strong cryptography library, and performance), C to offer library to create embedded wallets. 
- Database: BadgerDB that runs in RAM and can be dropped (backed up) on disk. This allows to run node on a machine we don't want to save data on.
- Docker or K8S support to scale or/and isolate node environment. 

