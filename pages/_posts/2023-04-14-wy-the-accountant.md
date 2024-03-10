---
layout: post
title: Why a new blockchain technology.
subtitle: What problem this software is solving.
cover-img: /assets/img/14-04-23.png
thumbnail-img: /assets/img/doctor-bag.png
share-img: /assets/img/14-04-23.png
tags: [software, motives, description, explanations]
---

Why a Custom System is Better for Secure Healthcare Validation

There is plenty if not a gazillion blockchains on the market. Why then create a new one?

It is a very good question. Before jumping into a quick answer let me describe the type of problem I am trying to address here.
I was visiting my wife’s family and in a casual chat, I was told a story about a scammer taking advantage of the Polish medical sector. In Poland, a large part of the healthcare system is founded by the government. When a Polish citizen goes to a doctor, hospital or dentist then he/she can receive free health for many medical procedures. After providing medical assistance the healthcare provider to claim money from the government needs to fill out documents and send them to the government agency. The Agency validates claims and provides compensation in money for all valid claims. The problem is that lots of claims are being fabricated. Validating the claims is not an easy job to do, as we have to check who received medical assistance and then validate its legitimacy.
Cryptography offers us not only a secure way of transmitting information by encrypting it but also allows us to sign provided data with a unique signature. 
The idea of creating this piece of a program is to allow for ease of validation of each medical assistance. Treating medical assistance as a transaction we are creating a contract between the issuer - the healthcare provider and the receiver - the person seeking medical care. After medical assistance is provided documents are signed cryptographically by the issuer and receiver and validated by the system. The system knows in an instant about all created transactions and if all signatures are legitimate then the transaction is added to the immutable history repository which is a blockchain.

It is easier said than done. What if someone has no access to the wallet holding cryptographic symmetrical keys? What if someone is not able to provide a signature, what then?

In that case, healthcare providers can use the generic replacement key. That kind of transaction can be reclaimed in the future by substituting transactions with the signature of the receiver. The healthcare provider may receive compensation in money from the healthcare system even when the receiver is unable to sign the transaction. Overuse of a generic replacement key will be easy to track and count. What is more important thou is that it will not be possible to corrupt documents by pretending that someone received medical care. When a generic replacement key is overused then the agency is gaining very important information. Healthcare which issues a lot of documents signed by generic keys is then easily identified and the underlying problem may be addressed efficiently. For example, it might be a real problem caused by homelessness or migrant crises and resources may be properly adjusted just on time.

Understood. Why not use the existing blockchain technology? And what is unique about this problem or solution?

The second question explains the first. Existing blockchains serve the purpose of exchanging tokens that have some value and are built as distributed systems. The scammer problem described here requires a less complex solution where exchanging tokens and accounting for held assets should be neglected. What we need here is a system for asymmetric cryptographically signed transactions that can be validated fast within a centralized blockchain that keeps rewriting past transactions impossible. A new block can only be added, never updated. The peer-to-peer part will be replaced by the centralized blockchain REST API backend and subscribed validators to ensure the system works as expected. The participant’s identity will be kept as a public key that holds no personal data.

So you say the transaction which is a signed document of received healthcare cannot be changed and is only accepted by the system when both; the issuer and receiver signatures are valid, yes?

Exactly.

And the blockchain can be only updated? What if someone will lose the crypto wallet?

The situation when someone will lose the crypto wallet is always possible. It might be the case that the crypto wallet is stolen too. In that case, the responsibility of the crypto wallet holder is to inform the corresponding institution. That institution then will issue a token allowing the creation of a new wallet. Because the only required information is a public key, there is no need to rewrite the blockchain or transactions. All transactions before the wallet change will be valid. From the time the new wallet is created the old wallet will not be valid to sign new transactions. Details on how the process of validating a person and issuing a new wallet will happen are up to the mentioned institution.

Voila.

