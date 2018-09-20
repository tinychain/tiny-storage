# TinyStorage
Decentralized storage in Tinychain, based on IPFS.

## Why we need decentralized storage
- Centralized storage is always not reliable.
  - The keys to some centralized database are always forgottern, which means we lost these data.
  - Centralized storage service providers always struggle for the commercial competition, and cannot promise to store our data forever.
  - Centralized databases suffer from the hacker attack.
- Make full use of existing network upper-layer infrastructure to store and retrieve data at a low cost.
  - CDN(content delivery network).
  - Node network built for some blockchain projects like Bitcoin and Ethereum.
  
## Decentralized storage in blockchain
In blockchain project, we met some demands that can improve the performance of blockchain:
- Old block data can be archived as single or several compressed files, to release more disk storage. And these archived files don't need storing in every node.
- Some event data made by Oracle need to be stored in blockchain for a certain time. It's the best way to store these kind of data in decentralized storage.
- Blockchain can provide storage service based on **tokens model** as an upper-layer application.

## Examples with Blockchain

### Scenario 1: Trusted File Storage
Alice created a music and wants to sell it through blockchain network. She deploys(or use) a smart contract, which allows buyers to upload their public keys and call the file demand as an order. Then Alice uses these public keys from buyers to encrypt the music file, uploads it to the IPFS, and transfers the content hash to the smart contract. The contract will automatically check the content hash and verify its corresponding file's existence in network. If pass, contract will transfer the token from buyers to Alice's account.

### Scenario 2: Copyright Protection
Alice created a music and wants to declare her copyright. She uploads the file to private IPFS network provided by blockchain, and signs the content hash IPFS returned. Also Alice deploys(or use) a smart contract, and send the signature, content hash of her work, her public key, and **the current timestamp** to the contract. Other people will find the content hash and its signature with earliest timestamp, **verify it with the a list of public key that belongs to different copyright declarers**, and find out which declarer is the real one.

### Scenario 3: Temporary Oracle Event Data

### Scenario 4: Play with IOT(Internet of Things)

## Features
TODO...
  
