---
layout: post
title: Why a new blockchain technology.
subtitle: What problem this software is solving.
cover-img: /assets/img/14-04-23.png
thumbnail-img: /assets/img/blockchain_block.png
share-img: /assets/img/14-04-23.png
tags: [software, motives, description, explonations]
---

There is plenty if not a gazillion blockchains on the market. Why then create a new one?

It is a very good question. Before answering jumping into a quick answer let me describe the type of problem I am trying to address here.
I was visiting my wife’s family and in a casual chat, I was told a story about a scammer taking advantage of the Polish medical sector. In Poland, a large part of the healthcare system is founded by the government. When a Polish citizen goes to a doctor, hospital or dentist for example then he/she can receive free health for many medical procedures. After providing medical assistance the healthcare provider in order to claim money from the government needs to fill out documents and send them to the government agency that validates these claims and provides compensation in money. Problem is that lots of claims are being fabricated. Validating the claims is not an easy job to do, as we have to check who received medical assistance and then validate its legitimacy.
Cryptography offers us not only a secure way of transmitting information by encrypting it but also allows us to sign provided data with a unique signature.  The idea of creating this piece of the program is to allow for ease of validation of each medical assistant. Treating medical assistance as a transaction we are creating a contract between the issuer - the healthcare provider and the receiver - the person seeking medical care. After medical assistance is provided documents are signed cryptographically by the issuer and receiver and validated by the system. A system that knows in instant about all created transactions and all documents can be validated with almost zero cost and time. 

All is good till this point but it is easier said than done. What if someone has no access to the wallet holding cryptographic symmetrical keys? What if someone is unconscious being a victim of a car accident, what then?

In that case, healthcare providers can use the generic replacement key. That kind of transaction can be reclaimed in the future by creating substituting transactions with the signature of the receiver. The healthcare provider may receive compensation in money from the healthcare system even when the receiver is not able to sign the transaction because of death but overuse of a generic replacement key will be seen easily. What is more important thou is that it will not be possible to corrupt documents by pretending that someone received medical care if it’s not being provided or any other fabrication of documents. We are gaining then very important information. Healthcare which issues a lot of documents signed by generic keys is then easily identified and the underlining problem may be addressed efficiently just in time when it is met. For example, it might be a real problem caused by homelessness or migrant crises and resources may be properly adjusted.

Understood. Why not use the existing blockchain technology? And what is unique about this problem or solution?

The second question explains the first. Existing blockchains serve the purpose of exchange of tokens that are having some value and are deeply dependent on being a distributed system. This problem requires a less complex solution where exchanging tokens and accounting for held assets by the participants isn’t required or even should be neglected. What we need here is a system for asymmetric cryptographically signed transactions that can be validated fast and a centralized blockchain that keeps history intact so no one will be able to rewrite past transactions. The only possibility is to update the blockchain. The peer-to-peer part will consist only of the validator’s logic which can be owned by any participant to ensure the whole system works as expected. The participant’s identity is not known to the system as well as the transaction details, which exist in a form of a hash.

So you say the transaction which is a signed document of received healthcare cannot be changed and is only accepted by the system when both; the issuer and receiver signatures are valid, yes? 

Exactly.

And the blockchain can be only updated? What if someone will lose the crypto wallet?

The situation when someone will lose the crypto wallet is always possible. It might be the case that the crypto wallet is stolen too. In that case, this is the responsibility of the crypto wallet to inform the corresponding institution that the wallet is stolen or lost and the wallet will be blocked and a new wallet will be created and assigned to the person claiming the problem. Because of the fact that the only required information is a public key, there is no need of rewriting the blockchain or transactions. All transactions before the wallet change will be valid just from now on the old wallet public key will not be valid to validate new transactions, so it cannot be signed by the old private key. Each new transaction will require a new signature. How the process of issuing a new wallet will happen is up to the government on the institution controlling the system in that matter.

Voila.

This is an oversimplified explanation that omits a lot of important details to keep the subject less technical and more generic, but I think it is a very straightforward and cost-effective solution.

