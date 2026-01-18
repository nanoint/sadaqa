package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"os"

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
	SecretKey  = "your_secret_key" // Should come from env
	ScriptName = "freedompay"      // The script name configured in Freedom Pay (e.g. "result.php" or "freedompay")
	BotToken   = "your_telegram_bot_token" // Env var
)

var db *pgxpool.Pool

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
	api := app.Group("/api", middleware.TelegramAuth(BotToken))
	
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

	// 5. Start Server
	log.Fatal(app.Listen(":8080"))
}

// handleFreedomPayWebhook processes the callback from Freedom Pay
func handleFreedomPayWebhook(c *fiber.Ctx) error {
	var req freedompay.WebhookRequest

	// 1. Parse Request (supports JSON, XML, Form)
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing webhook body: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("Invalid Request")
	}

	// 2. Validate Signature
	if !req.VerifySignature(ScriptName, SecretKey) {
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
	resp.PgSig = resp.GenerateSignature(ScriptName, SecretKey)

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
