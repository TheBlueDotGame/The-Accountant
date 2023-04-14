---
layout: page
title: About The Project
subtitle: ...and a bit about technology.
---

My name is Bartosz Lenart. I build the IoT, Edge and Cloud back-ends and I want to introduce to you the Accountant software.
This is distributed backend software allowing for reliable and corruption-resilient accounting. 
The Accountant software takes care of transactions between the issuer and receiver. Only known issuers and receivers can take part in the transaction but the Accountant doesn’t hold the issuer or receiver's personal information. This anonymises the transaction and keeps the issuer and receiver information private. The transaction requires to be signed cryptographically first by the issuer and then transferred to the receiver. When signed cryptographically by the receiver it goes to the backend and both signatures are validated.
The signature uses ED25519 elliptic curve asymmetric keys. The public keys of the issuer and the receiver are part of the transaction, whereas the private keys are not known to the accountant's backend and are kept secret by the issuer and the receiver wallets respectively.
When the transaction is validated with success it is saved in a temporary repository collection and hashes are sent as candidates for the next block to be forged. When the block is forged and validators accept the new block (and corresponding transactions) then all the transactions whose hashes belong to the new block are moved into the final transaction collection. 
Blockchain and transactions are duplicated along all the validators and stake nodes. But the Accountant system may work as a single node or single stake node with validators.

### Motto

The project motto is the one from quote of Steve Wozniak: "Wherever smart people work, doors are unlocked."

