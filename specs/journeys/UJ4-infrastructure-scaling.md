# UJ4: "I want to scale up my infrastructure"

**Persona:** Operator
**Goal:** Add more server capacity via cloud provisioning
**Preconditions:** Signed in. May or may not have existing nodes.

## Story

1. Signs in. Navigates to Nodes via sidebar. Sees existing nodes (if any) on the Nodes tab.
2. Wants to add cloud capacity instead of manually setting up a server. Clicks the "Cloud Servers" tab.
3. Realizes they need cloud credentials first. Clicks the "Credentials" tab.
4. Clicks "Add Credential." Fills in a name, selects provider (AWS, DigitalOcean, or Hetzner), and pastes the API key. Credential saved (key encrypted at rest).
5. Returns to the "Cloud Servers" tab. Clicks "Create Cloud Server."
6. Selects the credential from step 4. Selects provider, region (loaded from provider API), and instance size (loaded from provider API). Fills in an instance name. Submits.
7. Watches provisioning progress: pending → creating → configuring → ready. Status updates visible on the page.
8. On "ready": a node is auto-registered in the Nodes tab with "online" status. SSH key auto-assigned.
9. Navigates back to Nodes tab. Confirms the new node appears and is online.
10. Deploys an app to the new node via the marketplace deploy flow. Confirms the app runs on the new infrastructure.

## Pages & Features Touched

1. Login (`/login`)
2. Nodes — Nodes tab (`/nodes`)
3. Nodes — Cloud Servers tab (`/nodes` cloud tab)
4. Nodes — Credentials tab (`/nodes` credentials tab)
5. Add Credential dialog
6. Create Cloud Server form (`/nodes/cloud/new`)
7. Deploy Dialog (from marketplace)
8. Deployment Detail (`/deployments/{id}`)

## Acceptance Criteria

- [ ] Credential creation requires name, provider, and API key
- [ ] API key is encrypted at rest and never returned in GET responses
- [ ] Cloud server creation requires a valid credential
- [ ] Region and size options are loaded dynamically from the cloud provider
- [ ] Provisioning status transitions are visible: pending → creating → configuring → ready
- [ ] On "ready," node is auto-registered and appears in the Nodes tab as "online"
- [ ] Auto-registered node is usable for deployments immediately
- [ ] Newly provisioned node appears as an option in the deploy dialog

## Edge Cases

- **Invalid API key:** Credential creation succeeds (key stored), but cloud server creation fails with provider auth error. Clear error message shown.
- **Provisioning fails (provider quota, invalid region):** Status transitions to "failed" with error details. User can retry or destroy the failed instance.
- **Region/size API call fails:** Form shows error loading options, not empty dropdowns.
- **User deletes credential while a cloud server is provisioning:** Provisioning completes or fails based on in-flight state; credential removal doesn't crash anything.
- **Cloud server destroyed:** Status transitions to "destroying" → "destroyed." Associated node removed or marked offline.
