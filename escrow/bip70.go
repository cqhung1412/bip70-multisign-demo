package escrow

import (
	"encoding/json"
	"escrow-service/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// CreateBIP70PaymentRequest creates a BIP70 payment request
func CreateBIP70PaymentRequest(address string, amount int64) (utils.PaymentRequest, error) {
	// Validate input parameters
	if address == "" {
		return utils.PaymentRequest{}, fmt.Errorf("address is required")
	}

	if amount <= 0 {
		return utils.PaymentRequest{}, fmt.Errorf("amount must be positive")
	}

	// Call the utility function to create the payment request
	paymentRequest, err := utils.CreateBIP70PaymentRequest(address, amount)
	if err != nil {
		return utils.PaymentRequest{}, fmt.Errorf("failed to create payment request: %v", err)
	}

	return paymentRequest, nil
}

// CreateCustomBIP70PaymentRequest creates a BIP70 payment request with custom parameters
func CreateCustomBIP70PaymentRequest(address string, amount int64, memo string, expiryHours int) (utils.PaymentRequest, error) {
	// Validate input parameters
	if address == "" {
		return utils.PaymentRequest{}, fmt.Errorf("address is required")
	}

	if amount <= 0 {
		return utils.PaymentRequest{}, fmt.Errorf("amount must be positive")
	}

	// Call the utility function to create the payment request
	paymentRequest, err := utils.CreateBIP70PaymentRequest(address, amount)
	if err != nil {
		return utils.PaymentRequest{}, fmt.Errorf("failed to create payment request: %v", err)
	}

	// Set custom memo if provided
	if memo != "" {
		// Update the memo in the payment details
		details, err := utils.DeserializePaymentDetails(paymentRequest.SerializedDetails)
		if err != nil {
			return utils.PaymentRequest{}, fmt.Errorf("failed to deserialize payment details: %v", err)
		}

		details.Memo = memo

		// Re-serialize the payment details
		serializedDetails, err := utils.SerializePaymentDetails(details)
		if err != nil {
			return utils.PaymentRequest{}, fmt.Errorf("failed to serialize payment details: %v", err)
		}

		paymentRequest.SerializedDetails = serializedDetails
	}

	// Set custom expiry time if provided
	if expiryHours > 0 {
		expiryTime := time.Now().Add(time.Duration(expiryHours) * time.Hour)
		paymentRequest.Expires = expiryTime

		// Update the expiry in the payment details
		details, err := utils.DeserializePaymentDetails(paymentRequest.SerializedDetails)
		if err != nil {
			return utils.PaymentRequest{}, fmt.Errorf("failed to deserialize payment details: %v", err)
		}

		details.Expires = expiryTime.Unix()

		// Re-serialize the payment details
		serializedDetails, err := utils.SerializePaymentDetails(details)
		if err != nil {
			return utils.PaymentRequest{}, fmt.Errorf("failed to serialize payment details: %v", err)
		}

		paymentRequest.SerializedDetails = serializedDetails
	}

	return paymentRequest, nil
}

// VerifyBIP70Payment verifies a BIP70 payment
func VerifyBIP70Payment(paymentRequestID string, txID string) (bool, error) {
	// Validate input parameters
	if paymentRequestID == "" || txID == "" {
		return false, fmt.Errorf("payment request ID and transaction ID are required")
	}

	// LIMITATIONS:
	// This is a simplified implementation with several limitations:
	// - No blockchain connection to verify transaction existence
	// - No verification that transaction pays to the correct address
	// - No validation of payment amount
	// - No confirmation checking
	// - No mempool monitoring for unconfirmed transactions
	// - No actual transaction parsing or script validation
	
	// In a production implementation, you would:
	// 1. Look up the payment request in your database
	// 2. Verify the payment request exists and hasn't already been paid
	// 3. Connect to a Bitcoin node to verify the transaction exists
	// 4. Parse the transaction to ensure it pays to the correct address(es)
	// 5. Verify the payment amount matches the requested amount
	// 6. Check that the transaction has enough confirmations
	// 7. Validate transaction inputs and outputs according to Bitcoin consensus rules

	// For our demo purpose, we'll just check if the txID is valid and in our mock "blockchain"
	verified, err := utils.VerifyTransaction(txID)
	if err != nil {
		return false, fmt.Errorf("transaction verification failed: %v", err)
	}

	return verified, nil
}

// HandlePaymentRequest handles a BIP70 payment request
func HandlePaymentRequest(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the request ID from the URL
	// Expected format: /api/pay/request/{requestID}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid request URL", http.StatusBadRequest)
		return
	}

	// requestID := parts[len(parts)-1]

	// In a real implementation, you would fetch the payment request from a database
	// For demo purposes, we'll create a new one
	address := "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx" // Example testnet address
	amount := int64(100000)                                 // 0.001 BTC in satoshis

	paymentRequest, err := CreateBIP70PaymentRequest(address, amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create payment request: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the correct Content-Type header according to BIP70
	w.Header().Set("Content-Type", "application/bitcoin-paymentrequest")

	// Serialize the payment request
	// In a real implementation, this would use protobuf serialization
	requestBytes, err := json.Marshal(paymentRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize payment request: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(requestBytes)
}

// HandlePayment handles a BIP70 payment message
func HandlePayment(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check Content-Type header
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/bitcoin-payment" {
		http.Error(w, "Invalid Content-Type, expected application/bitcoin-payment", http.StatusUnsupportedMediaType)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	// Parse the payment
	payment, err := utils.DeserializePayment(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse payment: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the payment
	if len(payment.Transactions) == 0 {
		http.Error(w, "Payment contains no transactions", http.StatusBadRequest)
		return
	}

	// In a real implementation, you would:
	// 1. Verify that the transaction satisfies the payment request
	// 2. Broadcast the transaction to the Bitcoin network (if not already done)
	// 3. Update your database with the payment status

	// Create a PaymentACK message
	ack := utils.PaymentACK{
		Payment: *payment,
		Memo:    "Thank you for your payment",
	}

	// Serialize the PaymentACK
	ackBytes, err := utils.SerializePaymentACK(&ack)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize PaymentACK: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the correct Content-Type header according to BIP70
	w.Header().Set("Content-Type", "application/bitcoin-paymentack")

	w.Write(ackBytes)
}

// Utility function to extract transaction details from a BIP70 payment
func extractTransactionFromPayment(payment *utils.Payment) (string, error) {
	if len(payment.Transactions) == 0 {
		return "", fmt.Errorf("payment contains no transactions")
	}

	// In a real implementation, you would parse and validate the transaction
	// For demo purposes, we'll just return a mock transaction ID
	return fmt.Sprintf("tx-%d", time.Now().UnixNano()), nil
}

// ProcessPayment processes a BIP70 payment
func ProcessPayment(paymentData []byte) (*utils.PaymentACK, error) {
	// Parse the payment
	payment, err := utils.DeserializePayment(paymentData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payment: %v", err)
	}

	// Extract transaction details
	txID, err := extractTransactionFromPayment(payment)
	if err != nil {
		return nil, fmt.Errorf("failed to extract transaction: %v", err)
	}

	// Log the transaction (in a real system, you'd store this in a database)
	log.Printf("Received payment with transaction: %s", txID)

	// Create a PaymentACK
	ack := &utils.PaymentACK{
		Payment: *payment,
		Memo:    "Thank you for your payment. Your transaction is being processed.",
	}

	return ack, nil
}
