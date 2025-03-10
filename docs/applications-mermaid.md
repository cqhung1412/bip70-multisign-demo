## Application 1: Secure E-Commerce Payments for High-Value Transactions

```mermaid
sequenceDiagram
    participant Customer as Customer
    participant Merchant as Merchant
    participant Escrow as Escrow Service
    participant BitcoinNetwork as Bitcoin Network

    Note over Customer,Merchant: Customer selects item for purchase
    Customer->>Merchant: Initiate payment request
    Merchant->>Customer: Generate BIP70 payment request with multisig address
    Customer->>Customer: Verify merchant identity and payment details
    Customer->>BitcoinNetwork: Send payment to multisig address
    BitcoinNetwork->>Escrow: Funds locked in 2-of-3 multisig escrow
    Escrow->>Escrow: Wait for signatures from customer, merchant, and escrow
    Customer->>Escrow: Sign transaction
    Merchant->>Escrow: Sign transaction
    Escrow->>Escrow: Sign transaction
    Escrow->>BitcoinNetwork: Release funds to merchant
    BitcoinNetwork->>Merchant: Funds received
    Note over Merchant,Customer: Refund initiated if necessary via BIP70 refund address

```

## Application 2: Cryptocurrency Exchange Cold Storage with BIP70 Withdrawals

```mermaid
sequenceDiagram
    participant User as User
    participant Exchange as Exchange
    participant Admin1 as Exchange Admin 1
    participant Admin2 as Exchange Admin 2
    participant BitcoinNetwork as Bitcoin Network

    Note over User,Exchange: User requests withdrawal
    User->>Exchange: Submit BIP70 withdrawal request
    Exchange->>Exchange: Verify request details
    Exchange->>Admin1: Request signature for withdrawal
    Admin1->>Exchange: Sign transaction
    Exchange->>Admin2: Request signature for withdrawal
    Admin2->>Exchange: Sign transaction
    Exchange->>BitcoinNetwork: Broadcast signed transaction
    BitcoinNetwork->>User: Receive funds
    Note over User,Exchange: Withdrawal confirmation sent to user
```

## Application 3: Decentralized Crowdfunding Escrow with Refund Mechanism

```mermaid
sequenceDiagram
    participant Donor as Donor
    participant Platform as Crowdfunding Platform
    participant Validator1 as Validator 1
    participant Validator2 as Validator 2
    participant ProjectCreator as Project Creator
    participant BitcoinNetwork as Bitcoin Network

    Note over Donor,Platform: Donor contributes to project via BIP70
    Donor->>BitcoinNetwork: Send funds to 3-of-5 multisig escrow
    BitcoinNetwork->>Platform: Funds locked in escrow
    Platform->>Validator1: Notify validators of project progress
    Validator1->>Platform: Verify project milestones
    Validator2->>Platform: Verify project milestones
    Platform->>ProjectCreator: Request signatures for fund release
    Validator1->>Platform: Sign transaction
    Validator2->>Platform: Sign transaction
    Platform->>BitcoinNetwork: Release funds to project creator
    BitcoinNetwork->>ProjectCreator: Receive funds
    Note over Platform,Donor: Refund initiated if milestones missed
```

## Application 4: Institutional Payroll Management with Time-Locked Multisig Transactions

```mermaid
sequenceDiagram
    participant HR as HR Department
    participant Finance as Finance Department
    participant Compliance as Compliance Department
    participant Employee as Employee
    participant BitcoinNetwork as Bitcoin Network

    Note over HR,Finance: HR generates BIP70 payment requests
    HR->>Finance: Send structured payment details
    Finance->>Compliance: Request approval and signature
    Compliance->>Finance: Sign transaction
    Finance->>Finance: Sign transaction
    Finance->>BitcoinNetwork: Broadcast time-locked multisig transaction
    BitcoinNetwork->>Employee: Receive salary on scheduled payday
    Note over Employee,HR: Payroll system updated with transaction confirmation

```
