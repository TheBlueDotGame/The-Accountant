---
layout: post
title: Why a new blockchain technology.
subtitle: What problem this software is solving.
cover-img: /assets/img/14_04_20.png
thumbnail-img: /assets/img/blockchain_block.png
share-img: /assets/img/14_04_20.png
tags: [software, motives, description, explonations]
---

There is plenty if not a gazillion blockchains on the market. Why then create a new one?
It is a very good question and I think before starting to write software that aims to solve a problem it is the first question to ask. Next will be if our solution is unique or at least have some unique parts that need to be built from the ground up. Before answering those two questions let me introduce where the inspiration came from.

I was visiting my wife's family and in a chat, I was told that in Poland there are scams in the medical sector. Simplifying a bit, the Polish medical sector is founded by the government in a way that when a Polish citizen goes to a doctor, hospital or dentist for example then he/she can receive free health care. After providing medical assistance the healthcare provider in order to claim money from the government needs to fill out documents and send them to the government agency that validates these claims and provides compensation in money. Problem is that lots of claims are being fabricated. Validating the claims is not an easy job to do, as we have to check who received medical assistance and then validate its legitimacy.

So the idea of creating this piece of the program was to allow for ease of validation of each medical assistant. Treating medical assistance as a transaction we are creating a contract between the issuer - the healthcare provider and the receiver - the person seeking medical care. After medical assistance is provided documents are signed cryptographically by the issuer and receiver and validated by the system. A system that knows in instant about all created transactions and all documents can be validated with almost zero cost and time. 
 Yes, it is easier said than done but what if someone has no access to the wallet holding cryptographic symmetrical keys? What if someone is unconscious being a victim of a car accident, what then? Then healthcare providers can use the generic replacement key. That kind of transaction can be reclaimed in the future by creating substituting transactions with the signature of the receiver. The healthcare provider may receive recommendations from the healthcare system even when the receiver is not able to sign the transaction because of death but overuse of a generic replacement key will be seen easily. What is more important thou is that it will not be possible to corrupt documents by pretending that someone received medical care if it's not being provided or any other fabrication of documents.

Ok. Good. Why not use the existing blockchain technology? And what is unique about this problem? The second question explains the first. Existing blockchains serve purpose of exchange of tokens that are having some value and are deeply dependent on being a distributed system, where this problem requires a less complex solution where exchanging tokens and accounting for held assets by the participants isn't required or even should be neglected. What we need here is a system for asymmetric cryptographically signed transactions that can be validated fast and a centralized blockchain that keeps history intact so no one will be able to rewrite the past transaction but rather only updates them. The peer-to-peer part then will consist only of the validator's logic which can be owned by any participant to ensure the whole system works as expected. The participant's identity is not known to the system as well as the transaction details, which exist in a form of a hash. 

Voila.


PS. This is an oversimplified explanation that omits a lot of important details to keep the subject less technical and more generic.

