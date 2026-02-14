package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.temporal.io/sdk/temporal"
)

const mockServerURL = "http://localhost:8080"

type apiItem struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type apiRequest struct {
	OrderID string    `json:"order_id"`
	Email   string    `json:"email,omitempty"`
	Items   []apiItem `json:"items,omitempty"`
	Amount  float64   `json:"amount,omitempty"`
}

type apiResponse struct {
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
	TrackingID string  `json:"tracking_id,omitempty"`
	TotalPrice float64 `json:"total_price,omitempty"`
}

func ReserveInventory(ctx context.Context, orderID string, items []apiItem) (float64, error) {
	resp, err := callAPIRaw("/inventory/reserve", apiRequest{OrderID: orderID, Items: items})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result apiResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode == http.StatusConflict {
		return 0, temporal.NewNonRetryableApplicationError(
			result.Message, "InventoryUnavailable", fmt.Errorf("%s", result.Message),
		)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("inventory reserve failed: %s", result.Message)
	}
	return result.TotalPrice, nil
}

func ReleaseInventory(ctx context.Context, orderID string, items []apiItem) error {
	_, err := callAPI("/inventory/release", apiRequest{OrderID: orderID, Items: items})
	return err
}

func ChargePayment(ctx context.Context, orderID string, amount float64) error {
	_, err := callAPI("/payment/charge", apiRequest{OrderID: orderID, Amount: amount})
	return err
}

func RefundPayment(ctx context.Context, orderID string, amount float64) error {
	_, err := callAPI("/payment/refund", apiRequest{OrderID: orderID, Amount: amount})
	return err
}

func SendEmail(ctx context.Context, orderID, email string) error {
	resp, err := callAPIRaw("/email/send", apiRequest{OrderID: orderID, Email: email})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result apiResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return temporal.NewNonRetryableApplicationError(
			result.Message, "NonRetryableError", fmt.Errorf("%s", result.Message),
		)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("email send failed: %s", result.Message)
	}
	return nil
}

func ShipOrder(ctx context.Context, orderID string) (string, error) {
	resp, err := callAPI("/shipment/ship", apiRequest{OrderID: orderID})
	if err != nil {
		return "", err
	}
	return resp.TrackingID, nil
}

func callAPI(path string, req apiRequest) (*apiResponse, error) {
	resp, err := callAPIRaw(path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result apiResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API %s failed (%d): %s", path, resp.StatusCode, result.Message)
	}
	return &result, nil
}

func callAPIRaw(path string, req apiRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return http.Post(mockServerURL+path, "application/json", bytes.NewReader(body))
}
