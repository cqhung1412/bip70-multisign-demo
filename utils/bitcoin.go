package utils

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

var netParams = &chaincfg.TestNet3Params // Using testnet for demo purposes

// BIP70 message structures
// These structures align with the BIP70 specification
// https://github.com/bitcoin/bips/blob/master/bip-0070.mediawiki

// Output represents a transaction output - where coins are sent to
type Output struct {
	Amount int64  `json:"amount"`
	Script []byte `json:"script"` // Bitcoin script which specifies the conditions to claim this output
}

// PaymentDetails contains the payment details as per BIP70
type PaymentDetails struct {
	Network      string    `json:"network"`       // "main", "test", or "regtest"
	Outputs      []*Output `json:"outputs"`       // Where payment should be sent
	Time         int64     `json:"time"`          // Unix timestamp when created
	Expires      int64     `json:"expires"`       // Unix timestamp when expired
	Memo         string    `json:"memo"`          // Human-readable description of the payment request
	PaymentURL   string    `json:"payment_url"`   // URL to send the Payment message to
	MerchantData []byte    `json:"merchant_data"` // Arbitrary data for merchant's use
}

// PaymentRequest represents a BIP70 payment request
type PaymentRequest struct {
	PaymentDetailsVersion int64         `json:"payment_details_version"` // Should be 1 for BIP70
	PKIType               string        `json:"pki_type"`                // "none", "x509+sha256", etc.
	PKIData               []byte        `json:"pki_data"`                // PKI-dependent data
	SerializedDetails     []byte        `json:"serialized_details"`      // Serialized PaymentDetails
	Signature             []byte        `json:"signature"`               // PKI-dependent signature
	
	// Additional fields for our implementation (not part of BIP70 spec)
	Address               string        `json:"address"`                 // Bitcoin address for simplified use
	Amount                int64         `json:"amount"`                  // Amount in satoshis
	Expires               time.Time     `json:"expires_time"`            // Expiry as time.Time for easier handling
	MerchantID            string        `json:"merchant_id,omitempty"`   // Identifier for the merchant
	RequestID             string        `json:"request_id"`              // Unique identifier for this request
	CallbackURL           string        `json:"callback_url,omitempty"`  // URL for callbacks
}

// Payment represents a BIP70 payment message (from customer to merchant)
type Payment struct {
	MerchantData []byte   `json:"merchant_data"` // From PaymentDetails
	Transactions [][]byte `json:"transactions"`  // Signed transactions that satisfy PaymentDetails
	RefundTo     []*Output `json:"refund_to"`    // Where to send refunds
	Memo         string    `json:"memo"`         // Note from the customer
}

// PaymentACK represents a BIP70 payment acknowledgment (from merchant to customer)
type PaymentACK struct {
	Payment Payment `json:"payment"` // Copy of the Payment message
	Memo    string  `json:"memo"`    // Note from the merchant
}

// Transaction represents a bitcoin transaction
type Transaction struct {
	TxID          string `json:"txid"`
	RawTx         string `json:"raw_tx"`
	Fee           int64  `json:"fee"`
	Confirmations int64  `json:"confirmations"`
}

// CreateMultiSig creates a 2-of-3 multisig address (buyer, seller, escrow service)
func CreateMultiSig(buyerPubKey, sellerPubKey, escrowPubKey string) (string, error) {
	// LIMITATIONS:
	// The current implementation has several limitations:
	// - Only supports 2-of-3 multisig (no custom M-of-N configurations)
	// - Only creates P2SH addresses (no support for P2WSH or P2SH-P2WSH)
	// - Does not return the redeem script which is needed for spending
	// - Does not verify that the public keys are valid secp256k1 public keys
	// - No support for compressed vs uncompressed public key format detection
	// - Fixed to testnet (no support for mainnet or regtest)

	// Decode public keys
	buyerPubKeyBytes, err := hex.DecodeString(buyerPubKey)
	if err != nil {
		return "", fmt.Errorf("invalid buyer public key: %v", err)
	}

	sellerPubKeyBytes, err := hex.DecodeString(sellerPubKey)
	if err != nil {
		return "", fmt.Errorf("invalid seller public key: %v", err)
	}

	escrowPubKeyBytes, err := hex.DecodeString(escrowPubKey)
	if err != nil {
		return "", fmt.Errorf("invalid escrow public key: %v", err)
	}

	// Parse public keys
	defaultNet := &chaincfg.TestNet3Params

	buyerKey, err := btcutil.NewAddressPubKey(buyerPubKeyBytes, defaultNet)
	if err != nil {
		return "", fmt.Errorf("failed to parse buyer public key: %v", err)
	}

	sellerKey, err := btcutil.NewAddressPubKey(sellerPubKeyBytes, defaultNet)
	if err != nil {
		return "", fmt.Errorf("failed to parse seller public key: %v", err)
	}

	escrowKey, err := btcutil.NewAddressPubKey(escrowPubKeyBytes, defaultNet)
	if err != nil {
		return "", fmt.Errorf("failed to parse escrow public key: %v", err)
	}

	// Create multisig script (2 of 3)
	keys := []*btcutil.AddressPubKey{buyerKey, sellerKey, escrowKey}
	script, err := txscript.MultiSigScript(keys, 2)
	if err != nil {
		return "", fmt.Errorf("failed to create multisig script: %v", err)
	}

	// Create P2SH address
	scriptHash, err := btcutil.NewAddressScriptHash(script, netParams)
	if err != nil {
		return "", fmt.Errorf("failed to create script hash: %v", err)
	}

	// NOTE: In a production environment, you would want to:
	// 1. Store the redeem script along with the address
	// 2. Consider using native segwit P2WSH or nested segwit P2SH-P2WSH for lower fees
	// 3. Support different network types (mainnet, testnet, regtest)
	// 4. Add more comprehensive validation of public keys
	// 5. Support custom M-of-N configurations beyond 2-of-3

	return scriptHash.EncodeAddress(), nil
}

// CreateBIP70PaymentRequest creates a BIP70 payment request
func CreateBIP70PaymentRequest(address string, amount int64) (PaymentRequest, error) {
	// Validate the address
	addr, err := btcutil.DecodeAddress(address, netParams)
	if err != nil {
		return PaymentRequest{}, fmt.Errorf("invalid address: %v", err)
	}

	// Get the script for this address
	addrScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return PaymentRequest{}, fmt.Errorf("failed to create output script: %v", err)
	}

	// Create unique request ID
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	
	// Set timestamps
	now := time.Now()
	expiryTime := now.Add(1 * time.Hour) // Default expiry: 1 hour
	
	// Create a payment output
	output := &Output{
		Amount: amount,
		Script: addrScript,
	}
	
	// Create payment details
	details := PaymentDetails{
		Network:    "test", // Using testnet for demo
		Outputs:    []*Output{output},
		Time:       now.Unix(),
		Expires:    expiryTime.Unix(),
		Memo:       "Escrow payment",
		PaymentURL: fmt.Sprintf("http://localhost:8080/api/pay/%s", requestID),
		// MerchantData could contain additional data like order ID, customer info, etc.
		MerchantData: []byte(fmt.Sprintf(`{"order_id": "%s"}`, requestID)),
	}
	
	// Serialize payment details (in a real implementation, this would use protobuf)
	serializedDetails, err := SerializePaymentDetails(&details)
	if err != nil {
		return PaymentRequest{}, fmt.Errorf("failed to serialize payment details: %v", err)
	}
	
	// Create the payment request
	request := PaymentRequest{
		PaymentDetailsVersion: 1, // Version 1 as per BIP70
		PKIType:               "none", // No PKI for simplicity
		PKIData:               nil,    // No PKI data
		SerializedDetails:     serializedDetails,
		Signature:             nil, // No signature for this simple implementation
		
		// Additional fields for our implementation
		Address:               address,
		Amount:                amount,
		Expires:               expiryTime,
		MerchantID:            "EscrowService",
		RequestID:             requestID,
		CallbackURL:           fmt.Sprintf("http://localhost:8080/api/callback/%s", requestID),
	}
	
	return request, nil
}

// SerializePaymentDetails serializes PaymentDetails to JSON bytes
// In a real implementation, this would use protobuf as specified in BIP70
func SerializePaymentDetails(details *PaymentDetails) ([]byte, error) {
	return json.Marshal(details)
}

// DeserializePaymentDetails deserializes PaymentDetails from JSON bytes
func DeserializePaymentDetails(data []byte) (*PaymentDetails, error) {
	var details PaymentDetails
	err := json.Unmarshal(data, &details)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payment details: %v", err)
	}
	return &details, nil
}

// SerializePayment serializes a Payment to JSON bytes
func SerializePayment(payment *Payment) ([]byte, error) {
	return json.Marshal(payment)
}

// DeserializePayment deserializes a Payment from JSON bytes
func DeserializePayment(data []byte) (*Payment, error) {
	var payment Payment
	err := json.Unmarshal(data, &payment)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payment: %v", err)
	}
	return &payment, nil
}

// SerializePaymentACK serializes a PaymentACK to JSON bytes
func SerializePaymentACK(ack *PaymentACK) ([]byte, error) {
	return json.Marshal(ack)
}

// DeserializePaymentACK deserializes a PaymentACK from JSON bytes
func DeserializePaymentACK(data []byte) (*PaymentACK, error) {
	var ack PaymentACK
	err := json.Unmarshal(data, &ack)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payment ack: %v", err)
	}
	return &ack, nil
}

// CreateTransaction creates and signs a new Bitcoin transaction
func CreateTransaction(fromAddress, toAddress string, amount int64, privateKey string) (Transaction, error) {
	// This is a simplified implementation
	// In a real app, you would interact with a full node or service

	// For demo purposes, we'll just return a mock transaction
	txid := fmt.Sprintf("tx-%d", time.Now().UnixNano())

	return Transaction{
		TxID:          txid,
		RawTx:         "01000000...", // Simplified
		Fee:           1000,          // 1000 satoshis
		Confirmations: 0,             // New transaction
	}, nil
}

// knownTransactions is a map of valid transaction IDs for demo purposes
// In a real implementation, you would query the Bitcoin network
var knownTransactions = map[string]bool{
	"26dd4663518b3e24872fd5635fd889a8a0e1c232b8d488868ac378a0a2d28fb1": true, // Example valid transaction
	"3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b": true, // Another example
}

// VerifyTransaction verifies if a transaction is valid and confirmed
func VerifyTransaction(txID string) (bool, error) {
	// Validate input
	if txID == "" {
		return false, errors.New("transaction ID is empty")
	}

	// Validate the transaction ID format (hexadecimal string of the right length)
	if len(txID) != 64 {
		return false, fmt.Errorf("invalid transaction ID format: must be 64 characters, got %d", len(txID))
	}

	for _, c := range txID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false, fmt.Errorf("invalid transaction ID format: must contain only hexadecimal characters")
		}
	}

	// Check if the transaction is in our list of known transactions
	// In a real implementation, you would query a Bitcoin node or API
	// to verify the transaction exists and has enough confirmations
	isValid, exists := knownTransactions[txID]
	if !exists {
		return false, fmt.Errorf("transaction not found in the blockchain")
	}

	return isValid, nil
}

// SignTransaction signs a transaction with the provided private key
func SignTransaction(txHex string, privateKey string) (string, error) {
	// LIMITATIONS:
	// This is a highly simplified mock implementation with many limitations:
	// - No actual cryptographic signing takes place
	// - No validation of the transaction format
	// - No checking that the private key corresponds to an address in the transaction
	// - No support for different signature hash types (SIGHASH_ALL, SIGHASH_SINGLE, etc.)
	// - No segwit support
	// - No P2SH script execution or validation
	// - No handling of different address types
	// - No Partially Signed Bitcoin Transaction (PSBT) support
	
	// In a production implementation, you would:
	// 1. Parse the transaction from hex
	// 2. Identify which inputs need to be signed
	// 3. Verify the private key can sign those inputs
	// 4. For each input:
	//    a. Create the correct signature hash based on the previous output being spent
	//    b. Sign the hash with the private key
	//    c. Create the proper script signature
	//    d. Add the signature to the transaction
	// 5. Verify the signed transaction is valid
	// 6. Consider using the PSBT format for safer signing

	if txHex == "" {
		return "", errors.New("transaction hex is empty")
	}

	if privateKey == "" {
		return "", errors.New("private key is empty")
	}

	// For demo purposes only - this doesn't actually sign anything
	return "signed_" + txHex, nil
}

// CreateRawTransaction creates a raw Bitcoin transaction
func CreateRawTransaction(inputs []wire.TxIn, outputs []wire.TxOut) (*wire.MsgTx, error) {
	// Create a new transaction
	tx := wire.NewMsgTx(wire.TxVersion)

	// Add inputs
	for _, in := range inputs {
		tx.AddTxIn(&in)
	}

	// Add outputs
	for _, out := range outputs {
		tx.AddTxOut(&out)
	}

	return tx, nil
}

// GetTransactionByID retrieves a transaction by ID
func GetTransactionByID(txID string) (Transaction, error) {
	// This is a simplified implementation
	// In a real app, you would query a Bitcoin node or service

	if txID == "" {
		return Transaction{}, errors.New("transaction ID is empty")
	}

	// For demo, we'll just return a mock transaction
	hash, _ := chainhash.NewHashFromStr(txID)

	return Transaction{
		TxID:          hash.String(),
		RawTx:         "01000000...",
		Fee:           1000,
		Confirmations: 6, // Assume 6 confirmations
	}, nil
}
