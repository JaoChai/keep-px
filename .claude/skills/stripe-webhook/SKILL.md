---
name: stripe-webhook
description: Stripe webhook handling, idempotency, credit system, billing — ใช้เมื่อ: เครดิตไม่เข้า, ชำระเงินพัง, Stripe error, webhook ไม่ทำงาน, ซื้อแพ็กไม่ได้, จ่ายเงินแล้วไม่ได้เครดิต, billing error, แก้ระบบชำระเงิน, double charge
---

# Stripe Webhook

## When to Activate

Activate this skill when the user says:
- "Webhook" / "Stripe webhook" / "Handle webhook event"
- "Duplicate credits" / "Double charge" / "Idempotency"
- "Race condition" / "Concurrent replays" / "Credit consumption"
- "Checkout session" / "Subscription event"
- "Stripe API version mismatch"
- "Add new pack type" / "New pricing" / "Add addon"

## Architecture

```
Stripe → POST /webhooks/stripe → BillingHandler.HandleWebhook()
  ↓ signature verification + API version mismatch handling
  ↓ event type routing
  ↓ idempotent wrapper (ProcessWebhookEvent)
  ↓ business logic (HandleCheckoutCompleted / HandleSubscriptionEvent)
```

## Pattern 1: Webhook Handler

**File:** `backend/internal/handler/billing_handler.go`

```go
func (h *BillingHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. Read body with size limit (prevent DoS)
    const maxBodyBytes = 65536
    body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))

    // 2. Verify signature + handle API version mismatch
    sigHeader := r.Header.Get("Stripe-Signature")
    event, err := webhook.ConstructEventWithOptions(body, sigHeader,
        h.cfg.StripeWebhookSecret, webhook.ConstructEventOptions{
            IgnoreAPIVersionMismatch: true,  // CRITICAL: Stripe account may use newer API
        })

    // 3. Route by event type, wrap in idempotency
    switch event.Type {
    case "checkout.session.completed":
        var sess stripe.CheckoutSession
        json.Unmarshal(event.Data.Raw, &sess)
        processErr = h.billingService.ProcessWebhookEvent(r.Context(),
            event.ID, string(event.Type),
            func() error { return h.billingService.HandleCheckoutCompleted(r.Context(), &sess) })

    case "customer.subscription.created",
         "customer.subscription.updated",
         "customer.subscription.deleted":
        var sub stripe.Subscription
        json.Unmarshal(event.Data.Raw, &sub)
        processErr = h.billingService.ProcessWebhookEvent(r.Context(),
            event.ID, string(event.Type),
            func() error { return h.billingService.HandleSubscriptionEvent(r.Context(), &sub) })
    }

    // 4. Always return 200 (let Stripe retry on actual failures via idempotency rollback)
    w.WriteHeader(http.StatusOK)
}
```

**Rules:**
- Always use `IgnoreAPIVersionMismatch: true` — Stripe account API version may differ from library
- Size-limit the body to prevent memory exhaustion
- Always return HTTP 200 — idempotency layer handles retries
- Unmarshal the specific Stripe type from `event.Data.Raw`

## Pattern 2: Idempotent Event Processing (Claim-Process-Rollback)

**File:** `backend/internal/service/billing_service.go`

```go
func (s *BillingService) ProcessWebhookEvent(ctx context.Context,
    stripeEventID, eventType string, process func() error) error {

    // CLAIM: Atomically try to insert event (ON CONFLICT DO NOTHING)
    inserted, err := s.webhookRepo.CreateIfNotExists(ctx, stripeEventID, eventType)
    if err != nil {
        return err
    }
    if !inserted {
        slog.Info("skipping duplicate webhook event", "event_id", stripeEventID)
        return nil  // Already processed — silently succeed
    }

    // PROCESS: Run the handler
    if err := process(); err != nil {
        // ROLLBACK: Delete event record so Stripe retry will succeed
        if delErr := s.webhookRepo.Delete(ctx, stripeEventID); delErr != nil {
            slog.Error("failed to roll back webhook event claim", "error", delErr)
        }
        return err
    }

    return nil
}
```

**SQL for idempotency:**
```sql
-- INSERT with conflict detection (atomic)
INSERT INTO stripe_webhook_events (stripe_event_id, event_type)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
RETURNING stripe_event_id;
-- Returns row if inserted (first time)
-- Returns ErrNoRows if conflict (duplicate)
```

**Key behaviors:**
- First attempt: INSERT succeeds → process handler → keep record
- Duplicate: INSERT conflicts → skip silently → return nil
- Process fails: DELETE record → Stripe retries → INSERT succeeds again
- This is **NOT a transaction** — it's a claim pattern with manual rollback

## Pattern 3: Atomic Credit Consumption (Race Condition Prevention)

**File:** `backend/internal/repository/postgres/replay_credit_repo.go`

```go
func (r *ReplayCreditRepo) ConsumeOneCredit(ctx context.Context,
    customerID string) (*domain.ReplayCredit, error) {

    tx, err := r.pool.Begin(ctx)
    defer func() { _ = tx.Rollback(ctx) }()

    // SELECT with row lock + skip already-locked rows
    var credit domain.ReplayCredit
    err = tx.QueryRow(ctx, `
        SELECT id, customer_id, purchase_id, pack_type,
               total_replays, used_replays, max_events_per_replay,
               expires_at, created_at
        FROM replay_credits
        WHERE customer_id = $1
          AND (total_replays = -1 OR used_replays < total_replays)
          AND expires_at > NOW()
        ORDER BY expires_at ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED`, customerID).Scan(...)

    if err == pgx.ErrNoRows {
        return nil, nil  // No available credit
    }

    // Atomically increment used_replays with safety check
    _, err = tx.Exec(ctx, `
        UPDATE replay_credits SET used_replays = used_replays + 1
        WHERE id = $1 AND (total_replays = -1 OR used_replays < total_replays)`,
        credit.ID)

    if err := tx.Commit(ctx); err != nil {
        return nil, err
    }
    return &credit, nil
}
```

**Key patterns:**
- `FOR UPDATE` — locks the row, prevents concurrent reads
- `SKIP LOCKED` — other transactions skip locked rows (fair queueing, no deadlocks)
- `ORDER BY expires_at ASC` — consume soonest-expiring credits first (FIFO)
- Double safety: WHERE clause on both SELECT and UPDATE prevents overselling
- Transaction scope is minimal (just lock + increment)

## Pattern 4: Safe Persistence Order

When consuming limited resources, persist the audit trail FIRST:

```go
// CORRECT order:
// 1. Create session record (audit trail exists)
session := &domain.ReplaySession{Status: "pending"}
s.replayRepo.Create(ctx, session)

// 2. Consume credit (resource deducted)
credit, err := s.quotaService.ConsumeReplayCredit(ctx, customerID, eventCount)
if err != nil {
    // Mark session as failed — user sees what happened
    s.replayRepo.UpdateStatusWithError(ctx, session.ID, "failed", err.Error())
    return nil, err
}

// WRONG order:
// 1. Consume credit → credit gone
// 2. Create session → FAILS → credit lost, no audit trail
```

## Pattern 5: Purchase → Credit Flow

```go
// 1. Create pending purchase
purchase := &domain.Purchase{
    CustomerID:   customerID,
    PackType:     packType,
    AmountSatang: packCfg.AmountSatang,
    Status:       domain.PurchaseStatusPending,
}
s.purchaseRepo.Create(ctx, purchase)

// 2. Create Stripe checkout with metadata
sess, _ := checkoutsession.New(&stripe.CheckoutSessionParams{
    Metadata: map[string]string{
        "purchase_id": purchase.ID,    // Links back to our purchase
        "customer_id": customerID,
        "pack_type":   packType,
    },
})

// 3. Webhook: checkout.session.completed
//    → Update purchase status to "completed"
//    → Create replay_credit linked to purchase_id

// 4. Unique index prevents duplicate credits per purchase
// CREATE UNIQUE INDEX idx_credits_unique_purchase
//     ON replay_credits(purchase_id) WHERE purchase_id IS NOT NULL;
```

## Adding a New Pack Type

1. Add constant in `backend/internal/domain/billing.go`:
```go
const PackNewType = "new_type"
```

2. Add config in `backend/internal/service/billing_service.go` (packConfigs map):
```go
domain.PackNewType: {
    PriceID:            cfg.StripePriceNewType,
    AmountSatang:       99900,
    TotalReplays:       5,
    MaxEventsPerReplay: domain.FreeMaxEventsPerReplay,
    ExpiryDays:         180,
},
```

3. Add Stripe Price ID env var in `backend/internal/config/config.go`:
```go
StripePriceNewType string `env:"STRIPE_PRICE_NEW_TYPE"`
```

4. Create the Price in Stripe Dashboard or via API
5. Set the env var in Railway

## Adding a New Subscription Add-on

1. Add constant in `domain/billing.go`:
```go
const AddonNewFeature = "new_feature"
```

2. Add to addonPriceIDs map in billing service
3. Handle in `HandleSubscriptionEvent` → resolve addon type from price ID
4. Update quota service to check the new addon
5. Create Stripe Price (recurring) and set env var

## Common Pitfalls

| Pitfall | Fix |
|---------|-----|
| Webhook retries create duplicate credits | Use ProcessWebhookEvent wrapper (idempotency) |
| Concurrent replays oversell credits | Use FOR UPDATE SKIP LOCKED in transaction |
| Credit consumed but session fails | Create session FIRST, then consume credit |
| Stripe API version mismatch rejects webhook | Use IgnoreAPIVersionMismatch: true |
| Undefined Stripe env var crashes on startup | Make Stripe config optional, check before use |
| Purchase not linked to credit | Pass purchase_id in Stripe metadata + unique index |

## Database Schema

```sql
-- Webhook idempotency
CREATE TABLE stripe_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stripe_event_id VARCHAR(255) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);

-- Replay credits with purchase linking
CREATE TABLE replay_credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    purchase_id UUID REFERENCES purchases(id),
    pack_type VARCHAR(50) NOT NULL,
    total_replays INT NOT NULL,       -- -1 = unlimited
    used_replays INT NOT NULL DEFAULT 0,
    max_events_per_replay INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Prevent duplicate credits per purchase
CREATE UNIQUE INDEX idx_credits_unique_purchase
    ON replay_credits(purchase_id) WHERE purchase_id IS NOT NULL;
```

## Verification

```bash
# Backend builds and tests pass
cd backend && go build ./cmd/server && go vet ./... && go test ./internal/service/...

# Check webhook locally with Stripe CLI (if installed)
stripe listen --forward-to localhost:8080/webhooks/stripe
stripe trigger checkout.session.completed
```

## Related

- `go-service-scaffold` for creating new billing-related resources
- `db-migration` for schema changes to billing tables
- Built-in `stripe-best-practices` for general Stripe patterns
- Built-in `security-review` for webhook security audit
