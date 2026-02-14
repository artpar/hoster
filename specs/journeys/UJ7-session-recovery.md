# UJ7: "My session expired, I need to get back in"

**Persona:** Customer returning after 24h+ inactivity
**Goal:** Re-authenticate and continue where they left off
**Preconditions:** Previously signed in. JWT has expired.

## Story

1. Returns to the app by clicking a bookmarked deployment URL (e.g. `/deployments/depl_abc123`).
2. Page attempts to load. API call is made with the expired JWT.
3. APIGate validates JWT, finds it expired, returns 401 Unauthorized.
4. Frontend API client intercepts the 401. Auth state is cleared (token removed from localStorage).
5. `ProtectedRoute` component detects unauthenticated state. Redirects to `/login` with the intended destination preserved.
6. Signs in with email and password. New JWT issued and stored.
7. Redirected back to the originally intended page (`/deployments/depl_abc123`).
8. Page loads normally. Continues working as if nothing happened.

## Pages & Features Touched

1. Any protected page (initial deep link)
2. API client 401 interceptor
3. Login page (`/login`)
4. `ProtectedRoute` redirect logic
5. Original destination (post-login redirect)

## Acceptance Criteria

- [ ] Expired JWT triggers 401 from APIGate, not a server error
- [ ] 401 response clears auth state in the frontend
- [ ] User sees login page, not a raw error or blank screen
- [ ] Login page preserves the intended destination URL
- [ ] After re-login, user is redirected to the page they originally requested
- [ ] New JWT works for all subsequent API calls
- [ ] No stale data displayed from the expired session

## Edge Cases

- **User was on a form with unsaved data:** Form state is lost on redirect to login. This is acceptable — session expiry is a hard boundary.
- **Multiple tabs open:** Each tab independently detects 401 and redirects to login. Re-login in one tab does not auto-refresh others.
- **Token tampered/invalid (not just expired):** Same 401 flow — cleared and redirected to login.
- **APIGate is down:** Network error, not 401. Frontend shows connection error, not login redirect.
