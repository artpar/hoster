# UJ3: "I want to check on my apps"

**Persona:** Returning Customer
**Goal:** Monitor health, review logs, manage lifecycle of running deployments
**Preconditions:** Signed in. Has at least one deployment in "running" state on an online node.

## Story

1. Signs in at `/login`. JWT stored, redirected to marketplace. Navigates to Dashboard via sidebar.
2. Dashboard shows summary stats: running deployments count, online nodes count, monthly cost. Sees recent deployments list with status badges.
3. Clicks a running deployment. Arrives at Deployment Detail page.
4. **Overview tab:** Sees container health (up/down per service), resource usage bars, deployment info (template, node, created date, domain).
5. **Logs tab:** Reads recent container stdout/stderr output. Spots an error in the log stream.
6. **Stats tab:** Checks CPU percentage, memory usage (MB), and network bytes charts. Charts update periodically.
7. **Events tab:** Reviews lifecycle event timeline — creation, scheduling, start, health checks. Events ordered newest-first.
8. Decides to restart the app. Clicks "Stop." Watches status: running → stopping → stopped.
9. Clicks "Start." Watches status: stopped → starting → running. App is back up.
10. Goes back to "My Deployments" list. Sees all deployments with accurate status badges.

## Pages & Features Touched

1. Login (`/login`)
2. Dashboard (`/dashboard`)
3. Deployment List (`/deployments`)
4. Deployment Detail — Overview tab (`/deployments/{id}`)
5. Deployment Detail — Logs tab
6. Deployment Detail — Stats tab
7. Deployment Detail — Events tab
8. Stop action
9. Start action

## Acceptance Criteria

- [ ] Dashboard stats are accurate and reflect current state
- [ ] Recent deployments on dashboard link to detail pages
- [ ] Deployment detail shows correct status badge with color coding
- [ ] Overview tab shows per-service container health
- [ ] Logs tab streams container output
- [ ] Stats tab displays CPU, memory, and network charts
- [ ] Events tab shows lifecycle events newest-first
- [ ] Stop transitions: running → stopping → stopped
- [ ] Start transitions: stopped → starting → running
- [ ] Action buttons only show valid transitions for current state
- [ ] Deployment list shows only the current user's deployments (owner scoping)
- [ ] Cannot access another user's deployment (returns 404, not 403)

## Edge Cases

- **Node goes offline while deployment is running:** Container health shows degraded/unknown. Status may transition to "failed."
- **Logs unavailable (node unreachable):** Logs tab shows error message, not an infinite spinner.
- **Rapid start/stop:** System handles transitions correctly without getting stuck in intermediate states.
- **Session expires mid-monitoring:** 401 triggers auth prompt; after re-login, returns to same page (→ UJ7).
