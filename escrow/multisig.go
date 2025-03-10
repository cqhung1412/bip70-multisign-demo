package escrow

import (
	"escrow-service/utils"
	"fmt"
)

// CreateMultiSig creates a 2-of-3 multisig address
func CreateMultiSig(buyerPubKey, sellerPubKey, escrowPubKey string) (string, error) {
	// Validate input parameters
	if buyerPubKey == "" || sellerPubKey == "" || escrowPubKey == "" {
		return "", fmt.Errorf("all public keys are required")
	}

	// Call the utility function to create the multisig address
	multiSigAddress, err := utils.CreateMultiSig(buyerPubKey, sellerPubKey, escrowPubKey)
	if err != nil {
		return "", fmt.Errorf("failed to create multisig address: %v", err)
	}

	return multiSigAddress, nil
}

// SignMultiSigTransaction signs a multisig transaction with the provided private key
func SignMultiSigTransaction(txHex string, privateKey string) (string, error) {
	// Validate input parameters
	if txHex == "" || privateKey == "" {
		return "", fmt.Errorf("transaction hex and private key are required")
	}

	// Call the utility function to sign the transaction
	signedTxHex, err := utils.SignTransaction(txHex, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	return signedTxHex, nil
}

// VerifyMultiSigTransaction verifies a multisig transaction
func VerifyMultiSigTransaction(txID string) (bool, error) {
	// Validate input parameter
	if txID == "" {
		return false, fmt.Errorf("transaction ID is required")
	}

	// Call the utility function to verify the transaction
	verified, err := utils.VerifyTransaction(txID)
	if err != nil {
		return false, fmt.Errorf("failed to verify transaction: %v", err)
	}

	return verified, nil
}
