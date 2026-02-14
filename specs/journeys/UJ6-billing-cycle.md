# UJ6: "I want to review my costs and pay"

**Persona:** Customer with running deployments
**Goal:** Understand costs, review invoices, and pay outstanding balance
**Preconditions:** Signed in. Has at least one running deployment with a non-zero template price.

## Story

1. Signs in. Navigates to Billing via sidebar.
2. Sees summary cards: monthly cost (aggregated from running deployments), number of active deployments, usage event count.
3. Reviews the invoice list. Each invoice shows billing period, line items (deployment names and per-deployment amounts), total, and payment status (paid/outstanding).
4. Clicks "Pay Now" on an outstanding invoice. Redirected to Stripe Checkout (external).
5. Completes payment on Stripe. Returned to the Billing page with a success confirmation.
6. Invoice status updated to "paid."
7. Reviews usage history. Sees API events and deployment lifecycle events that contribute to metering.

## Pages & Features Touched

1. Login (`/login`)
2. Billing page (`/billing`)
3. Invoice list
4. Invoice detail (line items)
5. Stripe Checkout (external redirect)
6. Usage/event history

## Acceptance Criteria

- [ ] Billing page shows accurate monthly cost based on running deployments
- [ ] Active deployment count matches actual running deployments
- [ ] Invoice list shows correct billing periods and amounts
- [ ] Line items break down cost per deployment
- [ ] "Pay Now" redirects to Stripe Checkout with correct amount
- [ ] Successful payment redirects back to billing page
- [ ] Invoice status updates to "paid" after successful payment
- [ ] Free deployments ($0 templates) do not generate charges
- [ ] Stopped/deleted deployments stop accruing charges

## Edge Cases

- **No invoices yet:** Billing page shows summary with $0 and empty invoice list, not an error.
- **Stripe payment fails (card declined):** User returns to billing page; invoice remains outstanding. Clear error message.
- **All deployments are free:** Monthly cost shows $0. No invoices generated.
- **Deployment deleted mid-billing-cycle:** Charges pro-rated or stopped; reflected in next invoice.
- **User on free plan with no billing configured:** Billing page still loads, shows $0 summary.
