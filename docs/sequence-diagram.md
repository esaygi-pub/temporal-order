# Order Workflow — Sequence Diagrams

## Happy Path (shipmentSuccess)

```mermaid
sequenceDiagram
    participant CLI as Simulator CLI
    participant TC as Temporal Client
    participant WF as OrderWorkflow
    participant A as Activities
    participant MS as Mock Server

    CLI->>TC: StartWorkflow(orderID, item, qty, email)
    TC->>WF: Execute OrderWorkflow

    WF->>A: ReserveInventory(orderID, items)
    A->>MS: POST /inventory/reserve
    MS-->>A: 200 OK (totalPrice)
    A-->>WF: totalPrice

    WF->>A: ChargePayment(orderID, totalPrice)
    A->>MS: POST /payment/charge
    MS-->>A: 200 OK
    A-->>WF: success

    WF->>A: SendEmail(orderID, email)
    A->>MS: POST /email/send
    MS-->>A: 500 (random failure)
    Note over WF,A: Exponential backoff retry
    A->>MS: POST /email/send
    MS-->>A: 200 OK
    A-->>WF: success

    WF-->>WF: Waiting for shipment signal...

    CLI->>TC: SignalWorkflow(success=true)
    TC->>WF: ShipmentSignal{Success: true}

    WF->>A: ShipOrder(orderID)
    A->>MS: POST /shipment/ship
    MS-->>A: 200 OK (trackingID)
    A-->>WF: TRK-order-123

    WF-->>TC: OrderResult{Status: COMPLETED, TrackingID: TRK-order-123}
    TC-->>CLI: Print result
```

## Inventory Reservation Failure

```mermaid
sequenceDiagram
    participant CLI as Simulator CLI
    participant TC as Temporal Client
    participant WF as OrderWorkflow
    participant A as Activities
    participant MS as Mock Server

    CLI->>TC: StartWorkflow(orderID, item, qty, email)
    TC->>WF: Execute OrderWorkflow

    WF->>A: ReserveInventory(orderID, items)
    A->>MS: POST /inventory/reserve
    MS-->>A: 409 Conflict (insufficient stock)
    A-->>WF: NonRetryableError (InventoryUnavailable)

    Note over WF: No compensations to run — workflow terminates

    WF-->>TC: OrderResult{Status: FAILED, Error: "inventory reservation failed"}
    TC-->>CLI: Print result
```

## Payment Failure with Saga Rollback

```mermaid
sequenceDiagram
    participant CLI as Simulator CLI
    participant TC as Temporal Client
    participant WF as OrderWorkflow
    participant A as Activities
    participant MS as Mock Server

    CLI->>TC: StartWorkflow(orderID, item, qty, email)
    TC->>WF: Execute OrderWorkflow

    WF->>A: ReserveInventory(orderID, items)
    A->>MS: POST /inventory/reserve
    MS-->>A: 200 OK (totalPrice)
    A-->>WF: totalPrice
    Note over WF: Push compensation: ReleaseInventory

    WF->>A: ChargePayment(orderID, totalPrice)
    A->>MS: POST /payment/charge
    MS-->>A: 500 error
    A-->>WF: error

    rect rgb(255, 220, 220)
        Note over WF: Saga Compensation (reverse order)
        WF->>A: ReleaseInventory(orderID, items)
        A->>MS: POST /inventory/release
        MS-->>A: 200 OK
        A-->>WF: success
    end

    WF-->>TC: OrderResult{Status: FAILED, Error: "payment failed"}
    TC-->>CLI: Print result
```

## Shipment Failure with Full Saga Rollback (shipmentFailed)

```mermaid
sequenceDiagram
    participant CLI as Simulator CLI
    participant TC as Temporal Client
    participant WF as OrderWorkflow
    participant A as Activities
    participant MS as Mock Server

    CLI->>TC: StartWorkflow(orderID, item, qty, email)
    TC->>WF: Execute OrderWorkflow

    WF->>A: ReserveInventory(orderID, items)
    A->>MS: POST /inventory/reserve
    MS-->>A: 200 OK (totalPrice)
    A-->>WF: totalPrice
    Note over WF: Push compensation: ReleaseInventory

    WF->>A: ChargePayment(orderID, totalPrice)
    A->>MS: POST /payment/charge
    MS-->>A: 200 OK
    A-->>WF: success
    Note over WF: Push compensation: RefundPayment

    WF->>A: SendEmail(orderID, email)
    A->>MS: POST /email/send
    MS-->>A: 200 OK
    A-->>WF: success

    WF-->>WF: Waiting for shipment signal...

    CLI->>TC: SignalWorkflow(success=false)
    TC->>WF: ShipmentSignal{Success: false}

    WF->>A: SendEmail (failure notification)
    A->>MS: POST /email/send
    MS-->>A: 200 OK

    rect rgb(255, 220, 220)
        Note over WF: Saga Compensation (reverse order)
        WF->>A: RefundPayment(orderID, totalPrice)
        A->>MS: POST /payment/refund
        MS-->>A: 200 OK
        A-->>WF: success

        WF->>A: ReleaseInventory(orderID, items)
        A->>MS: POST /inventory/release
        MS-->>A: 200 OK
        A-->>WF: success
    end

    WF-->>TC: OrderResult{Status: FAILED, Error: "shipment failed"}
    TC-->>CLI: Print result
```
