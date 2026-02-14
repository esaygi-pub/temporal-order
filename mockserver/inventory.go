package mockserver

import (
	"fmt"
	"sync"
)

type InventoryItem struct {
	ID       string  `json:"id"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

var (
	invMu     sync.Mutex
	inventory = map[string]*InventoryItem{
		"ITEM-001": {ID: "ITEM-001", Price: 29.99, Quantity: 100},
		"ITEM-002": {ID: "ITEM-002", Price: 49.99, Quantity: 50},
		"ITEM-003": {ID: "ITEM-003", Price: 9.99, Quantity: 200},
	}
)

type ReserveItem struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

// Reserve decrements inventory for the given items. Returns an error if any
// item is unknown or has insufficient stock. On error, no quantities are changed.
// Reserve decrements inventory and returns the total price (quantity * unit price).
func Reserve(items []ReserveItem) (float64, error) {
	invMu.Lock()
	defer invMu.Unlock()

	// Validate all items first
	for _, item := range items {
		inv, ok := inventory[item.ItemID]
		if !ok {
			return 0, fmt.Errorf("item %s not found", item.ItemID)
		}
		if inv.Quantity < item.Quantity {
			return 0, fmt.Errorf("item %s: requested %d but only %d available", item.ItemID, item.Quantity, inv.Quantity)
		}
	}

	// All valid — apply and calculate total
	var total float64
	for _, item := range items {
		total += inventory[item.ItemID].Price * float64(item.Quantity)
		inventory[item.ItemID].Quantity -= item.Quantity
	}
	return total, nil
}

// Release increments inventory for the given items (compensation).
func Release(items []ReserveItem) {
	invMu.Lock()
	defer invMu.Unlock()

	for _, item := range items {
		if inv, ok := inventory[item.ItemID]; ok {
			inv.Quantity += item.Quantity
		}
	}
}

// GetItem returns a single inventory item by ID.
func GetItem(id string) (InventoryItem, bool) {
	invMu.Lock()
	defer invMu.Unlock()

	item, ok := inventory[id]
	if !ok {
		return InventoryItem{}, false
	}
	return *item, true
}

// GetInventory returns a snapshot of the current inventory.
func GetInventory() []InventoryItem {
	invMu.Lock()
	defer invMu.Unlock()

	result := make([]InventoryItem, 0, len(inventory))
	for _, item := range inventory {
		result = append(result, *item)
	}
	return result
}
