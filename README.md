# Escrow Service with BIP70 and MultiSign

This repository contains a full implementation of a Bitcoin escrow service using BIP70 payment protocol and MultiSign addresses, implemented in Golang.

## Introduction

An escrow service acts as a neutral third party to hold funds during a transaction between two parties, ensuring that the payment is only released when both parties fulfill their obligations. This service uses Bitcoin's BIP70 payment protocol for payment requests and MultiSign (2-of-3) addresses for enhanced security, requiring at least two signatures (from buyer, seller, or escrow service) to release funds.

## Features

- Create 2-of-3 MultiSign addresses for escrow transactions
- Generate BIP70 payment requests with customizable parameters
- Verify Bitcoin payments to escrow addresses
- Multi-signature release flow requiring 2 of 3 signatures
- Multi-signature refund flow requiring 2 of 3 signatures
- Tracking of signatures from each party (buyer, seller, escrow)
- Intermediate states for partial signature collection
- In-memory transaction storage (for demo purposes)
- RESTful API with JSON responses

## Sequence Diagram

```mermaid
sequenceDiagram
    participant Buyer
    participant Seller
    participant EscrowService

    Buyer->>EscrowService: Create escrow request
    EscrowService->>MultiSign: Create MultiSign address
    EscrowService->>BIP70: Generate BIP70 payment request
    EscrowService-->>Buyer: Return payment request

    Buyer->>BIP70: Make payment
    BIP70-->>EscrowService: Notify payment received
    Note over EscrowService: Status: "funded"

    alt Happy Path: Release Flow
        Seller->>EscrowService: Request release (first signature)
        Note over EscrowService: Status: "releasing"
        Buyer->>EscrowService: Confirm release (second signature)
        Note over EscrowService: Status: "released"
        EscrowService->>MultiSign: Release funds to Seller (2-of-3 signatures met)
    else Dispute Path: Refund Flow
        Buyer->>EscrowService: Request refund (first signature)
        Note over EscrowService: Status: "refunding"
        EscrowService->>EscrowService: Add escrow signature (second signature)
        Note over EscrowService: Status: "refunded"
        EscrowService->>MultiSign: Refund funds to Buyer (2-of-3 signatures met)
    end
```

## Setup

### Prerequisites

- Go 1.16 or later
- Git

### Clone the repository

```sh
git clone https://github.com/cqhung1412/bip70-multisign-demo.git
cd bip70-multisign-demo
```

### Install dependencies

```sh
go mod tidy
```

## Running the Application

### Starting the server

```sh
go run main.go
```

By default, the server will start on port 8080. You can specify a different port using the `PORT` environment variable:

```sh
PORT=9000 go run main.go
```

### Building the application

```sh
go build -o escrow-service
```

Then run the executable:

```sh
./escrow-service
```

## API Documentation

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/escrow/create` | POST | Create a new escrow transaction |
| `/api/escrow/release` | POST | Release funds from escrow to seller |
| `/api/escrow/refund` | POST | Refund funds from escrow to buyer |
| `/api/escrow/verify-payment` | POST | Verify a payment to an escrow |
| `/api/escrow/get` | GET | Get escrow details by ID |
| `/health` | GET | Health check endpoint |
| `/` | GET | API information |

### Creating an Escrow

Firstly, you need to get the public/private keys for the buyer, seller, and the escrow service. You can get these keys from [privatekeys.pw](https://privatekeys.pw/keys/bitcoin-testnet). You can also click `Random` for random keys.

**Request:**

```sh
curl -X POST http://localhost:8080/api/escrow/create \
  -H "Content-Type: application/json" \
  -d '{
    "buyer_pubkey": "03cd082c25b7f12eed9fba3295c1824148a72440894b42ddca7a73243c9d028f4a",
    "seller_pubkey": "03d70c8915a02010d575a9ae39f7689830822780a606cb6faa4b1d4dbd277240b6",
    "escrow_pubkey": "02a8bee3df56e1362c4db0154b4884a06edcc72e1d421b7c56c694a2df9d8ee867",
    "amount": 100000,
    "description": "Payment for product XYZ"
  }' | jq
```

**Response:**

```json
{
  "id": "escrow-1637142574328",
  "buyer_pubkey": "03cd082c25b7f12eed9fba3295c1824148a72440894b42ddca7a73243c9d028f4a",
  "seller_pubkey": "03d70c8915a02010d575a9ae39f7689830822780a606cb6faa4b1d4dbd277240b6",
  "escrow_pubkey": "02a8bee3df56e1362c4db0154b4884a06edcc72e1d421b7c56c694a2df9d8ee867",
  "multisig_address": "2NCEMwNagVAbbH9oKsNg1GgEcQm6xnKuhCZ",
  "amount": 100000,
  "description": "Payment for product XYZ",
  "status": "created",
  "payment_request": {
    "address": "2NCEMwNagVAbbH9oKsNg1GgEcQm6xnKuhCZ",
    "amount": 100000,
    "memo": "Escrow payment",
    "expires": "2023-11-17T12:49:34Z",
    "payment_url": "http://localhost:8080/pay/req-1637142574328",
    "merchant_id": "EscrowService",
    "request_id": "req-1637142574328",
    "callback_url": "http://localhost:8080/callback/req-1637142574328"
  },
  "created_at": "2023-11-17T11:49:34Z",
  "expires_at": "2023-11-18T11:49:34Z"
}
```

### Verifying Payment

**Request:**

```sh
curl -X POST http://localhost:8080/api/escrow/verify-payment \
  -H "Content-Type: application/json" \
  -d '{
    "escrow_id": "escrow-1637142574328",
    "txid": "abc123def456"
  }'
```

### Releasing Funds

**Request:**

```sh
curl -X POST http://localhost:8080/api/escrow/release \
  -H "Content-Type: application/json" \
  -d '{
    "escrow_id": "escrow-1637142574328",
    "private_key": "private-key-here",
    "signature": "signature-here",
    "party": "seller",
    "public_key": "03d70c8915a02010d575a9ae39f7689830822780a606cb6faa4b1d4dbd277240b6"
  }'
```

**Response (First Signature):**

```json
{
  "escrow_id": "escrow-1637142574328",
  "status": "releasing",
  "signatures_count": 1,
  "signatures_needed": 2,
  "signatures": [
    {
      "party": "seller",
      "signature": "signature-here",
      "timestamp": "2023-11-17T12:15:45Z",
      "public_key": "03d70c8915a02010d575a9ae39f7689830822780a606cb6faa4b1d4dbd277240b6"
    }
  ]
}
```

**Response (After Second Signature):**

```json
{
  "escrow_id": "escrow-1637142574328",
  "status": "released",
  "txid": "release-transaction-id",
  "signatures_count": 2,
  "signatures_needed": 2,
  "signatures": [
    {
      "party": "seller",
      "signature": "signature-here",
      "timestamp": "2023-11-17T12:15:45Z",
      "public_key": "03d70c8915a02010d575a9ae39f7689830822780a606cb6faa4b1d4dbd277240b6"
    },
    {
      "party": "buyer",
      "signature": "second-signature-here",
      "timestamp": "2023-11-17T12:20:30Z",
      "public_key": "03cd082c25b7f12eed9fba3295c1824148a72440894b42ddca7a73243c9d028f4a"
    }
  ]
}
```

### Refunding Funds

**Request:**

```sh
curl -X POST http://localhost:8080/api/escrow/refund \
  -H "Content-Type: application/json" \
  -d '{
    "escrow_id": "escrow-1637142574328",
    "private_key": "private-key-here",
    "signature": "signature-here",
    "party": "buyer",
    "public_key": "03cd082c25b7f12eed9fba3295c1824148a72440894b42ddca7a73243c9d028f4a"
  }'
```

**Response (First Signature):**

```json
{
  "escrow_id": "escrow-1637142574328",
  "status": "refunding",
  "signatures_count": 1,
  "signatures_needed": 2,
  "signatures": [
    {
      "party": "buyer",
      "signature": "signature-here",
      "timestamp": "2023-11-17T13:05:10Z",
      "public_key": "03cd082c25b7f12eed9fba3295c1824148a72440894b42ddca7a73243c9d028f4a"
    }
  ]
}
```

**Response (After Second Signature):**

```json
{
  "escrow_id": "escrow-1637142574328",
  "status": "refunded",
  "txid": "refund-transaction-id",
  "signatures_count": 2,
  "signatures_needed": 2,
  "signatures": [
    {
      "party": "buyer",
      "signature": "signature-here",
      "timestamp": "2023-11-17T13:05:10Z",
      "public_key": "03cd082c25b7f12eed9fba3295c1824148a72440894b42ddca7a73243c9d028f4a"
    },
    {
      "party": "escrow",
      "signature": "escrow-signature-here",
      "timestamp": "2023-11-17T13:10:22Z",
      "public_key": "02a8bee3df56e1362c4db0154b4884a06edcc72e1d421b7c56c694a2df9d8ee867"
    }
  ]
}
```

### Getting Escrow Details

**Request:**

```sh
curl -X GET http://localhost:8080/api/escrow/get?id=escrow-1637142574328 | jq
```

**Response:**

```json
{
  "escrow_id": "escrow-1637142574328",
  "status": "releasing",
  "multisig_address": "2NCEMwNagVAbbH9oKsNg1GgEcQm6xnKuhCZ",
  "amount": 100000,
  "buyer_pubkey": "03cd082c25b7f12eed9fba3295c1824148a72440894b42ddca7a73243c9d028f4a",
  "seller_pubkey": "03d70c8915a02010d575a9ae39f7689830822780a606cb6faa4b1d4dbd277240b6",
  "escrow_pubkey": "02a8bee3df56e1362c4db0154b4884a06edcc72e1d421b7c56c694a2df9d8ee867",
  "created_at": "2023-11-17T11:49:34Z",
  "expires_at": "2023-11-18T11:49:34Z",
  "description": "Payment for product XYZ",
  "payment_request": {
    "address": "2NCEMwNagVAbbH9oKsNg1GgEcQm6xnKuhCZ",
    "amount": 100000,
    "memo": "Escrow payment",
    "expires": "2023-11-17T12:49:34Z",
    "payment_url": "http://localhost:8080/pay/req-1637142574328",
    "merchant_id": "EscrowService",
    "request_id": "req-1637142574328",
    "callback_url": "http://localhost:8080/callback/req-1637142574328"
  },
  "payment_txid": "abc123def456",
  "release_signatures": [
    {
      "party": "seller",
      "signature": "signature-here",
      "timestamp": "2023-11-17T12:15:45Z",
      "public_key": "03d70c8915a02010d575a9ae39f7689830822780a606cb6faa4b1d4dbd277240b6"
    }
  ],
  "release_signatures_count": 1,
  "release_parties": ["seller"]
}
```

## Testing

### Manual Testing

You can use the provided curl commands above to test the API endpoints.

### Automated Testing

To run the test suite:

```sh
go test ./...
```

To run tests with coverage:

```sh
go test ./... -cover
```

To run a specific test:

```sh
go test ./escrow -run TestCreateEscrow
```

## Implementation Notes

- This is a demo implementation, using simplified versions of BIP70 and MultiSign.
- The escrow flow supports the following status transitions:
  - `created` → `funded` → `releasing` → `released`
  - `created` → `funded` → `refunding` → `refunded`
- Multi-signature validation requires 2 of 3 signatures (buyer, seller, escrow) to release or refund funds
- Each party can sign only once for each operation (release or refund)
- In a production environment, you would need to:
  - Connect to a Bitcoin node for transaction validation
  - Implement proper key management and security
  - Use a persistent database instead of in-memory storage
  - Add proper authentication and authorization
  - Implement complete BIP70 protocol including payment ACKs
  - Handle transaction fees properly
  - Add actual signature validation against public keys
