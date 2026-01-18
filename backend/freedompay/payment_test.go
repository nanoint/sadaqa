package freedompay

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
)

func TestGenerateSignature(t *testing.T) {
	// 1. Define inputs
	req := PaymentRequest{
		PgMerchantId:     "500123",
		PgAmount:         "1000",
		PgOrderId:        "ORDER-999",
		PgDescription:    "Mosque Donation",
		PgSalt:           "random_salt_123",
		PgRecurringStart: true,
	}
	scriptName := "init_payment.php"
	secretKey := "super_secret_key"

	// 2. Manually construct the expected string to hash
	// Parameters sorted alphabetically:
	// pg_amount: 1000
	// pg_description: Mosque Donation
	// pg_merchant_id: 500123
	// pg_order_id: ORDER-999
	// pg_recurring_start: 1
	// pg_salt: random_salt_123

	flatString := "init_payment.php;1000;Mosque Donation;500123;ORDER-999;1;random_salt_123;super_secret_key"
	
	hasher := md5.New()
	hasher.Write([]byte(flatString))
	expectedSig := hex.EncodeToString(hasher.Sum(nil))

	// 3. Run the method
	actualSig := req.GenerateSignature(scriptName, secretKey)

	// 4. Verify
	if actualSig != expectedSig {
		t.Errorf("Signature mismatch.\nExpected: %s\nGot:      %s", expectedSig, actualSig)
	}
}

func TestGenerateSignature_FalseBoolean(t *testing.T) {
	// Test to ensure boolean false is handled as "0"
	req := PaymentRequest{
		PgMerchantId:     "500123",
		PgAmount:         "500",
		PgRecurringStart: false,
		PgSalt:           "salt",
	}
	scriptName := "init_payment.php"
	secretKey := "key"

	// keys: pg_amount, pg_merchant_id, pg_recurring_start, pg_salt
	// values: 500, 500123, 0, salt
	// Note: Empty strings (PgDescription, PgOrderId) are included as empty in map but logic handles them as strings
	// Current logic in payment.go includes them if they are empty strings because the struct has them. 
	// Wait, standard map loop in payment.go:
	// `if strVal != "" { params[tag] = strVal }` 
	// This means empty strings are SKIPPED.
	
	flatString := "init_payment.php;500;500123;0;salt;key"

	hasher := md5.New()
	hasher.Write([]byte(flatString))
	expectedSig := hex.EncodeToString(hasher.Sum(nil))

	actualSig := req.GenerateSignature(scriptName, secretKey)

	if actualSig != expectedSig {
		t.Errorf("Signature mismatch for false bool/empty strings.\nExpected: %s\nGot:      %s", expectedSig, actualSig)
	}
}
