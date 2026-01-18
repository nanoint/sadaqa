package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"app/backend/freedompay"
)

const (
	WorkerPoolSize    = 10
	MaxRetries        = 3
	BaseURL           = "https://api.freedompay.money" // Check docs for actual URL
	RecurringScript   = "make_recurring_payment.php"
	SecretKey         = "your_secret_key"
	MerchantID        = "your_merchant_id"
)

// Subscription represents the DB row
type Subscription struct {
	ID             string
	Amount         int
	RecurringToken string
	NextPaymentDate time.Time
	Status         string
	Frequency      string
}

// ProcessDailyPayments triggers the worker pool to handle due payments
func ProcessDailyPayments(db *pgxpool.Pool) {
	ctx := context.Background()
	log.Println("[Worker] Starting Daily Payment Process...")

	// 1. Fetch Due Subscriptions
	// Logic: Active status AND next_payment_date is in the past or today
	rows, err := db.Query(ctx, `
		SELECT id, amount, recurring_token, next_payment_date, status, frequency 
		FROM subscriptions 
		WHERE status = 'active' 
		AND next_payment_date <= NOW()
	`)
	if err != nil {
		log.Printf("[Worker] Error fetching subscriptions: %v", err)
		return
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.Amount, &s.RecurringToken, &s.NextPaymentDate, &s.Status, &s.Frequency); err != nil {
			log.Printf("[Worker] Error scanning row: %v", err)
			continue
		}
		subs = append(subs, s)
	}

	if len(subs) == 0 {
		log.Println("[Worker] No due payments found.")
		return
	}

	log.Printf("[Worker] Found %d subscriptions to process.", len(subs))

	// 2. Setup Worker Pool
	jobs := make(chan Subscription, len(subs))
	var wg sync.WaitGroup

	// Start Workers
	for w := 1; w <= WorkerPoolSize; w++ {
		wg.Add(1)
		go worker(w, db, jobs, &wg)
	}

	// Send Jobs
	for _, s := range subs {
		jobs <- s
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	log.Println("[Worker] Daily Payment Process Completed.")
}

func worker(id int, db *pgxpool.Pool, jobs <-chan Subscription, wg *sync.WaitGroup) {
	defer wg.Done()
	for sub := range jobs {
		processSubscription(db, sub)
	}
}

func processSubscription(db *pgxpool.Pool, sub Subscription) {
	ctx := context.Background()
	log.Printf("[Worker] Processing Sub ID: %s, Amount: %d", sub.ID, sub.Amount)

	// Prepare Request
	newOrderID := fmt.Sprintf("rec_%s_%d", sub.ID, time.Now().Unix())
	req := freedompay.RecurringPaymentRequest{
		PgMerchantId:         MerchantID,
		PgAmount:             fmt.Sprintf("%d", sub.Amount),
		PgOrderId:            newOrderID,
		PgDescription:        "Auto Sadaqa Recurring",
		PgRecurringProfileId: sub.RecurringToken,
		PgSalt:               fmt.Sprintf("salt_%d", rand.Intn(10000)),
	}
	req.PgSig = req.GenerateSignature(RecurringScript, SecretKey)

	// Retry Logic (Exponential Backoff)
	var success bool
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		resp, err := req.SendRequest(BaseURL, RecurringScript)
		
		if err != nil {
			// Network error, retry
			lastErr = err
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			log.Printf("[Worker] Network error for Sub %s (Attempt %d/%d). Retrying in %v. Error: %v", sub.ID, attempt+1, MaxRetries, backoff, err)
			time.Sleep(backoff)
			continue
		}

		if resp.PgStatus == "ok" {
			success = true
			break
		} else {
			// API returned error (e.g. Insufficient funds)
			// We might NOT want to retry these immediately in a tight loop, 
			// but for this requirement, we treat it as a failure to be handled.
			lastErr = fmt.Errorf("API Error: %s", resp.PgError)
			// Don't retry logic errors immediately
			break
		}
	}

	// Update DB Result
	if success {
		nextDate := calculateNextDate(sub.NextPaymentDate, sub.Frequency)
		_, err := db.Exec(ctx, "UPDATE subscriptions SET next_payment_date = $1, last_payment_at = NOW() WHERE id = $2", nextDate, sub.ID)
		if err != nil {
			log.Printf("[Worker] CRITICAL: Payment success but DB update failed for Sub %s: %v", sub.ID, err)
		} else {
			log.Printf("[Worker] Success: Sub %s charged. Next payment: %s", sub.ID, nextDate.Format("2006-01-02"))
		}
	} else {
		// Mark as failed or paused
		log.Printf("[Worker] Failed: Sub %s after retries. Reason: %v", sub.ID, lastErr)
		// Option: Set status to 'payment_failed' so user is notified
		_, _ = db.Exec(ctx, "UPDATE subscriptions SET status = 'payment_failed', last_error = $1 WHERE id = $2", lastErr.Error(), sub.ID)
	}
}

func calculateNextDate(current time.Time, freq string) time.Time {
	// If current date is way in the past, base calculation on NOW? 
	// Usually better to base on schedule to keep day-of-month consistent.
	// Simplified logic:
	now := time.Now()
	if current.Before(now) {
		current = now
	}

	switch freq {
	case "Daily":
		return current.AddDate(0, 0, 1)
	case "Weekly":
		return current.AddDate(0, 0, 7)
	case "Monthly":
		return current.AddDate(0, 1, 0)
	default:
		return current.AddDate(0, 1, 0)
	}
}
