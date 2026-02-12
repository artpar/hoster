package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// generateInvoiceHandler creates an invoice from the user's current running deployments.
// POST /api/v1/billing/generate-invoice
func generateInvoiceHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		now := time.Now().UTC()
		periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

		// Get user's running deployments
		deployments, err := cfg.Store.List(ctx, "deployments", nil, Page{Limit: 1000})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list deployments")
			return
		}

		type lineItem struct {
			DeploymentID   string `json:"deployment_id"`
			DeploymentName string `json:"deployment_name"`
			TemplateName   string `json:"template_name"`
			MonthlyCents   int    `json:"monthly_cents"`
			Description    string `json:"description"`
		}

		var items []lineItem
		var totalCents int

		for _, d := range deployments {
			// Filter to this user's running deployments
			ownerID, _ := toInt64(d["customer_id"])
			if int(ownerID) != authCtx.UserID {
				continue
			}
			status, _ := d["status"].(string)
			if status != "running" {
				continue
			}

			// Look up template for price
			var priceCents int
			var templateName string
			if tmplID, ok := toInt64(d["template_id"]); ok && tmplID > 0 {
				tmpl, err := cfg.Store.GetByID(ctx, "templates", int(tmplID))
				if err == nil {
					if p, ok := toInt64(tmpl["price_monthly_cents"]); ok {
						priceCents = int(p)
					}
					templateName = strVal(tmpl["name"])
				}
			}

			deplName := strVal(d["name"])
			items = append(items, lineItem{
				DeploymentID:   strVal(d["reference_id"]),
				DeploymentName: deplName,
				TemplateName:   templateName,
				MonthlyCents:   priceCents,
				Description:    fmt.Sprintf("%s (%s) — %s", deplName, templateName, periodStart.Format("Jan 2006")),
			})
			totalCents += priceCents
		}

		if len(items) == 0 {
			writeError(w, http.StatusBadRequest, "no running deployments to invoice")
			return
		}

		// Serialize items to JSON
		itemsJSON, _ := json.Marshal(items)

		// Create the invoice — set user_id manually since Store.Create doesn't take AuthContext
		invoiceData := map[string]any{
			"user_id":        authCtx.UserID,
			"period_start":   periodStart.Format(time.RFC3339),
			"period_end":     periodEnd.Format(time.RFC3339),
			"items":          string(itemsJSON),
			"subtotal_cents": totalCents,
			"tax_cents":      0,
			"total_cents":    totalCents,
			"currency":       "USD",
		}

		row, err := cfg.Store.Create(ctx, "invoices", invoiceData)
		if err != nil {
			cfg.Logger.Error("failed to create invoice", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create invoice")
			return
		}

		res := cfg.Store.Resource("invoices")
		stripFields(res, row, cfg.Store)
		writeJSON(w, http.StatusCreated, map[string]any{
			"data": rowToJSONAPI("invoices", row),
		})
	}
}

// invoicePayHandler creates a Stripe Checkout session for the invoice.
// POST /api/v1/invoices/{id}/pay
func invoicePayHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		if cfg.StripeKey == "" {
			writeError(w, http.StatusServiceUnavailable, "payment not configured")
			return
		}

		invoice, err := cfg.Store.Get(ctx, "invoices", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}

		// Check ownership
		ownerID, ok := toInt64(invoice["user_id"])
		if !ok || int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		status, _ := invoice["status"].(string)
		if status != "draft" && status != "failed" {
			writeError(w, http.StatusConflict, "invoice is not payable in state: "+status)
			return
		}

		totalCents, _ := toInt64(invoice["total_cents"])
		if totalCents <= 0 {
			writeError(w, http.StatusBadRequest, "invoice has no amount")
			return
		}

		refID := strVal(invoice["reference_id"])

		// Parse request body for success/cancel URLs
		var body struct {
			SuccessURL string `json:"success_url"`
			CancelURL  string `json:"cancel_url"`
		}
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&body)
		}
		if body.SuccessURL == "" {
			body.SuccessURL = "http://localhost:3000/billing?payment=success&invoice=" + refID
		}
		if body.CancelURL == "" {
			body.CancelURL = "http://localhost:3000/billing?payment=cancelled"
		}

		// Append session_id placeholder for verification
		if !strings.Contains(body.SuccessURL, "{CHECKOUT_SESSION_ID}") {
			sep := "&"
			if !strings.Contains(body.SuccessURL, "?") {
				sep = "?"
			}
			body.SuccessURL += sep + "session_id={CHECKOUT_SESSION_ID}"
		}

		// Create Stripe Checkout Session via HTTP
		checkoutURL, sessionID, err := createStripeCheckout(
			cfg.StripeKey,
			totalCents,
			strVal(invoice["currency"]),
			body.SuccessURL,
			body.CancelURL,
			"Hoster Invoice "+refID,
		)
		if err != nil {
			cfg.Logger.Error("stripe checkout failed", "error", err, "invoice", refID)
			writeError(w, http.StatusBadGateway, "payment provider error: "+err.Error())
			return
		}

		// Update invoice with Stripe session info and transition to pending
		cfg.Store.Update(ctx, "invoices", id, map[string]any{
			"stripe_session_id":  sessionID,
			"stripe_payment_url": checkoutURL,
		})
		cfg.Store.Transition(ctx, "invoices", id, "pending")

		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{
				"checkout_url": checkoutURL,
				"session_id":   sessionID,
				"invoice_id":   refID,
			},
		})
	}
}

// verifyPaymentHandler checks a Stripe Checkout session and updates the invoice.
// GET /api/v1/billing/verify-payment?session_id=xxx
func verifyPaymentHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		if cfg.StripeKey == "" {
			writeError(w, http.StatusServiceUnavailable, "payment not configured")
			return
		}

		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			writeError(w, http.StatusBadRequest, "session_id required")
			return
		}

		// Find invoice by stripe_session_id — query all user's invoices
		allInvoices, err := cfg.Store.List(ctx, "invoices", nil, Page{Limit: 100})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to query invoices")
			return
		}

		var invoice map[string]any
		for _, inv := range allInvoices {
			if strVal(inv["stripe_session_id"]) == sessionID {
				ownerID, _ := toInt64(inv["user_id"])
				if int(ownerID) == authCtx.UserID {
					invoice = inv
					break
				}
			}
		}

		if invoice == nil {
			writeError(w, http.StatusNotFound, "invoice not found for this session")
			return
		}

		// Check Stripe session status
		paid, err := checkStripeSession(cfg.StripeKey, sessionID)
		if err != nil {
			cfg.Logger.Error("stripe session check failed", "error", err, "session", sessionID)
			writeError(w, http.StatusBadGateway, "payment verification failed")
			return
		}

		refID := strVal(invoice["reference_id"])
		status, _ := invoice["status"].(string)

		if paid && status == "pending" {
			now := time.Now().UTC().Format(time.RFC3339)
			cfg.Store.Update(ctx, "invoices", refID, map[string]any{
				"paid_at": now,
			})
			cfg.Store.Transition(ctx, "invoices", refID, "paid")
			cfg.Logger.Info("invoice paid", "invoice", refID, "session", sessionID)
		}

		resultStatus := status
		if paid {
			resultStatus = "paid"
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{
				"invoice_id": refID,
				"paid":       paid,
				"status":     resultStatus,
			},
		})
	}
}

// createStripeCheckout creates a Stripe Checkout Session via the REST API.
// Returns (checkout_url, session_id, error).
func createStripeCheckout(stripeKey string, amountCents int64, currency, successURL, cancelURL, description string) (string, string, error) {
	data := url.Values{}
	data.Set("mode", "payment")
	data.Set("line_items[0][price_data][currency]", strings.ToLower(currency))
	data.Set("line_items[0][price_data][product_data][name]", description)
	data.Set("line_items[0][price_data][unit_amount]", fmt.Sprintf("%d", amountCents))
	data.Set("line_items[0][quantity]", "1")
	data.Set("success_url", successURL)
	data.Set("cancel_url", cancelURL)

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(stripeKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("stripe request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("stripe error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("parse response: %w", err)
	}

	return result.URL, result.ID, nil
}

// checkStripeSession checks if a Stripe Checkout Session has been paid.
func checkStripeSession(stripeKey, sessionID string) (bool, error) {
	req, err := http.NewRequest("GET", "https://api.stripe.com/v1/checkout/sessions/"+sessionID, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(stripeKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("stripe request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("stripe error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		PaymentStatus string `json:"payment_status"`
		Status        string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("parse response: %w", err)
	}

	return result.PaymentStatus == "paid", nil
}
