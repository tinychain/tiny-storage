# TinyStorage
Decentralized storage in Tinychain, based on IPFS.

## Why we need decentralized storage
- centralized storage is not reliable
  - The keys to some centralized database are always forgottern, which means we lost these data.
  - Centralized storage service providers always struggle for the commercial competition, and cannot promise to store our data forever.
  - Centralized databases suffer from the hacker attack.
- Make full use of existing network upper-layer infrastructure to store and retrieve data at a low cost.
  - CDN(content delivery network).
  - Node network built for some blockchain projects like Bitcoin and Ethereum.
  
## Decentralized storage in blockchain
In blockchain project, we met some demands that can improve the performance of blockchain:
- Old block data can be archived as single or several compressed files, to release more disk storage. And the archived file don't need storing in every node.
- Some event data made by Oracle need to be stored in blockchain for a certain time. It's the best way to store these kind of data in decentralized storage.
- Blockchain can provide storage service based on **tokens model** as an upper-layer application.

TODO...
  
