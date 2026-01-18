package freedompay

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"time"
)

// PaymentRequest represents the standard fields for a Freedom Pay transaction
type PaymentRequest struct {
	PgMerchantId     string `json:"pg_merchant_id"`
	PgAmount         string `json:"pg_amount"`
	PgOrderId        string `json:"pg_order_id"`
	PgDescription    string `json:"pg_description"`
	PgSalt           string `json:"pg_salt"`
	PgRecurringStart bool   `json:"pg_recurring_start"`
	PgSig            string `json:"pg_sig"`
}

// RecurringPaymentRequest represents the fields for a subsequent recurring charge
type RecurringPaymentRequest struct {
	PgMerchantId         string `json:"pg_merchant_id"`
	PgAmount             string `json:"pg_amount"`
	PgOrderId            string `json:"pg_order_id"`
	PgDescription        string `json:"pg_description"`
	PgRecurringProfileId string `json:"pg_recurring_profile_id"`
	PgSalt               string `json:"pg_salt"`
	PgSig                string `json:"pg_sig"`
}

// WebhookRequest represents the data sent by Freedom Pay to the result/check URL
type WebhookRequest struct {
	PgOrderId            string `json:"pg_order_id" form:"pg_order_id" xml:"pg_order_id"`
	PgPaymentId          string `json:"pg_payment_id" form:"pg_payment_id" xml:"pg_payment_id"`
	PgStatus             string `json:"pg_status,omitempty" form:"pg_status" xml:"pg_status"` // sometimes sent
	PgResult             string `json:"pg_result" form:"pg_result" xml:"pg_result"`
	PgAmount             string `json:"pg_amount" form:"pg_amount" xml:"pg_amount"`
	PgCurrency           string `json:"pg_currency" form:"pg_currency" xml:"pg_currency"`
	PgCardId             string `json:"pg_card_id" form:"pg_card_id" xml:"pg_card_id"`
	PgRecurringProfileId string `json:"pg_recurring_profile_id" form:"pg_recurring_profile_id" xml:"pg_recurring_profile_id"`
	PgCardPan            string `json:"pg_card_pan" form:"pg_card_pan" xml:"pg_card_pan"`
	PgSalt               string `json:"pg_salt" form:"pg_salt" xml:"pg_salt"`
	PgSig                string `json:"pg_sig" form:"pg_sig" xml:"pg_sig"`
	PgDescription        string `json:"pg_description" form:"pg_description" xml:"pg_description"`
}

// WebhookResponse represents the XML response required by Freedom Pay
type WebhookResponse struct {
	PgStatus      string `xml:"pg_status"`
	PgDescription string `xml:"pg_description"`
	PgSalt        string `xml:"pg_salt"`
	PgSig         string `xml:"pg_sig"`
}

// APIResponse represents the immediate XML response from making a request
type APIResponse struct {
	PgStatus      string `xml:"pg_status"`
	PgPaymentId   string `xml:"pg_payment_id"`
	PgRedirectUrl string `xml:"pg_redirect_url"`
	PgError       string `xml:"pg_error_description"`
	PgSig         string `xml:"pg_sig"`
}

// GenerateSignature generates the MD5 signature required by Freedom Pay.
func (p *PaymentRequest) GenerateSignature(scriptName, secretKey string) string {
	return makeSignature(p, scriptName, secretKey)
}

// GenerateSignature generates the MD5 signature for recurring requests.
func (p *RecurringPaymentRequest) GenerateSignature(scriptName, secretKey string) string {
	return makeSignature(p, scriptName, secretKey)
}

// VerifySignature checks if the incoming webhook signature is valid
func (w *WebhookRequest) VerifySignature(scriptName, secretKey string) bool {
	expectedSig := makeSignature(w, scriptName, secretKey)
	return expectedSig == w.PgSig
}

// GenerateResponseSignature generates signature for the XML response
func (w *WebhookResponse) GenerateSignature(scriptName, secretKey string) string {
	return makeSignature(w, scriptName, secretKey)
}

// SendRequest sends the recurring payment request to Freedom Pay
func (p *RecurringPaymentRequest) SendRequest(baseUrl, scriptName string) (*APIResponse, error) {
	// Convert struct to url.Values for form-urlencoded post
	data := url.Values{}
	v := reflect.ValueOf(*p)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag != "" {
			data.Set(tag, v.Field(i).String())
		}
	}

	apiURL := fmt.Sprintf("%s/%s", baseUrl, scriptName)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := xml.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %v | Body: %s", err, string(bodyBytes))
	}

	return &apiResp, nil
}

// makeSignature is a generic helper that handles reflection for signature generation
func makeSignature(data interface{}, scriptName, secretKey string) string {
	params := make(map[string]string)
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		
		// Try to get key from json, form, or xml tag
		tag := field.Tag.Get("json")
		if tag == "" {
			tag = field.Tag.Get("form")
		}
		if tag == "" {
			tag = field.Tag.Get("xml")
		}

		// Clean up tag (remove omitempty)
		if len(tag) > 0 {
			if idx := 0; idx < len(tag) {
				// simple parse for comma
				for j := 0; j < len(tag); j++ {
					if tag[j] == ',' {
						tag = tag[:j]
						break
					}
				}
			}
		}

		if tag == "" || tag == "pg_sig" {
			continue
		}

		val := v.Field(i).Interface()
		strVal := ""

		switch v := val.(type) {
		case string:
			strVal = v
		case int:
			strVal = strconv.Itoa(v)
		case bool:
			if v {
				strVal = "1"
			} else {
				strVal = "0"
			}
		}

		if strVal != "" {
			params[tag] = strVal
		}
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	flatStr := scriptName
	for _, k := range keys {
		flatStr += ";" + params[k]
	}
	flatStr += ";" + secretKey

	hasher := md5.New()
	hasher.Write([]byte(flatStr))
	return hex.EncodeToString(hasher.Sum(nil))
}