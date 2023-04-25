---
layout: post
title: Give it an edge.
subtitle: The computantis in the Edge architecture.
cover-img: /assets/img/25-04-23.jpg
thumbnail-img: /assets/img/blockchain_comp.png
share-img: /assets/img/25-04-23.jpg
tags: [software, explanation, edge, cloud, IoT]
---

I work in IoT. So I am attracted to thinking about devices and their communication. Let's then use the power of our imagination and create a hypothetical problem. Humans are good at both; I mean imagination and creating problems, so we should succeed. 
The problem is as follows. We have a set of devices on one end of the edge architecture. They are connected to the mini-hub server that distributes commands and requests to that devices. The mini-hub server we are going to call: the “edge node”.
On the other side of the internet, we have a controller. It might be a mobile application that sends the requests and commands to the devices through the edge node. We will call that application the “edge application”. There can be more than one edge node and one application.

We are sending requests both from the edge application and the edge node through the cloud of applications doing many different things. We have AI analysing the data, the repository, the server to communicate with all the participants and more. All running on many different machines.

The case is to be sure that if any of these middleware machines, services, virtual machines and repositories will be corrupted or overtaken by hackers, our edge node will not allow to execute commands on the devices connected to it or the app will not receive invalid data. 

The other important thing is to have a repository that will hold proof that both parties; the edge node and the edge application agreed on the transaction. Just to remind you that transaction allows to pass the data that are proven to be valid by the issuer and the receiver.

The Computantis solution allows installing wallets on the edge node and the edge application that will take care of sealing the data by the transaction with signatures. The edge node and the edge application will allow only transactions from the set of addresses to be executed on the IoT devices or to be treated as legitimate data on the edge application. 

The central node blockchain will give legitimate proof that both the receiver and issuer of the transaction agreed on processing the data that the transaction holds. This is important if the entity that provides the Edge service needs the security and immutability of transactions for legal cases or any other claims coming from the clients. This allows the clients of the service to set their own validators to ensure the security of data and command execution on the IoT devices.

Of course, we can have someone hacking the edge node or the edge application, but those nodes are present as vertexes of the edge architecture so we can restrict them to receive traffic from as minimal as possible endpoints and ensure proper firewall. The Edge node for example may only receive traffic from a specific server only so the only way to access the edge node from outside is to take over that server. And because the transaction can only be executed by the edge node if the address of the issuer is on the edge node list of allowed addresses, and only if the transaction signature and data digest are valid. So hacking any middleware server will not allow the execution of the malicious command on the edge node. The transaction will not be confirmed by the edge node if corrupted, so the edge application will know that the transaction was corrupted. The corrupted transaction will not be added to the blockchain, it will be kept in awaited transactions repository to be examined. 

Adding the transaction to the blockchain proves ultimately that both the issuer and the receiver agreed that the transmitted transaction is intact. It creates an immutable history of all the transactions confirmed by the issuer and the receiver. The validator node can serve as the whistleblower of any malicious activities in the system. And the fact that the system can have many validators kept by independent wallet owners secures the network.

Lastly, it is important to mention that the central node can be scaled, and the fact of the block being forged without the need for validator nodes' agreement keeps the whole system performant.
