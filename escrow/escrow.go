package escrow

import (
	_ "encoding/json"
	"errors"
	"escrow-service/utils"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// In-memory database for demo purposes
var (
	escrowsMutex sync.RWMutex
	escrows      = make(map[string]*Escrow)
)

// EscrowRequest represents a request to create an escrow transaction
type EscrowRequest struct {
	BuyerPubKey  string `json:"buyer_pubkey"`
	SellerPubKey string `json:"seller_pubkey"`
	EscrowPubKey string `json:"escrow_pubkey"`
	Amount       int64  `json:"amount"`
	Description  string `json:"description,omitempty"`
	ExpiryHours  int    `json:"expiry_hours,omitempty"`
}

// ReleaseRequest represents a request to release funds from escrow
type ReleaseRequest struct {
	EscrowID   string `json:"escrow_id"`
	PrivateKey string `json:"private_key"`
	Signature  string `json:"signature"`
	Party      string `json:"party"` // "buyer", "seller", or "escrow"
	PublicKey  string `json:"public_key"`
}

// RefundRequest represents a request to refund funds from escrow
type RefundRequest struct {
	EscrowID   string `json:"escrow_id"`
	PrivateKey string `json:"private_key"`
	Signature  string `json:"signature"`
	Party      string `json:"party"` // "buyer", "seller", or "escrow"
	PublicKey  string `json:"public_key"`
}

// PartySignature represents a signature from a party
type PartySignature struct {
	Party     string    `json:"party"` // "buyer", "seller", or "escrow"
	Signature string    `json:"signature"`
	Timestamp time.Time `json:"timestamp"`
	PublicKey string    `json:"public_key"`
}

// Escrow represents an escrow transaction
type Escrow struct {
	ID                string               `json:"id"`
	BuyerPubKey       string               `json:"buyer_pubkey"`
	SellerPubKey      string               `json:"seller_pubkey"`
	EscrowPubKey      string               `json:"escrow_pubkey"`
	MultiSigAddress   string               `json:"multisig_address"`
	Amount            int64                `json:"amount"`
	Description       string               `json:"description,omitempty"`
	Status            string               `json:"status"`
	PaymentRequest    utils.PaymentRequest `json:"payment_request"`
	CreatedAt         time.Time            `json:"created_at"`
	ExpiresAt         time.Time            `json:"expires_at"`
	PaymentTxID       string               `json:"payment_txid,omitempty"`
	ReleaseTxID       string               `json:"release_txid,omitempty"`
	RefundTxID        string               `json:"refund_txid,omitempty"`
	ReleaseSignatures []PartySignature     `json:"release_signatures,omitempty"`
	RefundSignatures  []PartySignature     `json:"refund_signatures,omitempty"`
}

// CreateEscrow creates a new escrow transaction
func CreateEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, errors.New("method not allowed"), "Only POST method is allowed")
		return
	}

	var req EscrowRequest
	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, err, "Invalid request payload")
		return
	}

	// Validate request
	if req.BuyerPubKey == "" || req.SellerPubKey == "" || req.EscrowPubKey == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing required fields"), "Buyer, seller, and escrow public keys are required")
		return
	}

	if req.Amount <= 0 {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("invalid amount"), "Amount must be positive")
		return
	}

	// Create MultiSig address
	multiSigAddress, err := utils.CreateMultiSig(req.BuyerPubKey, req.SellerPubKey, req.EscrowPubKey)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, err, "Failed to create MultiSig address")
		return
	}

	// Create BIP70 payment request
	paymentRequest, err := utils.CreateBIP70PaymentRequest(multiSigAddress, req.Amount)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, err, "Failed to create BIP70 payment request")
		return
	}

	// Set expiry time if provided
	expiryHours := 24 // Default: 24 hours
	if req.ExpiryHours > 0 {
		expiryHours = req.ExpiryHours
	}
	expiryTime := time.Now().Add(time.Duration(expiryHours) * time.Hour)

	// Create escrow record
	escrow := &Escrow{
		ID:              fmt.Sprintf("escrow-%d", time.Now().UnixNano()),
		BuyerPubKey:     req.BuyerPubKey,
		SellerPubKey:    req.SellerPubKey,
		EscrowPubKey:    req.EscrowPubKey,
		MultiSigAddress: multiSigAddress,
		Amount:          req.Amount,
		Description:     req.Description,
		Status:          "created",
		PaymentRequest:  paymentRequest,
		CreatedAt:       time.Now(),
		ExpiresAt:       expiryTime,
	}

	// Store in "database"
	escrowsMutex.Lock()
	escrows[escrow.ID] = escrow
	escrowsMutex.Unlock()

	log.Printf("Created escrow with ID: %s", escrow.ID)
	utils.WriteJSONResponse(w, http.StatusCreated, escrow)
}

// ReleaseEscrow releases funds from escrow to the seller
func ReleaseEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, errors.New("method not allowed"), "Only POST method is allowed")
		return
	}

	var req ReleaseRequest
	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, err, "Invalid request payload")
		return
	}

	// Validate request
	if req.EscrowID == "" || req.PrivateKey == "" || req.Signature == "" || req.Party == "" || req.PublicKey == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing required fields"),
			"Escrow ID, private key, signature, party type, and public key are required")
		return
	}

	// Validate party type
	if req.Party != "buyer" && req.Party != "seller" && req.Party != "escrow" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("invalid party type"),
			"Party must be one of: buyer, seller, or escrow")
		return
	}

	// Get escrow from "database"
	escrowsMutex.RLock()
	escrow, exists := escrows[req.EscrowID]
	escrowsMutex.RUnlock()

	if !exists {
		utils.WriteErrorResponse(w, http.StatusNotFound, errors.New("escrow not found"), "Escrow with the specified ID does not exist")
		return
	}

	// Check escrow status
	if escrow.Status != "funded" && escrow.Status != "releasing" {
		utils.WriteErrorResponse(w,
			http.StatusBadRequest,
			errors.New("invalid escrow status"),
			fmt.Sprintf("Escrow status is %s, must be 'funded' or 'releasing' to process release request", escrow.Status))
		return
	}

	// LIMITATION: No cryptographic signature verification
	// In a production implementation:
	// 1. Verify that the signature is cryptographically valid for the transaction
	// 2. Verify that the public key matches the one provided during escrow creation
	// 3. Ensure the signature covers the correct transaction data
	// 4. Validate the signature against Bitcoin consensus rules

	// Check if this party has already signed
	for _, sig := range escrow.ReleaseSignatures {
		if sig.Party == req.Party {
			utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("duplicate signature"),
				fmt.Sprintf("A signature from %s has already been provided", req.Party))
			return
		}
	}

	// Create new signature record
	newSignature := PartySignature{
		Party:     req.Party,
		Signature: req.Signature,
		Timestamp: time.Now(),
		PublicKey: req.PublicKey,
	}

	// Update escrow record with new signature
	escrowsMutex.Lock()
	defer escrowsMutex.Unlock()

	// Add the signature
	escrow.ReleaseSignatures = append(escrow.ReleaseSignatures, newSignature)

	// Check if we have reached the 2-of-3 threshold
	if len(escrow.ReleaseSignatures) >= 2 {
		// LIMITATION: Simplified transaction creation
		// In a production implementation:
		// 1. Construct a proper Bitcoin transaction with correct inputs and outputs
		// 2. Use UTXO management to track available funds
		// 3. Apply proper fee estimation based on transaction size and network conditions
		// 4. Create and sign a proper multisig transaction using the redeem script
		// 5. Broadcast the transaction to the Bitcoin network
		// 6. Use Partially Signed Bitcoin Transactions (PSBT) for more robust handling
		
		// Create release transaction (simplified for demo)
		releaseTransaction, err := utils.CreateTransaction(
			escrow.MultiSigAddress,
			escrow.SellerPubKey, // This would be an actual address in a real implementation
			escrow.Amount-1000,  // Subtract fee (fixed fee, not dynamic)
			req.PrivateKey,
		)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err, "Failed to create release transaction")
			return
		}

		// Update to released status
		escrow.Status = "released"
		escrow.ReleaseTxID = releaseTransaction.TxID
		log.Printf("Released escrow with ID: %s, TxID: %s", escrow.ID, releaseTransaction.TxID)
	} else {
		// Update to releasing status
		escrow.Status = "releasing"
		log.Printf("Added release signature for escrow ID: %s from %s", escrow.ID, req.Party)
	}

	// Save the updated escrow
	escrows[req.EscrowID] = escrow

	// Response
	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"escrow_id":         escrow.ID,
		"status":            escrow.Status,
		"txid":              escrow.ReleaseTxID,
		"signatures_count":  len(escrow.ReleaseSignatures),
		"signatures_needed": 2,
		"signatures":        escrow.ReleaseSignatures,
	})
}

// RefundEscrow refunds funds from escrow to the buyer
func RefundEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, errors.New("method not allowed"), "Only POST method is allowed")
		return
	}

	var req RefundRequest
	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, err, "Invalid request payload")
		return
	}

	// Validate request
	if req.EscrowID == "" || req.PrivateKey == "" || req.Signature == "" || req.Party == "" || req.PublicKey == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing required fields"),
			"Escrow ID, private key, signature, party type, and public key are required")
		return
	}

	// Validate party type
	if req.Party != "buyer" && req.Party != "seller" && req.Party != "escrow" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("invalid party type"),
			"Party must be one of: buyer, seller, or escrow")
		return
	}

	// Get escrow from "database"
	escrowsMutex.RLock()
	escrow, exists := escrows[req.EscrowID]
	escrowsMutex.RUnlock()

	if !exists {
		utils.WriteErrorResponse(w, http.StatusNotFound, errors.New("escrow not found"), "Escrow with the specified ID does not exist")
		return
	}

	// Check escrow status
	if escrow.Status != "funded" && escrow.Status != "refunding" {
		utils.WriteErrorResponse(w,
			http.StatusBadRequest,
			errors.New("invalid escrow status"),
			fmt.Sprintf("Escrow status is %s, must be 'funded' or 'refunding' to process refund request", escrow.Status))
		return
	}

	// Verify signature (simplified for demo)
	// In a real app, you would verify that the signature is valid and corresponds to an authorized key

	// Check if this party has already signed
	for _, sig := range escrow.RefundSignatures {
		if sig.Party == req.Party {
			utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("duplicate signature"),
				fmt.Sprintf("A signature from %s has already been provided", req.Party))
			return
		}
	}

	// Create new signature record
	newSignature := PartySignature{
		Party:     req.Party,
		Signature: req.Signature,
		Timestamp: time.Now(),
		PublicKey: req.PublicKey,
	}

	// Update escrow record with new signature
	escrowsMutex.Lock()
	defer escrowsMutex.Unlock()

	// Add the signature
	escrow.RefundSignatures = append(escrow.RefundSignatures, newSignature)

	// Check if we have reached the 2-of-3 threshold
	if len(escrow.RefundSignatures) >= 2 {
		// Create refund transaction (simplified for demo)
		refundTransaction, err := utils.CreateTransaction(
			escrow.MultiSigAddress,
			escrow.BuyerPubKey, // This would be an actual address in a real implementation
			escrow.Amount-1000, // Subtract fee
			req.PrivateKey,
		)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err, "Failed to create refund transaction")
			return
		}

		// Update to refunded status
		escrow.Status = "refunded"
		escrow.RefundTxID = refundTransaction.TxID
		log.Printf("Refunded escrow with ID: %s, TxID: %s", escrow.ID, refundTransaction.TxID)
	} else {
		// Update to refunding status
		escrow.Status = "refunding"
		log.Printf("Added refund signature for escrow ID: %s from %s", escrow.ID, req.Party)
	}

	// Save the updated escrow
	escrows[req.EscrowID] = escrow

	// Response
	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"escrow_id":         escrow.ID,
		"status":            escrow.Status,
		"txid":              escrow.RefundTxID,
		"signatures_count":  len(escrow.RefundSignatures),
		"signatures_needed": 2,
		"signatures":        escrow.RefundSignatures,
	})
}

// VerifyPayment verifies a payment to an escrow
func VerifyPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, errors.New("method not allowed"), "Only POST method is allowed")
		return
	}

	var req struct {
		EscrowID string `json:"escrow_id"`
		TxID     string `json:"txid"`
	}

	if err := utils.DecodeJSONBody(r, &req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, err, "Invalid request payload")
		return
	}

	// Validate request
	if req.EscrowID == "" || req.TxID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing required fields"), "Escrow ID and transaction ID are required")
		return
	}

	// Get escrow from "database"
	escrowsMutex.RLock()
	escrow, exists := escrows[req.EscrowID]
	escrowsMutex.RUnlock()

	if !exists {
		utils.WriteErrorResponse(w, http.StatusNotFound, errors.New("escrow not found"), "Escrow with the specified ID does not exist")
		return
	}

	// Check if payment is already verified for this escrow
	if escrow.Status == "funded" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("payment already verified"),
			fmt.Sprintf("Escrow ID: %s already has a verified payment with txID: %s", escrow.ID, escrow.PaymentTxID))
		return
	}

	// Verify transaction against our validation rules and "blockchain"
	verified, err := utils.VerifyTransaction(req.TxID)
	if err != nil {
		// Provide specific error from transaction verification
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			// Transaction not found in our known transactions list
			code = http.StatusNotFound
		}
		utils.WriteErrorResponse(w, code, err, fmt.Sprintf("Transaction verification failed: %v", err))
		return
	}

	if !verified {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("invalid transaction"),
			"Transaction could not be verified - it may be invalid or have insufficient confirmations")
		return
	}

	// Update escrow record
	escrowsMutex.Lock()
	escrow.Status = "funded"
	escrow.PaymentTxID = req.TxID
	escrows[req.EscrowID] = escrow
	escrowsMutex.Unlock()

	log.Printf("Payment verified for escrow ID: %s, TxID: %s", escrow.ID, req.TxID)

	// Create comprehensive response with all details
	response := map[string]interface{}{
		"escrow_id":        escrow.ID,
		"status":           escrow.Status,
		"payment_txid":     req.TxID,
		"multisig_address": escrow.MultiSigAddress,
		"amount":           escrow.Amount,
		"buyer_pubkey":     escrow.BuyerPubKey,
		"seller_pubkey":    escrow.SellerPubKey,
		"escrow_pubkey":    escrow.EscrowPubKey,
		"created_at":       escrow.CreatedAt,
		"expires_at":       escrow.ExpiresAt,
	}

	// Add signatures information if any exists
	if len(escrow.ReleaseSignatures) > 0 {
		response["release_signatures"] = escrow.ReleaseSignatures
		response["release_signatures_count"] = len(escrow.ReleaseSignatures)
	}

	if len(escrow.RefundSignatures) > 0 {
		response["refund_signatures"] = escrow.RefundSignatures
		response["refund_signatures_count"] = len(escrow.RefundSignatures)
	}

	utils.WriteJSONResponse(w, http.StatusOK, response)
}

// GetEscrow gets an escrow by ID
func GetEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteErrorResponse(w, http.StatusMethodNotAllowed, errors.New("method not allowed"), "Only GET method is allowed")
		return
	}

	// Get escrow ID from URL query parameter
	escrowID := r.URL.Query().Get("id")
	if escrowID == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing escrow ID"), "Escrow ID is required")
		return
	}

	// Get escrow from "database"
	escrowsMutex.RLock()
	escrow, exists := escrows[escrowID]
	escrowsMutex.RUnlock()

	if !exists {
		utils.WriteErrorResponse(w, http.StatusNotFound, errors.New("escrow not found"), "Escrow with the specified ID does not exist")
		return
	}

	// Create comprehensive response with all details
	response := map[string]interface{}{
		"escrow_id":        escrow.ID,
		"status":           escrow.Status,
		"multisig_address": escrow.MultiSigAddress,
		"amount":           escrow.Amount,
		"buyer_pubkey":     escrow.BuyerPubKey,
		"seller_pubkey":    escrow.SellerPubKey,
		"escrow_pubkey":    escrow.EscrowPubKey,
		"created_at":       escrow.CreatedAt,
		"expires_at":       escrow.ExpiresAt,
		"description":      escrow.Description,
		"payment_request":  escrow.PaymentRequest,
	}

	// Add transaction IDs if they exist
	if escrow.PaymentTxID != "" {
		response["payment_txid"] = escrow.PaymentTxID
	}

	if escrow.ReleaseTxID != "" {
		response["release_txid"] = escrow.ReleaseTxID
	}

	if escrow.RefundTxID != "" {
		response["refund_txid"] = escrow.RefundTxID
	}

	// Add signatures information if any exists
	if len(escrow.ReleaseSignatures) > 0 {
		response["release_signatures"] = escrow.ReleaseSignatures
		response["release_signatures_count"] = len(escrow.ReleaseSignatures)

		// For convenience, list of parties that have signed
		parties := make([]string, 0, len(escrow.ReleaseSignatures))
		for _, sig := range escrow.ReleaseSignatures {
			parties = append(parties, sig.Party)
		}
		response["release_parties"] = parties
	}

	if len(escrow.RefundSignatures) > 0 {
		response["refund_signatures"] = escrow.RefundSignatures
		response["refund_signatures_count"] = len(escrow.RefundSignatures)

		// For convenience, list of parties that have signed
		parties := make([]string, 0, len(escrow.RefundSignatures))
		for _, sig := range escrow.RefundSignatures {
			parties = append(parties, sig.Party)
		}
		response["refund_parties"] = parties
	}

	utils.WriteJSONResponse(w, http.StatusOK, response)
}
