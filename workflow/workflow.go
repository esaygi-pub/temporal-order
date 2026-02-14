package workflow

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"order/activity"
	"order/model"
)

func OrderWorkflow(ctx workflow.Context, input model.OrderInput) (model.OrderResult, error) {
	logger := workflow.GetLogger(ctx)

	// Default activity options. 
	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2, // 2 retry by default
		},
	})

	// Email activity with exponential backoff retry
	emailCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    10, // max 10 retries
		},
	})

	// Saga compensation stack
	var compensations []func(workflow.Context)

	runCompensations := func() {
		for i := len(compensations) - 1; i >= 0; i-- {
			compensations[i](actCtx)
		}
	}

	// Convert model items to activity items
	type activityItem = struct {
		ItemID   string `json:"item_id"`
		Quantity int    `json:"quantity"`
	}
	actItems := make([]activityItem, len(input.Items))
	for i, item := range input.Items {
		actItems[i] = activityItem{ItemID: item.ItemID, Quantity: item.Quantity}
	}

	// Step 1: Reserve inventory
	logger.Info("Reserving inventory", "orderID", input.OrderID)
	var totalPrice float64
	err := workflow.ExecuteActivity(actCtx, activity.ReserveInventory, input.OrderID, actItems).Get(ctx, &totalPrice)
	if err != nil {
		logger.Error("Failed to reserve inventory", "error", err)
		return model.OrderResult{
			OrderID: input.OrderID,
			Status:  "FAILED",
			Error:   fmt.Sprintf("inventory reservation failed: %v", err),
		}, nil
	}
	compensations = append(compensations, func(ctx workflow.Context) {
		logger.Info("Compensating: releasing inventory")
		_ = workflow.ExecuteActivity(ctx, activity.ReleaseInventory, input.OrderID, actItems).Get(ctx, nil)
	})

	// Step 2: Charge payment
	logger.Info("Charging payment", "orderID", input.OrderID, "amount", totalPrice)
	err = workflow.ExecuteActivity(actCtx, activity.ChargePayment, input.OrderID, totalPrice).Get(ctx, nil)
	if err != nil {
		logger.Error("Payment failed, running compensations", "error", err)
		runCompensations()
		return model.OrderResult{
			OrderID: input.OrderID,
			Status:  "FAILED",
			Error:   fmt.Sprintf("payment failed: %v", err),
		}, nil
	}
	compensations = append(compensations, func(ctx workflow.Context) {
		logger.Info("Compensating: refunding payment")
		_ = workflow.ExecuteActivity(ctx, activity.RefundPayment, input.OrderID, totalPrice).Get(ctx, nil)
	})

	// Step 3: Send confirmation email (retry with exponential backoff)
	logger.Info("Sending confirmation email", "orderID", input.OrderID)
	err = workflow.ExecuteActivity(emailCtx, activity.SendEmail, input.OrderID, input.Email).Get(ctx, nil)
	if err != nil {
		// Non-retryable error (e.g., invalid email) — log and proceed
		logger.Warn("Email send failed with non-retryable error, proceeding", "error", err)
	}

	// Step 4: Wait for shipment signal
	logger.Info("Waiting for shipment signal", "orderID", input.OrderID)
	var shipmentSignal model.ShipmentSignal
	signalCh := workflow.GetSignalChannel(ctx, model.ShipmentSignalName)
	signalCh.Receive(ctx, &shipmentSignal)

	if !shipmentSignal.Success {
		logger.Error("Shipment signal indicates failure, running compensations")
		// Send failure notification email
		_ = workflow.ExecuteActivity(emailCtx, activity.SendEmail, input.OrderID, input.Email).Get(ctx, nil)
		runCompensations()
		return model.OrderResult{
			OrderID: input.OrderID,
			Status:  "FAILED",
			Error:   "shipment failed",
		}, nil
	}

	// Step 5: Ship order
	logger.Info("Shipping order", "orderID", input.OrderID)
	var trackingID string
	err = workflow.ExecuteActivity(actCtx, activity.ShipOrder, input.OrderID).Get(ctx, &trackingID)
	if err != nil {
		logger.Error("Ship API failed, running compensations", "error", err)
		_ = workflow.ExecuteActivity(emailCtx, activity.SendEmail, input.OrderID, input.Email).Get(ctx, nil)
		runCompensations()
		return model.OrderResult{
			OrderID: input.OrderID,
			Status:  "FAILED",
			Error:   fmt.Sprintf("shipment failed: %v", err),
		}, nil
	}

	logger.Info("Order completed successfully", "orderID", input.OrderID, "trackingID", trackingID)
	return model.OrderResult{
		OrderID:    input.OrderID,
		Status:     "COMPLETED",
		TrackingID: trackingID,
	}, nil
}
