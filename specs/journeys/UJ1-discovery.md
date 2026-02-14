# UJ1: "I want to try this platform"

**Persona:** Visitor
**Goal:** Understand what Hoster offers and decide whether to commit
**Preconditions:** None — anonymous user arriving from a search engine or shared link

## Story

1. Lands on the homepage from a search result or direct link. Sees hero copy ("Deploy apps to your own servers"), the 3-step guide (Deploy, Bring Servers, Create Templates), and feature cards (Monitoring, SSH Keys, Self-Hosted).
2. Clicks "Browse Apps" in the hero or top navigation. Arrives at the Marketplace page.
3. Sees published templates grouped by category. Uses category pills to filter (e.g. "Database", "Web Apps") and searches by name.
4. Clicks a template card. Arrives at the Template Detail page. Reads the description, included services (parsed from compose spec), full Docker Compose specification, and price.
5. Clicks "Deploy Now." An "Authentication Required" prompt appears — not a crash, not a redirect to a broken page.
6. **Decision point:** Signs up (→ UJ2), or leaves.

## Pages & Features Touched

1. Homepage (`/`)
2. Marketplace (`/marketplace`)
3. Template Detail (`/marketplace/{template_id}`)
4. Auth Required prompt (modal or inline)

## Acceptance Criteria

- [ ] Homepage loads without authentication errors; no sidebar navigation shown
- [ ] "Browse Apps" and "Get Started" CTAs link to correct destinations
- [ ] Marketplace shows only published templates (drafts hidden)
- [ ] Category filter and search work on marketplace
- [ ] Template detail renders all fields: name, version, description, services, compose spec, price
- [ ] "Deploy Now" on template detail shows auth prompt for anonymous visitors
- [ ] Page load is fast (<2s) with no layout shift

## Edge Cases

- **No published templates:** Marketplace shows an empty state with a message, not a blank page.
- **Invalid template ID in URL:** 404 page, not a crash.
- **Visitor bookmarks template detail:** Page is accessible without login on return visit.
- **JavaScript disabled/slow load:** Core content still readable before JS hydrates.
