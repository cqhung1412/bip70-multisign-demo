package main

import (
	"escrow-service/escrow"
	"escrow-service/utils"
	"log"
	"net/http"
	"os"
)

// Define API routes
func setupRoutes() {
	// Escrow API endpoints
	http.HandleFunc("/api/escrow/create", escrow.CreateEscrow)
	http.HandleFunc("/api/escrow/release", escrow.ReleaseEscrow)
	http.HandleFunc("/api/escrow/refund", escrow.RefundEscrow)
	http.HandleFunc("/api/escrow/verify-payment", escrow.VerifyPayment)
	http.HandleFunc("/api/escrow/get", escrow.GetEscrow)

	// BIP70 Payment Protocol endpoints
	http.HandleFunc("/api/pay/request/", escrow.HandlePaymentRequest) // endpoint for getting payment requests
	http.HandleFunc("/api/pay/", escrow.HandlePayment)                // endpoint for receiving payments

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSONResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// 404 handler for undefined routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			utils.WriteErrorResponse(w, http.StatusNotFound,
				os.ErrNotExist, "Endpoint not found")
			return
		}

		// Root endpoint returns API info
		utils.WriteJSONResponse(w, http.StatusOK, map[string]interface{}{
			"name":        "Escrow Service API",
			"version":     "1.0.0",
			"description": "A Bitcoin escrow service using BIP70 and MultiSign",
			"endpoints": []string{
				// Escrow endpoints
				"/api/escrow/create",
				"/api/escrow/release",
				"/api/escrow/refund",
				"/api/escrow/verify-payment",
				"/api/escrow/get",
				// BIP70 endpoints
				"/api/pay/request/{requestID}",
				"/api/pay/{requestID}",
				"/health",
			},
		})
	})
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Set up routes
	setupRoutes()

	// Set up middleware
	handler := corsMiddleware(loggingMiddleware(http.DefaultServeMux))

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
