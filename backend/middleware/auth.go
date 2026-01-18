package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TelegramUser represents the JSON structure of the "user" field in initData
type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Language  string `json:"language_code"`
}

// TelegramAuth creates a middleware that validates the initData passed from the Telegram WebApp.
// It requires the Telegram Bot Token to validate the HMAC-SHA256 signature.
func TelegramAuth(botToken string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Get initData from the custom header
		initData := c.Get("X-Telegram-Init-Data")
		if initData == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authentication data",
			})
		}

		// 2. Parse the query string
		// initData format: query_id=...&user=...&auth_date=...&hash=...
		parsedData, err := url.ParseQuery(initData)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authentication data format",
			})
		}

		// 3. Extract the hash provided by Telegram
		receivedHash := parsedData.Get("hash")
		if receivedHash == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing signature hash",
			})
		}

		// 4. Construct the data-check-string
		// Iterate over all keys, exclude 'hash', sort alphabetically, and join with '\n'
		parsedData.Del("hash") // Remove hash from map to allow sorting remaining keys

		var keys []string
		for k := range parsedData {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var dataCheckList []string
		for _, k := range keys {
			// Key=Value format
			dataCheckList = append(dataCheckList, fmt.Sprintf("%s=%s", k, parsedData.Get(k)))
		}
		dataCheckString := strings.Join(dataCheckList, "\n")

		// 5. Calculate the HMAC-SHA256 signature
		// Step A: Calculate the secret key = HMAC_SHA256("WebAppData", botToken)
		skHmac := hmac.New(sha256.New, []byte("WebAppData"))
		skHmac.Write([]byte(botToken))
		secretKey := skHmac.Sum(nil)

		// Step B: Calculate hash = HMAC_SHA256(secretKey, dataCheckString)
		h := hmac.New(sha256.New, secretKey)
		h.Write([]byte(dataCheckString))
		calculatedHash := hex.EncodeToString(h.Sum(nil))

		// 6. Compare hashes
		if calculatedHash != receivedHash {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid signature",
			})
		}

		// 7. Validate auth_date to prevent replay attacks (expire after 24 hours)
		authDateStr := parsedData.Get("auth_date")
		if authDateStr != "" {
			authDate, err := strconv.ParseInt(authDateStr, 10, 64)
			if err == nil {
				// 86400 seconds = 24 hours
				if time.Now().Unix()-authDate > 86400 {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Authentication data expired",
					})
				}
			}
		}

		// 8. Extract User Information
		userStr := parsedData.Get("user")
		var user TelegramUser
		if err := json.Unmarshal([]byte(userStr), &user); err != nil {
			// Even if parsing fails, the auth is valid, but we might lack user info.
			// Depending on requirements, we might want to fail here.
			// For now, we proceed but log it internally if needed.
		}

		// 9. Store in Context
		c.Locals("user_id", user.ID)
		c.Locals("first_name", user.FirstName)
		c.Locals("user_data", user)

		return c.Next()
	}
}
