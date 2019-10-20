# KawAPI

From the Japanese 河 (Kawa), which translates to stream.

KawAPI is a bridge between data consumers and data producers. It is a tool that allows the monetization of any data stream

## Give it a try

Open http://www.kawapi.io/37c1e3be-da52-4f85-b97d-b407639ffe7b/endpoint/molecule/api/project in your browser. You'll get 10 free transactions per session before the server responds that you need to start sending IOTA micro-transactions.

## Inspiration

We believe there are industries with large datasets that have non-trivial value, and no easy way to open up these datasets. We'd like to create a way to value and provide access to these datasets, with dynamic pricing based on the consumer and producer requirements.

When Google gives you a Google Form to request access to paid APIs, that's when you know the current system is broken: https://docs.google.com/forms/d/e/1FAIpQLScK9o4sb96aJMRsK9UUCAiKLjQRJGyrBg2d9XKSL9rOFfpOzg/viewform

## The Tech

![system-diagram](/media/system.png)

KawAPI uses IOTA to perform microtransactions for dynamically priced data streams. It consists of a client CLI that securely manages the transactions from your IOTA wallet, and a server that acts as a reverse proxy between the consumer and the producer, to securely provide access to datasets and APIs.

The server is hosted at kawapi.io, but could be self hosted. Your seed never leaves your side, and all code is open-source. 

## Roadmap

* Implement a Ngrok/Chisel style tunnel to allow easy provision of local datasets.
* Provide interfaces to decentralized datastores, like Ocean, OrbitDB, IPFS, and even IOTA itself.
* Create a marketplace of API endpoints, open for producers to add their endpoints, and consumers to discover endpoints.
