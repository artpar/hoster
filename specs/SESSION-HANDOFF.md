# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

## Phase 1: Context Loading (MANDATORY)

### Step 1: Read Core Documents (in order)

```bash
# Run this to verify project exists
ls -la
```

Read these files in this exact order:

1. **`CLAUDE.md`** - Project memory, decisions, current state
2. **`specs/README.md`** - How specs work
3. **`specs/decisions/ADR-000-stc-methodology.md`** - THE methodology
4. **`specs/decisions/ADR-001-docker-direct.md`** - Architecture decision
5. **`specs/decisions/ADR-002-values-as-boundaries.md`** - Code organization

### Step 2: Verify Understanding

After reading, you should know:
- [ ] What is Hoster? (Deployment marketplace platform)
- [ ] What is STC? (Spec → Test → Code)
- [ ] What is "Values as Boundaries"? (Pure core, thin shell)
- [ ] Where do specs go? (`specs/` directory)
- [ ] Where does core code go? (`internal/core/`)
- [ ] Where does I/O code go? (`internal/shell/`)
- [ ] What libraries to use? (Listed in CLAUDE.md)

### Step 3: Verify Tests Pass

```bash
make test
```

If tests fail, something is broken. Fix before proceeding.

---

## Phase 2: Task Understanding

### Step 4: Check Current Status

Read `CLAUDE.md` section "Current Implementation Status" to understand:
- What's DONE
- What's TODO
- What's BLOCKED

### Step 5: Check Agile Project

```
Use mcp__agile__workflow_execute with workflow: "backlog_status"
and project_id: "HOSTER" to see current tasks.
```

### Step 6: Understand User's Request

Now you can ask the user what they want to do. Compare against:
- What's already implemented (don't redo)
- What's in the TODO list
- What's explicitly NOT supported (don't implement)

---

## Phase 3: Before Making Changes

### Step 7: Identify Relevant Specs

For ANY change:
1. Find the relevant spec in `specs/`
2. If no spec exists → WRITE SPEC FIRST
3. If spec exists → READ IT before changing code

### Step 8: Pre-Flight Checklist

Before writing ANY code, verify:

- [ ] Spec exists for this feature/change
- [ ] I understand the spec's acceptance criteria
- [ ] I know what's "NOT Supported" (don't implement those)
- [ ] Tests exist (or I'll write them first)
- [ ] I know which directory: `internal/core/` or `internal/shell/`
- [ ] I'm using the approved libraries (check CLAUDE.md)

---

## Phase 4: Making Changes (STC Flow)

### For New Features

```
1. SPEC   → Create specs/features/F###-name.md
2. TEST   → Create internal/core/xxx/feature_test.go (failing tests)
3. CODE   → Create internal/core/xxx/feature.go (make tests pass)
4. VERIFY → make test
```

### For Bug Fixes

```
1. SPEC   → Update spec if behavior was wrong
2. TEST   → Add test that demonstrates the bug
3. CODE   → Fix code to pass test
4. VERIFY → make test
```

### For Refactoring

```
1. VERIFY → make test (all pass before)
2. REFACTOR → Make changes
3. VERIFY → make test (all pass after)
```

---

## Phase 5: After Making Changes

### Step 9: Verify Sync

After any change:
- [ ] Spec still matches implementation
- [ ] Tests still match spec
- [ ] All tests pass (`make test`)

### Step 10: Update CLAUDE.md

If you:
- Completed a TODO item → Move to DONE
- Made a new decision → Document in CLAUDE.md
- Added new spec → Reference in CLAUDE.md
- Changed architecture → Update ADR or create new one

### Step 11: Update Agile Project

```
Use mcp__agile__task_transition to update task status.
```

---

## Common Mistakes to Avoid

| Mistake | Why It's Bad | Prevention |
|---------|--------------|------------|
| Writing code without reading specs | You'll implement wrong behavior | Always read spec first |
| Writing code before tests | No safety net for refactoring | Write test first, see it fail |
| Putting I/O in `internal/core/` | Breaks architecture, needs mocks | Keep core pure |
| Using different library | Inconsistency, harder to maintain | Check CLAUDE.md library list |
| Implementing "NOT Supported" items | Scope creep, wasted effort | Read spec's NOT Supported section |
| Skipping `make test` | Broken code goes unnoticed | Run after every change |
| Not updating CLAUDE.md | Next session loses context | Update after significant changes |

---

## Quick Reference: Where Things Go

| What | Where | Example |
|------|-------|---------|
| Domain specs | `specs/domain/` | `template.md` |
| Feature specs | `specs/features/` | `F001-parse-compose.md` |
| ADRs | `specs/decisions/` | `ADR-001-docker-direct.md` |
| Pure logic | `internal/core/` | `domain/template.go` |
| I/O code | `internal/shell/` | `docker/client.go` |
| Unit tests | Same dir as code | `template_test.go` |
| E2E tests | `tests/e2e/` | `deploy_test.go` |
| Sample templates | `examples/` | `wordpress/compose.yml` |

---

## Emergency Recovery

### If Tests Are Failing

```bash
# See what's broken
make test 2>&1 | grep FAIL

# Check recent changes
git log --oneline -10

# Revert if needed
git checkout HEAD~1 -- <file>
```

### If You're Lost

1. Re-read `CLAUDE.md` from the beginning
2. Run `make test` to verify baseline
3. Read the specific spec for what you're working on
4. Ask user for clarification

### If Spec Doesn't Exist

DO NOT write code. Instead:

1. Create spec file
2. Write requirements
3. Write "NOT Supported" section
4. Get user confirmation
5. Then proceed with tests and code

---

## Session End Checklist

Before ending a session:

- [ ] All tests pass (`make test`)
- [ ] CLAUDE.md is updated with:
  - [ ] New DONE items
  - [ ] New TODO items
  - [ ] Any new decisions
- [ ] Agile project updated
- [ ] User informed of current state
