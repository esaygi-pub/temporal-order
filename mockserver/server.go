package mockserver

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

type apiRequest struct {
	OrderID string        `json:"order_id"`
	Email   string        `json:"email,omitempty"`
	Items   []ReserveItem `json:"items,omitempty"`
	Amount  float64       `json:"amount,omitempty"`
}

type apiResponse struct {
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
	TrackingID string  `json:"tracking_id,omitempty"`
	TotalPrice float64 `json:"total_price,omitempty"`
}

func NewMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /inventory/reserve", handleInventoryReserve)
	mux.HandleFunc("POST /inventory/release", handleInventoryRelease)
	mux.HandleFunc("POST /payment/charge", handlePaymentCharge)
	mux.HandleFunc("POST /payment/refund", handlePaymentRefund)
	mux.HandleFunc("POST /email/send", handleEmailSend)
	mux.HandleFunc("POST /shipment/ship", handleShipmentShip)
	mux.HandleFunc("GET /inventory", handleGetInventory)
	mux.HandleFunc("GET /inventory/{id}", handleGetInventoryItem)

	return mux
}

func handleInventoryReserve(w http.ResponseWriter, r *http.Request) {
	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	totalPrice, err := Reserve(req.Items)
	if err != nil {
		fmt.Printf("[mock] inventory/reserve order=%s -> FAIL: %v\n", req.OrderID, err)
		writeJSON(w, http.StatusConflict, apiResponse{Success: false, Message: err.Error()})
		return
	}
	fmt.Printf("[mock] inventory/reserve order=%s items=%v total=$%.2f -> OK\n", req.OrderID, req.Items, totalPrice)
	writeJSON(w, http.StatusOK, apiResponse{Success: true, Message: "inventory reserved", TotalPrice: totalPrice})
}

func handleInventoryRelease(w http.ResponseWriter, r *http.Request) {
	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	Release(req.Items)
	fmt.Printf("[mock] inventory/release order=%s items=%v -> OK\n", req.OrderID, req.Items)
	writeJSON(w, http.StatusOK, apiResponse{Success: true, Message: "inventory released"})
}

func handleGetInventory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, GetInventory())
}

func handleGetInventoryItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, ok := GetItem(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, apiResponse{Success: false, Message: fmt.Sprintf("item %s not found", id)})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func handlePaymentCharge(w http.ResponseWriter, r *http.Request) {
	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	/*
	// 90% random failure, since no re-tries, failure will terminate the workflow
	if rand.Float64() < 0.9 {
		fmt.Printf("[mock] payment/charge order=%s email=%s -> FAIL (random)\n", req.OrderID, req.Email)
		writeJSON(w, http.StatusInternalServerError, apiResponse{Success: false, Message: "Paymnt failed"})
		return
	}
	*/

	fmt.Printf("[mock] payment/charge order=%s -> I am charging $%.2f -> OK\n", req.OrderID, req.Amount)
	writeJSON(w, http.StatusOK, apiResponse{Success: true, Message: fmt.Sprintf("payment charged $%.2f", req.Amount)})
}

func handlePaymentRefund(w http.ResponseWriter, r *http.Request) {
	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("[mock] payment/refund order=%s -> I am refunding $%.2f -> OK\n", req.OrderID, req.Amount)
	writeJSON(w, http.StatusOK, apiResponse{Success: true, Message: fmt.Sprintf("payment refunded $%.2f", req.Amount)})
}

func handleEmailSend(w http.ResponseWriter, r *http.Request) {
	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 422 for known-bad email addresses
	if req.Email == "invalid@nonexistent.test" {
		fmt.Printf("[mock] email/send order=%s email=%s -> 422 (non-retryable)\n", req.OrderID, req.Email)
		writeJSON(w, http.StatusUnprocessableEntity, apiResponse{Success: false, Message: "email address does not exist"})
		return
	}

	// 50% random failure
	if rand.Float64() < 0.5 {
		fmt.Printf("[mock] email/send order=%s email=%s -> FAIL (random)\n", req.OrderID, req.Email)
		writeJSON(w, http.StatusInternalServerError, apiResponse{Success: false, Message: "SMTP server error"})
		return
	}

	fmt.Printf("[mock] email/send order=%s email=%s -> OK\n", req.OrderID, req.Email)
	writeJSON(w, http.StatusOK, apiResponse{Success: true, Message: "email sent"})
}

func handleShipmentShip(w http.ResponseWriter, r *http.Request) {
	var req apiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("[mock] shipment/ship order=%s -> OK\n", req.OrderID)
	writeJSON(w, http.StatusOK, apiResponse{Success: true, Message: "shipment created", TrackingID: fmt.Sprintf("TRK-%s", req.OrderID)})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
