package utils

import (
	"encoding/hex"
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

// PaymentRequest represents a simplified BIP70 payment request
type PaymentRequest struct {
	Address     string    `json:"address"`
	Amount      int64     `json:"amount"`
	Memo        string    `json:"memo,omitempty"`
	Expires     time.Time `json:"expires"`
	PaymentURL  string    `json:"payment_url"`
	MerchantID  string    `json:"merchant_id,omitempty"`
	RequestID   string    `json:"request_id"`
	CallbackURL string    `json:"callback_url,omitempty"`
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

	return scriptHash.EncodeAddress(), nil
}

// CreateBIP70PaymentRequest creates a simplified BIP70 payment request
func CreateBIP70PaymentRequest(address string, amount int64) (PaymentRequest, error) {
	// Validate the address
	_, err := btcutil.DecodeAddress(address, netParams)
	if err != nil {
		return PaymentRequest{}, fmt.Errorf("invalid address: %v", err)
	}

	// Create a payment request with some defaults
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	expiryTime := time.Now().Add(1 * time.Hour) // Default expiry: 1 hour

	return PaymentRequest{
		Address:     address,
		Amount:      amount,
		Memo:        "Escrow payment",
		Expires:     expiryTime,
		PaymentURL:  fmt.Sprintf("http://localhost:8080/pay/%s", requestID),
		MerchantID:  "EscrowService",
		RequestID:   requestID,
		CallbackURL: fmt.Sprintf("http://localhost:8080/callback/%s", requestID),
	}, nil
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

// VerifyTransaction verifies if a transaction is valid and confirmed
func VerifyTransaction(txID string) (bool, error) {
	// This is a simplified implementation
	// In a real app, you would check with a Bitcoin node

	if txID == "" {
		return false, errors.New("transaction ID is empty")
	}

	// For demo, we'll just return true
	return true, nil
}

// SignTransaction signs a transaction with the provided private key
func SignTransaction(txHex string, privateKey string) (string, error) {
	// This is a simplified implementation
	// In a real app, you would use btcd/txscript to sign

	if txHex == "" {
		return "", errors.New("transaction hex is empty")
	}

	if privateKey == "" {
		return "", errors.New("private key is empty")
	}

	// For demo, we'll just return a signed tx hex
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
