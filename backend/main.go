package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"app/backend/freedompay"
	"app/backend/middleware"
	"app/backend/worker"
)

// Config vars (mock env)
const (
	// In production, fetch these from os.Getenv
	DefaultScriptName = "freedompay"
)

var db *pgxpool.Pool

// InitPaymentInput defines the expected body for payment initialization
type InitPaymentInput struct {
	MosqueID  string `json:"mosqueId"`
	Amount    int    `json:"amount"` // Amount in KZT
	Frequency string `json:"frequency"`
}

func main() {
	// 1. Initialize DB Connection
	// Ensure DATABASE_URL is set in environment: postgres://user:password@host:port/dbname
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:password@localhost:5432/autosadaqa" // Default for dev
	}

	var err error
	db, err = pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		// handling for dev environment without real db
	} else {
		defer db.Close()
		log.Println("Connected to Database")
	}

	// 2. Setup Fiber
	app := fiber.New()
	app.Use(logger.New())

	// 3. Public Routes (Webhooks)
	app.Post("/webhooks/freedompay", handleFreedomPayWebhook)
	
	// Admin endpoint to manually trigger the worker
	app.Post("/admin/run-worker", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(500).SendString("DB not connected")
		}
		go worker.ProcessDailyPayments(db)
		return c.SendString("Worker started")
	})

	// 4. Protected API Routes (User interaction)
	// All routes in this group require valid Telegram Init Data
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		botToken = "your_telegram_bot_token"
	}
	api := app.Group("/api", middleware.TelegramAuth(botToken))
	
	api.Get("/me", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		firstName := c.Locals("first_name")
		
		return c.JSON(fiber.Map{
			"status": "authenticated",
			"user_id": userID,
			"first_name": firstName,
			"message": fmt.Sprintf("Welcome, %s!", firstName),
		})
	})

	api.Post("/init-payment", func(c *fiber.Ctx) error {
		var input InitPaymentInput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
		}

		if db == nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database not connected"})
		}

		// Retrieve User ID from context (set by TelegramAuth middleware)
		userID, ok := c.Locals("user_id").(int64)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "User ID not found in context"})
		}

		orderID := fmt.Sprintf("sub_%d_%d", userID, time.Now().Unix())
		
		// Config - prefer Env vars
		merchantID := os.Getenv("FREEDOMPAY_MERCHANT_ID")
		if merchantID == "" {
			merchantID = "526868" // Fallback/Test ID
		}
		secretKey := os.Getenv("FREEDOMPAY_SECRET")
		if secretKey == "" {
			secretKey = "SecretKey01" // Fallback/Test Key
		}
		// Base URL for webhooks
		appBaseUrl := os.Getenv("APP_BASE_URL")
		if appBaseUrl == "" {
			appBaseUrl = "https://your-app-url.com" 
		}

		// 1. Construct Freedom Pay Request
		req := freedompay.PaymentRequest{
			PgMerchantId:     merchantID,
			PgAmount:         fmt.Sprintf("%d", input.Amount),
			PgOrderId:        orderID,
			PgDescription:    "Sadaqa Subscription",
			PgSalt:           "random_salt_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			PgRecurringStart: true, // CRITICAL: This tells Freedom Pay to save the card
		}

		// 2. Generate Signature (init_payment.php)
		req.PgSig = req.GenerateSignature("init_payment.php", secretKey)

		// 3. Construct the Redirect URL
		queryParams := url.Values{}
		queryParams.Add("pg_merchant_id", req.PgMerchantId)
		queryParams.Add("pg_amount", req.PgAmount)
		queryParams.Add("pg_order_id", req.PgOrderId)
		queryParams.Add("pg_description", req.PgDescription)
		queryParams.Add("pg_salt", req.PgSalt)
		queryParams.Add("pg_recurring_start", "1")
		queryParams.Add("pg_sig", req.PgSig)
		
		// IMPORTANT: Set Tunnel/Production URL for webhook result
		webhookURL := fmt.Sprintf("%s/webhooks/freedompay", appBaseUrl)
		queryParams.Add("pg_result_url", webhookURL)

		paymentUrl := "https://api.freedompay.money/init_payment.php?" + queryParams.Encode()

		// 4. Create "Pending" Subscription in DB
		// Note: We use the mosque name "Hazrat Sultan" as default/placeholder based on instructions, 
		// or fetch it if we had a Mosques table.
		_, err := db.Exec(c.Context(), `
			INSERT INTO subscriptions (id, user_id, mosque_id, mosque_name, amount, frequency, status, next_payment_date)
			VALUES ($1, $2, $3, $4, $5, $6, 'pending', NOW() + INTERVAL '1 week')
		`, orderID, userID, input.MosqueID, "Hazrat Sultan Mosque", input.Amount, input.Frequency)

		if err != nil {
			log.Printf("DB Insert Error: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Database error creating subscription"})
		}

		return c.JSON(fiber.Map{
			"paymentUrl":    paymentUrl,
			"transactionId": orderID,
		})
	})

	// 5. Start Server
	log.Fatal(app.Listen(":8080"))
}

// handleFreedomPayWebhook processes the callback from Freedom Pay
func handleFreedomPayWebhook(c *fiber.Ctx) error {
	var req freedompay.WebhookRequest
	secretKey := os.Getenv("FREEDOMPAY_SECRET")
	if secretKey == "" {
		secretKey = "SecretKey01"
	}
	scriptName := os.Getenv("FREEDOMPAY_SCRIPT_NAME")
	if scriptName == "" {
		scriptName = "freedompay"
	}

	// 1. Parse Request (supports JSON, XML, Form)
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing webhook body: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Request")
	}

	// 2. Validate Signature
	if !req.VerifySignature(scriptName, secretKey) {
		log.Println("Invalid signature received")
		return c.Status(fiber.StatusForbidden).SendString("Invalid Signature")
	}

	// 3. Check Payment Result
	if req.PgResult == "1" {
		// 4. Update Database
		recurringToken := req.PgRecurringProfileId
		if recurringToken == "" {
			recurringToken = req.PgCardId
		}

		err := updateSubscription(c.Context(), req.PgOrderId, recurringToken, req.PgCardPan)
		if err != nil {
			log.Printf("Failed to update subscription: %v", err)
		} else {
			log.Printf("Subscription activated for Order ID: %s", req.PgOrderId)
		}
	} else {
		log.Printf("Payment failed for Order ID: %s", req.PgOrderId)
	}

	// 5. Return XML Response
	resp := freedompay.WebhookResponse{
		PgStatus:      "ok",
		PgDescription: "Accepted",
		PgSalt:        req.PgSalt,
	}
	resp.PgSig = resp.GenerateSignature(scriptName, secretKey)

	c.Set("Content-Type", "application/xml")
	xmlBytes, _ := xml.Marshal(resp)
	
	return c.SendString(xml.Header + string(xmlBytes))
}

// updateSubscription updates the subscription status in Postgres
func updateSubscription(ctx context.Context, orderId string, token string, pan string) error {
	if db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	query := `
		UPDATE subscriptions 
		SET 
			status = 'active', 
			recurring_token = $1, 
			card_pan = $2,
			updated_at = NOW()
		WHERE id = $3
	`
	
	cmdTag, err := db.Exec(ctx, query, token, pan, orderId)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("no subscription found with id: %s", orderId)
	}

	return nil
}
