package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	"order/model"
)

func main() {
	workflowID := flag.String("workflow", "", "Workflow/Order ID (required)")
	step := flag.String("step", "", "Scenario: shipmentSuccess, shipmentFailed (blank = start workflow only)")
	orderJSON := flag.String("order", "", `Order JSON: {"item_id":"ITEM-001","quantity":2,"email":"customer@example.com"}`)
	flag.Parse()

	if *workflowID == "" {
		log.Fatal("--workflow flag is required")
	}
	if *orderJSON == "" {
		//if (*step == "") {
			log.Fatalf("--order flag is required, e.g.\n  --order '{\"item_id\":\"ITEM-001\",\"quantity\":2,\"email\":\"customer@example.com\"}'\nReceived args: %v", flag.Args())
		//}
	}

	var order struct {
		ItemID   string `json:"item_id"`
		Quantity int    `json:"quantity"`
		Email    string `json:"email"`
	}
	if err := json.Unmarshal([]byte(*orderJSON), &order); err != nil {
		log.Fatalf("Invalid --order JSON: %v", err)
	}

	ctx := context.Background()

	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	input := model.OrderInput{
		OrderID: *workflowID,
		Email:   order.Email,
		Items: []model.OrderItem{
			{ItemID: order.ItemID, Quantity: order.Quantity},
		},
	}

	opts := client.StartWorkflowOptions{
		ID:        *workflowID,
		TaskQueue: model.TaskQueue,
	}

	run, err := c.ExecuteWorkflow(ctx, opts, "OrderWorkflow", input)
	if err != nil {
		log.Fatalf("Failed to start workflow: %v", err)
	}
	fmt.Printf("Started workflow: %s (runID: %s)\n", run.GetID(), run.GetRunID())
	fmt.Printf("Order: item=%s quantity=%d email=%s\n", input.Items[0].ItemID, input.Items[0].Quantity, input.Email)

	// For shipment scenarios, wait a bit then send the signal
	switch *step {
	case "shipmentSuccess":
		time.Sleep(3 * time.Second)
		fmt.Println("Sending shipment signal: success=true")
		err = c.SignalWorkflow(ctx, *workflowID, "", model.ShipmentSignalName, model.ShipmentSignal{Success: true})
		if err != nil {
			log.Fatalf("Failed to signal workflow: %v", err)
		}

	case "shipmentFailed":
		time.Sleep(3 * time.Second)
		fmt.Println("Sending shipment signal: success=false")
		err = c.SignalWorkflow(ctx, *workflowID, "", model.ShipmentSignalName, model.ShipmentSignal{Success: false})
		if err != nil {
			log.Fatalf("Failed to signal workflow: %v", err)
		}

	default:
		// No step provided — just start workflow and wait for it
		fmt.Println("No step provided. Workflow started and waiting for external signal.")
		fmt.Println("Use another simulator instance with -step shipmentSuccess or shipmentFailed to send a signal.")
	}

	// Wait for result
	var result model.OrderResult
	err = run.Get(ctx, &result)
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	fmt.Printf("\n=== Workflow Result ===\n")
	fmt.Printf("Order ID:    %s\n", result.OrderID)
	fmt.Printf("Status:      %s\n", result.Status)
	if result.TrackingID != "" {
		fmt.Printf("Tracking ID: %s\n", result.TrackingID)
	}
	if result.Error != "" {
		fmt.Printf("Error:       %s\n", result.Error)
	}
}
