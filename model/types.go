package model

const TaskQueue = "ecommerce-order"

type OrderItem struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type OrderInput struct {
	OrderID string      `json:"order_id"`
	Email   string      `json:"email"`
	Items   []OrderItem `json:"items"`
}

type OrderResult struct {
	OrderID    string `json:"order_id"`
	Status     string `json:"status"`
	TrackingID string `json:"tracking_id,omitempty"`
	Error      string `json:"error,omitempty"`
}

type ShipmentSignal struct {
	Success bool `json:"success"`
}

const ShipmentSignalName = "shipment-signal"
