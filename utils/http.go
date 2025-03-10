package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// WriteJSONResponse writes a JSON response with the given status code and data
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// WriteErrorResponse writes an error response with the given status code and error message
func WriteErrorResponse(w http.ResponseWriter, statusCode int, err error, message string) {
	resp := ErrorResponse{
		Error:   err.Error(),
		Code:    statusCode,
		Message: message,
	}
	WriteJSONResponse(w, statusCode, resp)
}

// DecodeJSONBody decodes the JSON body of a request into the given struct
func DecodeJSONBody(r *http.Request, dst interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %v", err)
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	return nil
}

// CreateHTTPClient creates an HTTP client with default settings
func CreateHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 30,
	}
}

// MakeHTTPRequest makes an HTTP request with the given method, URL, and body
func MakeHTTPRequest(method, url string, body io.Reader) (*http.Response, error) {
	client := CreateHTTPClient()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}
