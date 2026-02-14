# UJ5: "I want to create and sell a template"

**Persona:** Creator
**Goal:** Package a docker-compose application and publish it for others to deploy
**Preconditions:** Signed in.

## Story

1. Signs in. Navigates to "App Templates" via sidebar. Sees their existing templates (if any) with status badges (Draft/Published).
2. Clicks "Create Template." A dialog opens.
3. Fills in template details: name, description, version (semver X.Y.Z), docker-compose spec, and optionally category, variables, and price.
4. Submits. Template is created with "Draft" status. Appears in the template list with a "Draft" badge.
5. Reviews the template details. Optionally edits fields to refine.
6. Satisfied with the template. Clicks "Publish" on the template card.
7. Badge changes from "Draft" to "Published." "Publish" button disappears (already published). Edit and Delete remain available.
8. Navigates to Marketplace. Sees their template listed alongside all other published templates. Other users can now see it.
9. Returns to Dashboard. Can see deployment count for their template as others deploy it.

## Pages & Features Touched

1. Login (`/login`)
2. App Templates list (`/templates`)
3. Create Template dialog
4. Template card actions (Edit, Publish, Delete)
5. Marketplace (`/marketplace`)
6. Dashboard (`/dashboard`)

## Acceptance Criteria

- [ ] Template list shows only the current user's templates (owner scoping)
- [ ] Create dialog validates: name (3-100 chars, alphanumeric + spaces + hyphens), version (semver X.Y.Z), compose spec (required)
- [ ] Slug auto-generated from template name
- [ ] Resource defaults applied on creation (cpu=0, memory=0, disk=0, price=0)
- [ ] New template created as "Draft"
- [ ] Publish transitions template to published status
- [ ] Published template is immediately visible in the Marketplace
- [ ] Other users can see and deploy published templates
- [ ] Creator can still Edit and Delete after publishing
- [ ] Draft templates are NOT visible in the Marketplace

## Edge Cases

- **Duplicate template name:** Validation error, not a crash.
- **Invalid compose spec:** Template still creates (compose spec validation is basic), but deployment from it may fail at start time.
- **Delete template with active deployments:** Error surfaced (FK constraint). Template cannot be deleted while deployments reference it.
- **Unpublish:** Not currently supported (by design). Creator can delete and recreate if needed.
