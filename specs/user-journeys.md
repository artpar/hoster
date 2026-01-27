# Hoster User Journeys

**Purpose:** Document every user interaction flow for manual testing validation.

**CRITICAL:** Unit tests mean nothing if actual user experience is broken. These journeys must be tested manually before every production deployment.

---

## Journey 1: New User Signup & First Deployment

**Actors:** Non-technical user, never used Hoster before

**Steps:**
1. Visit https://emptychair.dev
2. Click "Get Started" button
3. Fill signup form (name, email, password)
4. Submit → Should redirect to /marketplace
5. See marketplace with all templates
6. Click on a template (e.g., "Redis Cache")
7. Click "Deploy Now" button
8. Fill deployment name and variables
9. Click "Deploy"
10. See deployment in "My Deployments"

**Expected Results:**
- ✅ No "Sign In Required" errors after signup
- ✅ Header shows user profile/email (NOT "Sign in via APIGate")
- ✅ "My Deployments" link visible in navigation
- ✅ Deployment appears in list
- ✅ Only user's own deployments shown

**Test Data:**
- Email: testuser-{timestamp}@example.com
- Name: Test User
- Password: test123456

---

## Journey 2: Returning User Login

**Actors:** Existing user with deployments

**Steps:**
1. Visit https://emptychair.dev
2. Click "Sign In"
3. Enter email and password
4. Submit → Should redirect to /marketplace or /deployments
5. Navigate to "My Deployments"
6. See list of deployments

**Expected Results:**
- ✅ Login succeeds without errors
- ✅ Redirected to appropriate page
- ✅ Session persists across page refreshes
- ✅ Only user's deployments shown (privacy check)

---

## Journey 3: Deploy Additional Template

**Actors:** Authenticated user

**Steps:**
1. Navigate to /marketplace
2. Browse templates
3. Select different template
4. Click "Deploy Now"
5. Fill deployment details
6. Submit

**Expected Results:**
- ✅ No auth errors
- ✅ Deployment created successfully
- ✅ Appears in "My Deployments"
- ✅ Can manage (start/stop/restart)

---

## Journey 4: Deployment Lifecycle Management

**Actors:** User with existing deployment

**Steps:**
1. Go to "My Deployments"
2. Click on a deployment
3. View monitoring tabs (Events, Stats, Logs)
4. Click "Stop" button
5. Deployment status → "stopped"
6. Click "Start" button
7. Deployment status → "running"
8. Click "Restart" button
9. Deployment restarts

**Expected Results:**
- ✅ All buttons work without auth errors
- ✅ Status updates correctly
- ✅ Monitoring data shows
- ✅ No access to other users' deployments

---

## Journey 5: Exceed Plan Limits

**Actors:** Free tier user

**Steps:**
1. Deploy first template (within free limit)
2. Try to deploy second template (exceeds limit)
3. See billing error message
4. Navigate to billing/upgrade page
5. Select paid plan
6. Complete payment
7. Retry deployment

**Expected Results:**
- ✅ Clear error message about plan limits
- ✅ Upgrade flow works
- ✅ After upgrade, deployment succeeds
- ✅ Billing integration functional

---

## Journey 6: Creator Template Publishing

**Actors:** Template creator

**Steps:**
1. Navigate to "Creator Dashboard"
2. Click "Create Template"
3. Fill template details:
   - Name, description
   - Docker Compose spec
   - Environment variables
   - Pricing
4. Click "Publish"
5. Template appears in marketplace
6. Other users can deploy it

**Expected Results:**
- ✅ Creator can create and publish
- ✅ Template validation works
- ✅ Marketplace shows published template
- ✅ Deployment from published template works

---

## Journey 7: Auth Persistence

**Actors:** Authenticated user

**Steps:**
1. Sign in successfully
2. Refresh page → Should stay logged in
3. Close browser tab
4. Reopen tab → Should stay logged in
5. Wait 10 minutes
6. Interact with site → Should still be logged in
7. Wait 24 hours
8. Return to site → Session may expire, redirect to login

**Expected Results:**
- ✅ Session persists across refreshes
- ✅ Session persists across tab close/reopen
- ✅ Session timeout handled gracefully
- ✅ No data loss on session expiry

---

## Journey 8: Multi-User Privacy (CRITICAL)

**Actors:** Two different users (User A, User B)

**Steps:**
1. User A signs in, creates deployment "A-deployment"
2. User A signs out
3. User B signs in, creates deployment "B-deployment"
4. User B views "My Deployments"
5. User B should NOT see "A-deployment"
6. User B tries to access A's deployment URL directly
7. Should get 403 Forbidden

**Expected Results:**
- ✅ Users only see their own deployments
- ✅ No way to view/modify other users' deployments
- ✅ Privacy enforced at API level
- ✅ Direct URL access blocked

---

## Journey 9: Node Registration & Remote Deployment

**Actors:** Template creator with remote node

**Steps:**
1. Navigate to "Creator Dashboard" → "Nodes" tab
2. Click "Add SSH Key"
3. Upload private key
4. Click "Add Node"
5. Fill node details (name, host, port, SSH key)
6. Submit → Node health check passes
7. Deploy template → Gets scheduled to remote node
8. Verify deployment running on remote Docker host

**Expected Results:**
- ✅ SSH key encrypted and stored
- ✅ Node health check succeeds
- ✅ Deployment scheduled to available node
- ✅ Container events recorded from remote operations
- ✅ Monitoring (Events, Stats, Logs) works for remote deployments

---

## Journey 10: Session Recovery After Expiry

**Actors:** User whose session expired

**Steps:**
1. User signs in
2. Leave browser open for 24+ hours (session expires)
3. User returns and clicks on deployment
4. See "Sign In Required" message
5. User clicks "Sign In"
6. Redirected to login page
7. User logs in
8. Redirected back to original destination

**Expected Results:**
- ✅ Clear message when session expires
- ✅ Redirect to login preserves intended destination
- ✅ After login, redirected to original page
- ✅ No data loss

---

## Testing Protocol

For EVERY deployment to production:

1. **Test ALL journeys manually**
2. Use Chrome DevTools MCP for automation where possible
3. Document results in test report (template below)
4. Take screenshots of critical steps
5. Verify no regressions

**Never deploy without testing ALL journeys.**

---

## Test Report Template

```markdown
## Production Test Report - v{VERSION}

**Tested by:** {Name}
**Date:** {YYYY-MM-DD}
**Environment:** https://emptychair.dev

### Journey Results

- [ ] Journey 1: New User Signup & First Deployment
- [ ] Journey 2: Returning User Login
- [ ] Journey 3: Deploy Additional Template
- [ ] Journey 4: Deployment Lifecycle Management
- [ ] Journey 5: Exceed Plan Limits
- [ ] Journey 6: Creator Template Publishing
- [ ] Journey 7: Auth Persistence
- [ ] Journey 8: Multi-User Privacy (CRITICAL)
- [ ] Journey 9: Node Registration & Remote Deployment
- [ ] Journey 10: Session Recovery After Expiry

### Issues Found

- None / {List issues with details}

### Screenshots

{Attach screenshots of critical flows}

### Notes

{Any additional observations}

### Recommendation

- [ ] PASS - Ready for production
- [ ] FAIL - Blocking issues found, DO NOT DEPLOY
```

---

## Automation Guidelines

### Using Chrome DevTools MCP

For automated testing, use these tools:
- `mcp__chrome-devtools__navigate_page` - Navigate to pages
- `mcp__chrome-devtools__take_snapshot` - Capture page state
- `mcp__chrome-devtools__fill` - Fill form fields
- `mcp__chrome-devtools__click` - Click buttons
- `mcp__chrome-devtools__wait_for` - Wait for elements
- `mcp__chrome-devtools__take_screenshot` - Capture visual state

### Example Test Script

```typescript
// Journey 1: New User Signup
1. navigate_page({ url: "https://emptychair.dev" })
2. take_snapshot() // Verify homepage loads
3. click({ uid: "sign-up-button" })
4. fill({ uid: "email-input", value: "test@example.com" })
5. fill({ uid: "password-input", value: "test123456" })
6. fill({ uid: "name-input", value: "Test User" })
7. click({ uid: "submit-button" })
8. wait_for({ text: "Marketplace" })
9. take_screenshot() // Verify redirect
```

---

## Failure Patterns to Watch For

### Auth Issues
- User appears logged out after successful signup/login
- "Sign In Required" dialog appears for authenticated users
- Header shows "Sign in via APIGate" instead of user profile
- Session doesn't persist across page loads

### Privacy Issues
- Users can see other users' deployments
- Users can access other users' deployments via direct URL
- Deployment list not filtered by customer_id

### UX Issues
- Confusing error messages
- No indication of what went wrong
- Broken navigation links
- Missing visual feedback

---

## Pre-Deployment Checklist

Before tagging a release:

- [ ] All journeys tested locally
- [ ] All journeys tested on staging (if available)
- [ ] Test report completed
- [ ] Screenshots captured
- [ ] No critical issues found
- [ ] UX flows feel smooth
- [ ] Error messages are clear
- [ ] Privacy is enforced
- [ ] Auth persistence works
- [ ] No regressions from previous version

**If ANY item fails, DO NOT DEPLOY.**
