package escrow

import (
	"fmt"
	"escrow-service/utils"
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
		paymentRequest.Memo = memo
	}
	
	// Set custom expiry time if provided
	if expiryHours > 0 {
		paymentRequest.Expires = time.Now().Add(time.Duration(expiryHours) * time.Hour)
	}
	
	return paymentRequest, nil
}

// VerifyBIP70Payment verifies a BIP70 payment
func VerifyBIP70Payment(paymentRequestID string, txID string) (bool, error) {
	// Validate input parameters
	if paymentRequestID == "" || txID == "" {
		return false, fmt.Errorf("payment request ID and transaction ID are required")
	}
	
	// Call the utility function to verify the transaction
	verified, err := utils.VerifyTransaction(txID)
	if err != nil {
		return false, fmt.Errorf("failed to verify transaction: %v", err)
	}
	
	return verified, nil
}