# Hoster User Journey Book

## Personas

### Visitor
Anonymous user browsing the platform. Has not signed up. Wants to understand what Hoster offers before committing.

### Customer
Signed-up user who deploys applications from the marketplace onto their own infrastructure. Manages deployments, monitors health, controls lifecycle.

### Creator
Power user who builds docker-compose templates, sets pricing, and publishes them to the marketplace for others to deploy.

### Operator
User who manages infrastructure — registers servers, uploads SSH keys, provisions cloud instances, monitors node health.

---

## Part 1: Discovery

### J1. Visitor Lands on Homepage

**Persona:** Visitor
**Goal:** Understand what Hoster is and decide whether to sign up
**Entry:** Direct URL or search engine

**Flow:**
```
Homepage (/)
  |
  |- Reads hero: "Deploy apps to your own servers"
  |- Reads 3-step guide (Deploy, Bring Servers, Create Templates)
  |- Reads feature cards (Monitoring, SSH Keys, Self-Hosted)
  |
  +-> "Browse Apps" -> Marketplace (J2)
  +-> "Get Started" -> Sign Up (J3)
```

**Sees:**
- Hero with value proposition
- 3-step explainer
- Feature highlights (monitoring, SSH keys, self-hosted)
- Navigation: Marketplace, Sign In, Get Started
- Footer with links

**Acceptance:**
- [ ] Page loads without auth errors
- [ ] No sidebar navigation (anonymous layout)
- [ ] CTAs link to correct destinations
- [ ] Page is fast (<2s load)

---

### J2. Visitor Browses Marketplace

**Persona:** Visitor
**Goal:** See available templates before signing up
**Entry:** Homepage CTA or direct /marketplace link

**Flow:**
```
Marketplace (/marketplace)
  |
  |- Sees published templates grouped by category
  |- Can search by name/description
  |- Can filter by category pill
  |
  +-> Click template card -> Template Detail (J5)
  +-> Click "Deploy Now" -> Auth Required dialog -> Sign In (J3)
```

**Sees:**
- Template cards with name, version, description, price
- Category pills (Database, Web Apps, Monitoring, etc.)
- Search box
- "Sign In" button in header (not username)

**Acceptance:**
- [ ] Published templates visible without login
- [ ] Unpublished (draft) templates NOT visible
- [ ] Category filter works
- [ ] Search filters by name and description
- [ ] "Deploy Now" on template detail shows auth prompt, not a crash

---

## Part 2: Onboarding

### J3. Visitor Signs Up

**Persona:** Visitor -> Customer
**Goal:** Create an account to deploy apps
**Entry:** Homepage "Get Started" or Login page "Sign up" link

**Flow:**
```
Sign Up (/signup)
  |
  |- Fills: Full Name, Email, Password
  |- Submits form
  |
  +-> Success -> Redirect to Marketplace (/marketplace)
  +-> Error (duplicate email) -> Inline error message
```

**Sees:**
- Clean form: name, email, password
- "Create Account" button
- "Already have an account? Sign in" link
- After success: header shows username + "Sign Out"

**Acceptance:**
- [ ] Form validates required fields client-side
- [ ] Password minimum length enforced
- [ ] Duplicate email shows clear error
- [ ] Redirect to marketplace after success
- [ ] JWT stored in localStorage, sent on subsequent requests
- [ ] User resolved in Hoster DB via `ResolveUser()`

---

### J4. Customer Signs In

**Persona:** Customer
**Goal:** Access their account on return visit
**Entry:** Homepage "Sign In" or direct /login

**Flow:**
```
Sign In (/login)
  |
  |- Fills: Email, Password
  |- Submits form
  |
  +-> Success -> Redirect to Marketplace (/marketplace)
  +-> Error (wrong password) -> Inline error message
```

**Sees:**
- Email + password form
- "Sign in" button
- "Don't have an account? Sign up" link
- After success: full sidebar navigation, username in header

**Acceptance:**
- [ ] Login succeeds with correct credentials
- [ ] Wrong password shows clear error (not 500)
- [ ] Session persists across page refresh
- [ ] Session persists across tab close/reopen
- [ ] JWT stored in localStorage under `hoster-auth` key

---

## Part 3: Deploying Applications

### J5. Customer Views Template Detail

**Persona:** Customer (or Visitor)
**Goal:** Evaluate a template before deploying
**Entry:** Click template card from Marketplace

**Flow:**
```
Template Detail (/marketplace/{template_id})
  |
  |- Reads: name, version, description, published status
  |- Reviews: included services, compose spec
  |- Sees: price, publish date
  |
  +-> "Deploy Now" -> Deploy Dialog (J6)
  +-> "Back to Marketplace" -> Marketplace (J2)
```

**Sees:**
- Template name + "Published" badge + version
- Description section
- Included Services (parsed from compose spec)
- Full Docker Compose specification
- Price card (Free or $X.XX/month)
- Published date, last updated date

**Acceptance:**
- [ ] All template fields rendered correctly
- [ ] Services extracted from compose spec
- [ ] Price formatted correctly (Free vs dollar amount)
- [ ] "Deploy Now" opens deploy dialog (authenticated) or auth prompt (anonymous)

---

### J6. Customer Deploys a Template

**Persona:** Customer
**Goal:** Launch a running instance of a template
**Precondition:** Signed in, viewing template detail

**Flow:**
```
Deploy Dialog (modal on template detail)
  |
  |- Auto-generated deployment name (editable)
  |- Optional: custom domain
  |- Optional: environment variable overrides
  |- Shows monthly cost
  |- Clicks "Deploy"
  |
  +-> Success -> Deployment Detail (/deployments/{id})
  |             Status: "pending" -> "scheduled" -> "starting" -> "running"
  |             (or "failed" if no nodes available)
  |
  +-> Error (plan limit) -> Error message in dialog
  +-> Cancel -> Closes dialog
```

**Sees:**
- Deployment name field (pre-filled slug)
- Domain hint: "{name}.yourdomain.com"
- Custom domain input (optional)
- Environment overrides textarea (KEY=value format)
- Monthly cost summary
- Deploy / Cancel buttons

**Acceptance:**
- [ ] Default name is valid slug
- [ ] Name validation (lowercase, alphanumeric, hyphens)
- [ ] Template version auto-populated from template
- [ ] template_id resolved from reference_id to integer FK
- [ ] Deployment created in DB with correct customer_id
- [ ] Auto-start triggered after creation
- [ ] Navigates to deployment detail page
- [ ] Status transitions visible in real-time

---

### J7. Customer Views Deployment Detail

**Persona:** Customer
**Goal:** Monitor and manage a running deployment
**Entry:** Deploy success redirect, or My Deployments list

**Flow:**
```
Deployment Detail (/deployments/{id})
  |
  |- Header: name, status badge, action buttons
  |- Tabs: Overview | Logs | Stats | Events | Domains
  |
  |- Overview: container health, resource usage, deployment info
  |- Logs: container stdout/stderr streams
  |- Stats: CPU, memory, network charts
  |- Events: lifecycle event timeline
  |- Domains: assigned domains, DNS status
  |
  +-> "Start" -> Transition to starting -> running
  +-> "Stop" -> Transition to stopping -> stopped
  +-> "Delete" -> Transition to deleting -> deleted -> redirect to list
  +-> "Back to Deployments" -> Deployments list (J8)
```

**Sees:**
- Deployment name as heading
- Status badge (pending/scheduled/starting/running/stopping/stopped/failed/deleted)
- Error message (if failed)
- Action buttons contextual to current state
- Tab navigation for monitoring views
- Created/updated timestamps

**Acceptance:**
- [ ] Status badge color matches state
- [ ] Action buttons only show valid transitions
- [ ] Error message displayed when status is "failed"
- [ ] Tabs switch content without page reload
- [ ] Start/Stop dispatch correct state machine transitions
- [ ] Cannot access another user's deployment (404 not 403)

---

### J8. Customer Views Deployment List

**Persona:** Customer
**Goal:** See all their deployments at a glance
**Entry:** Sidebar "My Deployments"

**Flow:**
```
My Deployments (/deployments)
  |
  |- List of deployment cards (name, status, created date)
  |
  +-> Click card -> Deployment Detail (J7)
  +-> "New Deployment" -> Marketplace (J2)
```

**Sees:**
- Page heading + description
- "New Deployment" link to marketplace
- Deployment cards with: name, status badge, created date
- Empty state if no deployments: "No deployments yet" + link to marketplace

**Acceptance:**
- [ ] Only shows current user's deployments (owner scoping)
- [ ] Status badges accurate
- [ ] Cards link to detail page
- [ ] Empty state has helpful CTA

---

## Part 4: Creating Templates

### J9. Creator Builds a Template

**Persona:** Creator
**Goal:** Package a docker-compose app as a deployable template
**Entry:** Sidebar "App Templates"

**Flow:**
```
App Templates (/templates)
  |
  |- List of creator's templates (drafts + published)
  |- Search + status filter (All/Drafts/Published)
  |- "Create Template" button
  |
  +-> "Create Template" -> Create Dialog
      |
      |- Fills: name, description, version (semver)
      |- Fills: docker-compose spec
      |- Optional: category, variables, pricing
      |- Submits
      |
      +-> Success -> Template appears in list with "Draft" badge
      +-> Validation error -> Inline error
```

**Sees:**
- Template cards with: name, status (Draft/Published), version, description, price
- Action buttons per card: Publish (if draft), Edit, Delete
- Search box + status filter dropdown
- Create dialog with form fields

**Acceptance:**
- [ ] Only shows current user's templates
- [ ] Name validated: 3-100 chars, alphanumeric + spaces + hyphens
- [ ] Version validated: semver X.Y.Z
- [ ] Compose spec required
- [ ] Slug auto-generated from name
- [ ] Resource defaults applied (cpu=0, memory=0, disk=0, price=0)
- [ ] New template appears as "Draft"

---

### J10. Creator Publishes a Template

**Persona:** Creator
**Goal:** Make template visible in the public marketplace
**Precondition:** Has a draft template

**Flow:**
```
App Templates (/templates)
  |
  |- Finds draft template card
  |- Clicks "Publish" button
  |
  +-> Success -> "Draft" badge removed, "Publish" button removed
  +-> Template now visible in Marketplace for all users
```

**Acceptance:**
- [ ] Publish button only on draft templates
- [ ] After publish: badge gone, button gone
- [ ] Template appears in Marketplace immediately
- [ ] Other users can see and deploy it
- [ ] Creator can still Edit and Delete

---

## Part 5: Infrastructure Management

### J11. Operator Adds an SSH Key

**Persona:** Operator
**Goal:** Store an SSH private key for node authentication
**Entry:** Sidebar "SSH Keys"

**Flow:**
```
SSH Keys (/ssh-keys)
  |
  |- "Add SSH Key" button
  |
  +-> Dialog: name + private key paste
      |
      +-> Success -> Key appears in list with fingerprint
      +-> Error (invalid key) -> Inline error
```

**Sees:**
- Page description: AES-256 encryption
- Key list with: name, fingerprint, created date
- Empty state with explanation of what SSH keys are for

**Acceptance:**
- [ ] Private key encrypted at rest (AES-256)
- [ ] Private key never returned in GET responses (write-only)
- [ ] Fingerprint derived and displayed
- [ ] Key usable when registering nodes

---

### J12. Operator Registers a Node

**Persona:** Operator
**Goal:** Connect a remote server for deployments
**Entry:** Sidebar "My Nodes"

**Flow:**
```
My Nodes (/nodes)
  |
  |- Tabs: Nodes | Cloud Servers | Credentials
  |- "Add Existing Server" button
  |
  +-> Add Node form (/nodes/new)
      |
      |- Fills: name, SSH host, SSH port, SSH user
      |- Selects: SSH key (from J11)
      |- Submits
      |
      +-> Success -> Node appears in list, health check runs
      +-> Health check passes -> Status: "online"
      +-> Health check fails -> Status: "offline" with error
```

**Sees:**
- Node cards with: name, host, status (online/offline), capacity
- Cloud Servers tab for provisioned instances
- Credentials tab for cloud API keys
- Node Setup Guide in empty state

**Acceptance:**
- [ ] Node created with correct SSH details
- [ ] SSH key association works
- [ ] Health check runs automatically after creation
- [ ] Status reflects actual connectivity
- [ ] Node available for deployment scheduling

---

### J13. Operator Provisions a Cloud Server

**Persona:** Operator
**Goal:** Spin up a new VPS from a cloud provider
**Precondition:** Cloud credential stored

**Flow:**
```
Cloud Servers (/nodes/cloud)
  |
  +-> "Create Cloud Server" (/nodes/cloud/new)
      |
      |- Selects: credential, provider, region, size
      |- Fills: instance name
      |- Submits
      |
      +-> Provision starts: pending -> creating -> configuring -> ready
      +-> On ready: Node auto-registered, SSH key auto-assigned
      +-> On failure: error message, can retry or destroy
```

**Acceptance:**
- [ ] Cloud credential required
- [ ] Region/size options loaded from provider
- [ ] Provisioning progress visible in real-time
- [ ] On success: node auto-registered and online
- [ ] On failure: clear error, destroy option available

---

## Part 6: Dashboard & Monitoring

### J14. Customer Views Dashboard

**Persona:** Customer
**Goal:** Get an overview of everything at a glance
**Entry:** Sidebar "Dashboard"

**Flow:**
```
Dashboard (/dashboard)
  |
  |- Stat cards: Deployments (running/total), Templates (published/total),
  |              Nodes (online/total), Monthly Revenue
  |- Recent Deployments list
  |- Node Health summary
  |- Template Performance (deployment count, revenue)
```

**Acceptance:**
- [ ] Stats accurate and up-to-date
- [ ] Recent deployments link to detail pages
- [ ] Node health reflects actual status
- [ ] Template performance shows deployment counts
- [ ] Revenue calculated from running deployments

---

### J15. Customer Monitors a Running Deployment

**Persona:** Customer
**Goal:** Check health, read logs, view metrics for a live deployment
**Precondition:** Deployment in "running" state on an online node

**Flow:**
```
Deployment Detail (/deployments/{id})
  |
  |- Overview tab: container health (up/down), resource bars
  |- Logs tab: real-time container stdout/stderr
  |- Stats tab: CPU %, memory MB, network bytes charts
  |- Events tab: lifecycle event timeline (created, started, health checks)
  |- Domains tab: assigned domains, DNS verification status
```

**Acceptance:**
- [ ] Container health shows per-service status
- [ ] Logs stream in near-real-time
- [ ] Stats charts update periodically
- [ ] Events ordered newest-first
- [ ] Domain DNS status reflects actual resolution

---

## Part 7: Account & Session

### J16. Customer Signs Out

**Persona:** Customer
**Goal:** End their session
**Entry:** "Sign Out" button in header

**Flow:**
```
Any page (authenticated)
  |
  |- Click "Sign Out"
  |
  +-> JWT cleared from localStorage
  +-> Redirect to Homepage (/)
  +-> Header shows "Sign In" instead of username
```

**Acceptance:**
- [ ] Token removed from localStorage
- [ ] Subsequent API calls fail with 401
- [ ] Redirected to landing page
- [ ] No stale data visible after sign out

---

### J17. Session Expires Gracefully

**Persona:** Customer
**Goal:** Continue working after session timeout
**Precondition:** JWT has expired (24h+ inactivity)

**Flow:**
```
Any authenticated page (expired token)
  |
  |- User clicks any action
  |- API returns 401
  |
  +-> "Authentication Required" dialog appears
      |
      +-> "Sign In" -> Login page (/login)
      +-> After login -> Redirect back to original page
```

**Acceptance:**
- [ ] Clear "session expired" message (not raw 401 error)
- [ ] Login preserves intended destination
- [ ] No data loss from in-progress forms
- [ ] Re-authentication restores full access

---

## Part 8: Security & Privacy

### J18. Multi-User Data Isolation

**Persona:** Customer A + Customer B
**Goal:** Verify users cannot see each other's data
**Priority:** CRITICAL

**Flow:**
```
User A: sign in -> create deployment "A-deploy" -> sign out
User B: sign in -> create deployment "B-deploy"
  |
  |- "My Deployments" shows only "B-deploy"
  |- Direct URL to A's deployment -> 404 Not Found
  |- API call for A's deployment -> 404 Not Found
  |- "App Templates" shows only B's templates
```

**Acceptance:**
- [ ] Deployment list scoped by customer_id
- [ ] Template list scoped by creator_id
- [ ] Node list scoped by creator_id
- [ ] SSH key list scoped by creator_id
- [ ] Direct reference_id access returns 404 (not 403) for wrong owner
- [ ] API-level enforcement (not just frontend filtering)

---

### J19. Anonymous Access Boundaries

**Persona:** Visitor
**Goal:** Verify anonymous users can only access public resources

**Flow:**
```
Anonymous user:
  |
  |- GET /marketplace -> Published templates visible
  |- GET /marketplace/{id} -> Template detail visible
  |- POST /api/v1/deployments -> 401 Unauthorized
  |- GET /deployments -> Empty (no auth context)
  |- GET /templates -> Empty (no auth context)
  |- GET /nodes -> Empty (no auth context)
```

**Acceptance:**
- [ ] Marketplace browsable without login
- [ ] Template detail accessible without login
- [ ] All write operations require authentication
- [ ] Owner-scoped lists return empty (not error) for anonymous users
- [ ] "Deploy Now" on template shows auth prompt

---

## Part 9: Billing & Limits

### J20. Free Tier Limit Enforcement

**Persona:** Customer on free plan
**Goal:** Understand limits and upgrade path
**Precondition:** Free plan allows 1 deployment

**Flow:**
```
Customer deploys first template -> Success
Customer deploys second template
  |
  +-> APIGate returns 429 / quota error
  +-> Clear message: "Free plan allows 1 deployment. Upgrade to deploy more."
  +-> Link to upgrade path
```

**Acceptance:**
- [ ] First deployment succeeds
- [ ] Second deployment blocked with clear message
- [ ] Error message includes plan name and limit
- [ ] Upgrade path accessible
- [ ] After upgrade, deployment succeeds

---

## Appendix: State Machine Reference

### Deployment States
```
pending -> scheduled -> starting -> running -> stopping -> stopped -> deleting -> deleted
                           |           |
                           v           v
                         failed <------+
                           |
                           +-> starting (retry)
                           +-> deleting (give up)
```

### Cloud Provision States
```
pending -> creating -> configuring -> ready -> destroying -> destroyed
   |          |            |                       |
   v          v            v                       v
 failed <-----+------------+-----------------------+
   |
   +-> pending (retry)
   +-> destroying (give up)
```
