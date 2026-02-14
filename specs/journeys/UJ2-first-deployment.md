# UJ2: "I want to deploy my first app"

**Persona:** New Customer (just signed up)
**Goal:** Go from zero to a running application
**Preconditions:** None — user starts unauthenticated, signs up during this journey

## Story

1. Signs up at `/signup` (name, email, password). On success, JWT stored in localStorage, redirected to Marketplace.
2. Browses published templates. Picks one and clicks "Deploy Now."
3. Deploy dialog opens. Fills in a deployment name (auto-suggested slug, editable). Sees "No online nodes available" warning because they have no infrastructure yet.
4. Realizes infrastructure is needed. Navigates to SSH Keys via sidebar.
5. Clicks "Add SSH Key." Pastes a name and private key. Key appears in the list with a derived fingerprint. Private key is encrypted at rest and never returned in API responses.
6. Navigates to Nodes. Clicks "Add Existing Server." Fills in name, SSH host, port, user, and selects the SSH key from step 5. Submits.
7. Health check runs automatically. Node status transitions to "online" if the server is reachable, or "offline" with an error if not.
8. Returns to the template (via Marketplace or back navigation). Clicks "Deploy Now" again. This time the deploy dialog shows the online node. Optionally sets environment variable overrides and a custom domain. Clicks "Deploy."
9. Deployment is created. Auto-start triggers. Navigated to the Deployment Detail page.
10. Watches status progress: pending → scheduled → starting → running. Status badge updates.
11. Visits the app URL (subdomain or custom domain) and sees the running application.

## Pages & Features Touched

1. Sign Up (`/signup`)
2. Marketplace (`/marketplace`)
3. Template Detail (`/marketplace/{template_id}`)
4. Deploy Dialog (modal)
5. SSH Keys (`/ssh-keys`)
6. Add SSH Key dialog
7. Nodes (`/nodes`)
8. Add Node form (`/nodes/new`)
9. Deployment Detail (`/deployments/{id}`)
10. App URL (external)

## Acceptance Criteria

- [ ] Signup creates account, stores JWT, redirects to marketplace
- [ ] Deploy dialog warns when no online nodes are available
- [ ] SSH key is encrypted at rest (AES-256); never returned in GET responses
- [ ] SSH key fingerprint is derived and displayed after creation
- [ ] Node creation triggers automatic health check
- [ ] Node status accurately reflects SSH connectivity
- [ ] Deploy dialog shows available online nodes after node is registered
- [ ] Deployment name validates (lowercase, alphanumeric, hyphens)
- [ ] Deployment created with correct `customer_id` and `template_id` (integer FK from reference_id)
- [ ] Auto-start triggered after deployment creation
- [ ] Status transitions visible on deployment detail page
- [ ] App is accessible at its assigned URL once running

## Edge Cases

- **SSH key is invalid format:** Error shown in dialog, key not saved.
- **Node health check fails:** Node shows as "offline" with error details. User can fix server config and re-check.
- **Deployment fails to start:** Status transitions to "failed" with error message visible on detail page.
- **Plan limit reached:** APIGate returns quota error; deploy dialog shows clear message with upgrade path.
- **Duplicate deployment name:** Validation error in deploy dialog.
