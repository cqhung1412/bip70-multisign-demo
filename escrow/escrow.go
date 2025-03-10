package escrow

import (
	_ "encoding/json"
	"errors"
	"escrow-service/utils"
	"fmt"
	"log"
	"net/http"
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
}

// RefundRequest represents a request to refund funds from escrow
type RefundRequest struct {
	EscrowID   string `json:"escrow_id"`
	PrivateKey string `json:"private_key"`
	Signature  string `json:"signature"`
}

// Escrow represents an escrow transaction
type Escrow struct {
	ID              string               `json:"id"`
	BuyerPubKey     string               `json:"buyer_pubkey"`
	SellerPubKey    string               `json:"seller_pubkey"`
	EscrowPubKey    string               `json:"escrow_pubkey"`
	MultiSigAddress string               `json:"multisig_address"`
	Amount          int64                `json:"amount"`
	Description     string               `json:"description,omitempty"`
	Status          string               `json:"status"`
	PaymentRequest  utils.PaymentRequest `json:"payment_request"`
	CreatedAt       time.Time            `json:"created_at"`
	ExpiresAt       time.Time            `json:"expires_at"`
	PaymentTxID     string               `json:"payment_txid,omitempty"`
	ReleaseTxID     string               `json:"release_txid,omitempty"`
	RefundTxID      string               `json:"refund_txid,omitempty"`
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
	if req.EscrowID == "" || req.PrivateKey == "" || req.Signature == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing required fields"), "Escrow ID, private key, and signature are required")
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
	if escrow.Status != "funded" {
		utils.WriteErrorResponse(w,
			http.StatusBadRequest,
			errors.New("invalid escrow status"),
			fmt.Sprintf("Escrow status is %s, must be 'funded' to release", escrow.Status))
		return
	}

	// Verify signature (simplified for demo)
	// In a real app, you would verify that the signature is valid and corresponds to an authorized key

	// Create release transaction (simplified for demo)
	releaseTransaction, err := utils.CreateTransaction(
		escrow.MultiSigAddress,
		escrow.SellerPubKey, // This would be an actual address in a real implementation
		escrow.Amount-1000,  // Subtract fee
		req.PrivateKey,
	)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, err, "Failed to create release transaction")
		return
	}

	// Update escrow record
	escrowsMutex.Lock()
	escrow.Status = "released"
	escrow.ReleaseTxID = releaseTransaction.TxID
	escrows[req.EscrowID] = escrow
	escrowsMutex.Unlock()

	log.Printf("Released escrow with ID: %s, TxID: %s", escrow.ID, releaseTransaction.TxID)
	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"escrow_id": escrow.ID,
		"status":    escrow.Status,
		"txid":      releaseTransaction.TxID,
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
	if req.EscrowID == "" || req.PrivateKey == "" || req.Signature == "" {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("missing required fields"), "Escrow ID, private key, and signature are required")
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
	if escrow.Status != "funded" {
		utils.WriteErrorResponse(w,
			http.StatusBadRequest,
			errors.New("invalid escrow status"),
			fmt.Sprintf("Escrow status is %s, must be 'funded' to refund", escrow.Status))
		return
	}

	// Verify signature (simplified for demo)
	// In a real app, you would verify that the signature is valid and corresponds to an authorized key

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

	// Update escrow record
	escrowsMutex.Lock()
	escrow.Status = "refunded"
	escrow.RefundTxID = refundTransaction.TxID
	escrows[req.EscrowID] = escrow
	escrowsMutex.Unlock()

	log.Printf("Refunded escrow with ID: %s, TxID: %s", escrow.ID, refundTransaction.TxID)
	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"escrow_id": escrow.ID,
		"status":    escrow.Status,
		"txid":      refundTransaction.TxID,
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

	// Verify transaction (simplified for demo)
	verified, err := utils.VerifyTransaction(req.TxID)
	if err != nil {
		utils.WriteErrorResponse(w, http.StatusInternalServerError, err, "Failed to verify transaction")
		return
	}

	if !verified {
		utils.WriteErrorResponse(w, http.StatusBadRequest, errors.New("invalid transaction"), "Transaction could not be verified")
		return
	}

	// Update escrow record
	escrowsMutex.Lock()
	escrow.Status = "funded"
	escrow.PaymentTxID = req.TxID
	escrows[req.EscrowID] = escrow
	escrowsMutex.Unlock()

	log.Printf("Payment verified for escrow ID: %s, TxID: %s", escrow.ID, req.TxID)
	utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
		"escrow_id": escrow.ID,
		"status":    escrow.Status,
		"txid":      req.TxID,
	})
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

	utils.WriteJSONResponse(w, http.StatusOK, escrow)
}
