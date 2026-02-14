package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"order/activity"
	"order/model"
	wf "order/workflow"
)

func main() {

	hostPort := os.Getenv("TEMPORAL_ADDRESS")
	if hostPort == "" {
		hostPort = "localhost:7233"
		log.Println("TEMPORAL_ADDRESS not set, defaulting to", hostPort)
	} else {
		log.Println("Read from the TEMPORAL_ADDRESS env variable: ", hostPort)
	}

	c, err := client.Dial(client.Options{
		HostPort: hostPort,
	})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	w := worker.New(c, model.TaskQueue, worker.Options{})

	w.RegisterWorkflow(wf.OrderWorkflow)
	w.RegisterActivity(activity.ReserveInventory)
	w.RegisterActivity(activity.ReleaseInventory)
	w.RegisterActivity(activity.ChargePayment)
	w.RegisterActivity(activity.RefundPayment)
	w.RegisterActivity(activity.SendEmail)
	w.RegisterActivity(activity.ShipOrder)

	log.Println("Worker started, listening on task queue:", model.TaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}
