# UJ8: "I'm done with this deployment"

**Persona:** Customer
**Goal:** Stop and remove a deployment, verify no ongoing charges
**Preconditions:** Signed in. Has at least one deployment (running or stopped).

## Story

1. Signs in. Navigates to "My Deployments" via sidebar. Finds the deployment to remove.
2. Clicks the deployment card. Arrives at the Deployment Detail page.
3. If the deployment is running, clicks "Stop." Watches status transition: running → stopping → stopped. Waits for stopped state.
4. Clicks "Delete." A confirmation prompt appears. Confirms deletion.
5. Watches status transition: stopped → deleting → deleted. Redirected to the deployment list.
6. The deleted deployment no longer appears in "My Deployments."
7. Navigates to Billing. Monthly cost has decreased. No charges accruing for the deleted deployment.
8. Optionally: if the node is no longer needed, navigates to Nodes and removes it. If a cloud server, can destroy it to stop provider charges.

## Pages & Features Touched

1. Login (`/login`)
2. Deployment List (`/deployments`)
3. Deployment Detail (`/deployments/{id}`)
4. Stop action
5. Delete action (with confirmation)
6. Billing page (`/billing`)
7. Nodes page (`/nodes`) — optional cleanup

## Acceptance Criteria

- [ ] Stop transitions: running → stopping → stopped
- [ ] Delete is only available when deployment is in "stopped" or "failed" state
- [ ] Delete confirmation prevents accidental deletion
- [ ] Delete transitions: stopped → deleting → deleted
- [ ] After deletion, user is redirected to deployment list
- [ ] Deleted deployment does not appear in the deployment list
- [ ] Billing reflects reduced cost after deletion (no charges for deleted deployment)
- [ ] Direct URL to deleted deployment returns 404

## Edge Cases

- **Delete a failed deployment:** Allowed directly (skip stop step). failed → deleting → deleted.
- **Delete while containers are still cleaning up:** Backend handles container cleanup asynchronously. Deployment marked deleted even if container removal is in progress.
- **Node removal with remaining deployments:** Error — node cannot be deleted while deployments are scheduled on it.
- **Rapid stop then delete:** System handles sequential transitions correctly; delete waits for stopped state.
- **User refreshes during deletion:** Page shows current state (deleting or redirects to list if already deleted).
