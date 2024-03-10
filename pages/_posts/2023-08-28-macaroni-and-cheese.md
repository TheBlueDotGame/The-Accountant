---
layout: post
title: Macaroni and cheese?
subtitle: What is a reply attack and how to prevent that?
cover-img: /assets/img/29-04-23.jpg
thumbnail-img: /assets/img/cheese.png
share-img: /assets/img/29-04-23.jpg
tags: [software, security, cryptography, replay attack]
---


This year (2023) The International Association for Cryptologic Research meeting held a talk about ORAM.
Oblivious RAM is a compiler that transforms algorithms, so that the resulting algorithms preserve the input-output behaviour of the original algorithm, rather the distribution of the memory access pattern of the transformed algorithm is independent of the memory access pattern of the original algorithm.
One of the implementations introduced by the researcher during the talk was the MacORAM implementation which contained two distinct features:
1. Mac (message authentication code) - for portions of the schema where the access pattern is time-stampable.
2. AND CHEcking Efficiently and SEcurley - use offline memory checker with amortized 0(1) blow up.

The main reference I would like to make from the Computantis point of view is that the "Mac and Cheese" solution is securing against the replay attack.
A replay attack is a fairly simple but powerful way to make a lot of mess without putting too much effort into system details.
In the case of the system when participants exchange information of security value this may be the single most efficient vector of an attack.

Let me create a hypothetical but very realistic scenario. Imagine you are running a company buying the electricity produced by a house equipped with solar panels. There is a special meter equipped with an encrypted client that sends everything to my company and based on that meter I am paying back to the client for received energy. Even if all messages are encrypted hackers can try to record the conversation when the device was producing a lot of power and then replay back the messages claiming compensation in money for faked measurements. This is one of the examples of when a replay attack can make a huge impact on a company's financial condition. This scam is hard to discover and prove without a proper system. 

The computantis solution seals the transactions making the replay attack impossible.
Each data measurement, or any other packet is embedded within an immutable transaction with a timestamp being part of the transaction digest. This makes each transaction unique, so each one is treated as separate. Repeating the same message many times at once or after some time shows that the message is just a replication and will always be rejected by the central node of the computantis solution  Such messages are not sent to the receiver client and are not part of the blockchain. It saves on traffic to the receiver node and replications are not required to be sealed in the blockchain after being classified as corrupted messages. The computantis can still save or account for those corrupted messages just for legal cases.

The transaction in the computantis solution is only valid after the receiver confirms to agree with the data being received and signs the transaction embedding that data. This allows the receiver to reject any transaction and makes the transaction sender sure that the receiver hasn’t accepted the same transaction a few times. 
The unique transaction is added to the blockchain only if both sides agree, by signing it and cannot be altered in the future which is ensured by the computantis blockchain protocol.

Bon apetti.
