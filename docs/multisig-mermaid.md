```mermaid
sequenceDiagram
    participant User1 as Key Holder 1
    participant User2 as Key Holder 2
    participant User3 as Key Holder 3
    participant Wallet as Multisig Wallet
    participant Blockchain as Blockchain Network

    Note over User1,User2,User3: Generate and share public keys
    User1->>Wallet: Send public key
    User2->>Wallet: Send public key
    User3->>Wallet: Send public key

    Note over Wallet: Create multisig address (2-of-3)
    Wallet->>Blockchain: Receive funds

    Note over User1,User2,User3: Transaction proposal
    User1->>Wallet: Sign transaction
    User2->>Wallet: Sign transaction

    Note over Wallet: Combine signatures (2-of-3 fulfilled)
    Wallet->>Blockchain: Broadcast transaction

    Blockchain->>Blockchain: Validate transaction
    Blockchain->>Wallet: Update balance
```
