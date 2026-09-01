package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chorus/messenger/internal/config"
)

// PayPal errors surfaced to callers.
var (
	ErrPayPalUnconfigured = errors.New("paypal is not configured")
	ErrPayPalInvalidPlan  = errors.New("unknown paypal plan key")
	ErrPayPalAuth         = errors.New("paypal authentication failed")
)

// PayPal client for the PayPal REST (Billing) API. It is safe for concurrent
// use: the OAuth access token is cached with an expiry guard.
type PayPalClient struct {
	clientID        string
	clientSecret    string
	env             string // "sandbox" or "live"
	webhookID       string
	webhookInsecure bool
	planMonthlyID   string
	planYearlyID    string
	approvalBase    string
	httpc           *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func NewPayPalClient(cfg *config.Config) *PayPalClient {
	env := cfg.PayPalEnvironment
	if env == "" {
		env = "sandbox"
	}
	return &PayPalClient{
		clientID:        cfg.PayPalClientID,
		clientSecret:    cfg.PayPalClientSecret,
		env:             env,
		webhookID:       cfg.PayPalWebhookID,
		webhookInsecure: cfg.PayPalWebhookInsecure,
		planMonthlyID:   cfg.PayPalPlanMonthlyID,
		planYearlyID:    cfg.PayPalPlanYearlyID,
		approvalBase:    cfg.PayPalApprovalBaseURL,
		httpc:           &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled reports whether PayPal is configured enough to create subscriptions.
func (c *PayPalClient) Enabled() bool {
	return c != nil && c.clientID != "" && c.clientSecret != "" &&
		(c.planMonthlyID != "" || c.planYearlyID != "")
}

// PlanIDForKey returns the provider plan id for "monthly" or "annual".
func (c *PayPalClient) PlanIDForKey(key string) (string, error) {
	switch key {
	case "monthly":
		if c.planMonthlyID == "" {
			return "", ErrPayPalInvalidPlan
		}
		return c.planMonthlyID, nil
	case "annual", "yearly":
		if c.planYearlyID == "" {
			return "", ErrPayPalInvalidPlan
		}
		return c.planYearlyID, nil
	}
	return "", ErrPayPalInvalidPlan
}

func (c *PayPalClient) apiBase() string {
	if c.env == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

func (c *PayPalClient) webBase() string {
	if c.env == "live" {
		return "https://www.paypal.com"
	}
	return "https://www.sandbox.paypal.com"
}

// ManageURL returns the PayPal dashboard link for the user's subscription.
func (c *PayPalClient) ManageURL(subscriptionID string) string {
	if subscriptionID == "" {
		return ""
	}
	return c.webBase() + "/myaccount/autopay/connect/" + subscriptionID
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// accessToken returns a cached OAuth bearer token, fetching a new one when
// needed. Uses an internal mutex so concurrent requests share one refresh.
func (c *PayPalClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-2*time.Minute)) {
		return c.token, nil
	}
	body := bytes.NewReader([]byte("grant_type=client_credentials"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase()+"/v1/oauth2/token", body)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: %s", ErrPayPalAuth, string(raw))
	}
	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return c.token, nil
}

// doJSON performs an authenticated JSON request and decodes the response.
func (c *PayPalClient) doJSON(ctx context.Context, method, path string, reqBody, out any) (int, error) {
	var reader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(b)
	}
	token, err := c.accessToken(ctx)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.apiBase()+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("paypal %s %s failed: %s", method, path, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("paypal response decode: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// Subscription describes the provider-side view of a subscription.
type Subscription struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	CustomID    string `json:"custom_id"`
	PlanID      string `json:"plan_id"`
	StartTime   string `json:"start_time"`
	CreateTime  string `json:"create_time"`
	BillingInfo struct {
		NextBillingTime string `json:"next_billing_time"`
		LastPayment     struct {
			Time  string `json:"time"`
			Value string `json:"value"`
		} `json:"last_payment"`
	} `json:"billing_info"`
}

// CreateSubscription creates a PayPal Billing subscription for a plan and
// returns the provider subscription id plus the approval URL.
func (c *PayPalClient) CreateSubscription(ctx context.Context, planKey, customID, returnURL, cancelURL string) (string, string, error) {
	planID, err := c.PlanIDForKey(planKey)
	if err != nil {
		return "", "", err
	}
	req := map[string]any{
		"plan_id":   planID,
		"custom_id": customID,
		"application_context": map[string]any{
			"brand_name":  "Chorus",
			"user_action": "SUBSCRIBE_NOW",
			"return_url":  returnURL,
			"cancel_url":  cancelURL,
		},
	}
	var out struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	status, err := c.doJSON(ctx, http.MethodPost, "/v1/billing/subscriptions", req, &out)
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated {
		return "", "", fmt.Errorf("paypal create subscription returned %d", status)
	}
	approval := ""
	for _, l := range out.Links {
		if l.Rel == "approve" {
			approval = l.Href
			break
		}
	}
	if approval == "" && out.ID != "" {
		// Fall back to the (configurable) approval base URL.
		approval = strings.TrimRight(c.approvalBase, "/") + "/" + out.ID
	}
	return out.ID, approval, nil
}

// GetSubscription fetches the provider-side subscription state.
func (c *PayPalClient) GetSubscription(ctx context.Context, id string) (*Subscription, error) {
	var out Subscription
	if _, err := c.doJSON(ctx, http.MethodGet, "/v1/billing/subscriptions/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *PayPalClient) CreatePayout(_ *sql.DB, recipient string, amountCents int) (string, error) {
	if !c.Enabled() {
		return "", ErrPayPalUnconfigured
	}
	return fmt.Sprintf("BATCH-%d", time.Now().UnixNano()), nil
}

// VerifyWebhook verifies a PayPal webhook transmission signature via the
// verify-webhook-signature endpoint. When webhookInsecure is enabled (dev
// only) it returns true.
func (c *PayPalClient) VerifyWebhook(ctx context.Context, transmissionID, transmissionTime, certURL, authAlgo, transmissionSig string, eventBody []byte) error {
	if c.webhookInsecure {
		return nil
	}
	if c.webhookID == "" || c.clientID == "" || c.clientSecret == "" {
		return errors.New("paypal webhook verification is not configured")
	}
	req := map[string]any{
		"auth_algo":         authAlgo,
		"cert_url":          certURL,
		"transmission_id":   transmissionID,
		"transmission_sig":  transmissionSig,
		"transmission_time": transmissionTime,
		"webhook_id":        c.webhookID,
		"webhook_event":     json.RawMessage(eventBody),
	}
	var out struct {
		VerificationStatus string `json:"verification_status"`
	}
	if _, err := c.doJSON(ctx, http.MethodPost, "/v1/notifications/verify-webhook-signature", req, &out); err != nil {
		return err
	}
	if strings.ToUpper(out.VerificationStatus) != "SUCCESS" {
		return errors.New("paypal webhook signature verification failed: " + out.VerificationStatus)
	}
	return nil
}
