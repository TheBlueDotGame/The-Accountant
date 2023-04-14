---
layout: page
title: About The Project
subtitle: Explain me like for 10 years old.
---

My name is Bartosz Lenart. I build the IoT, Edge and Cloud back-ends. This is distributed backend software allowing for reliable and corruption resilient accounting. 
This project aims to be a simple package that offers full capability of setting accounting backend and fronted with single library usage, or to use that library to build your own solution on top of the package. Packages are decoupled to the point that custom solution can be easily used.
Backend then is build fully on [Go](https://go.dev/) programming language environment, where the frontend is build using Go and [WebAssembly](https://webassembly.org/) technology.


### Deeper in to Technology
 
The project is coded in [Go](https://go.dev/) with addition of [WebAssembly](https://webassembly.org/). It is recommended to compile WebAssembly part with [TinyGo](https://tinygo.org/) compiler to create the smaller binary.
The repository used for the project is MongoDB but logic depends on abstraction so other solution can be provided. MongoDB has been chosen because of its document based nature and no need for setting schema upfront. Nevertheless, the indexing is required as uniqueness and lack of transaction repetition is dependent on setting unique indexes in the repository. 


### Motto

The project motto is the one from quote of Steve Wozniak: "Wherever smart people work, doors are unlocked."

