# ADR-000: STC (Spec → Test → Code) Methodology

## Status
Accepted (Foundational - cannot be changed)

## Context

Software projects accumulate technical debt when:
1. Code is written without clear requirements
2. Tests are written after code (or not at all)
3. Documentation drifts from implementation
4. Developers make inconsistent decisions

We need a methodology that:
- Prevents these issues by design
- Is simple enough to follow consistently
- Creates self-documenting code
- Enables future sessions with zero context to continue correctly

## Decision

We adopt **STC (Spec → Test → Code)** as our development methodology.

### The Flow

```
SPEC ─────► TEST ─────► CODE
  │           │           │
  │           │           │
  └───────────┴───────────┘
        Always In Sync
```

### Rules (Non-Negotiable)

1. **Every feature starts with a spec**
   - Written in markdown in `specs/` directory
   - Contains: requirements, acceptance criteria, edge cases, NOT supported

2. **Tests are written based on spec, before code**
   - Each spec item has corresponding test(s)
   - Tests fail initially (TDD red phase)

3. **Code is written to pass tests**
   - Implementation follows naturally
   - Tests pass (TDD green phase)

4. **Changes flow through the chain**
   - If requirements change → update spec → update tests → update code
   - Never skip steps

5. **Sync is mandatory**
   - At any point: spec ↔ tests ↔ code must match
   - Drift is a bug

## Consequences

### Positive
- **Self-documenting**: Specs explain "what and why", tests explain "how"
- **Zero-context continuation**: New session reads specs, understands everything
- **Reduced bugs**: Tests exist before code
- **Clear boundaries**: "Not Supported" in specs prevents scope creep
- **Refactoring safety**: Tests catch regressions

### Negative
- **Slower initial velocity**: Must write spec + tests before code
- **More files**: Three artifacts per feature instead of one
- **Discipline required**: Easy to skip steps under pressure

### Failure Modes (What Goes Wrong If Violated)

| Violation | Consequence | Recovery Cost |
|-----------|-------------|---------------|
| Code without spec | Future dev doesn't know requirements, makes wrong assumptions | High - must reverse-engineer intent |
| Code without tests | Bugs introduced, refactoring unsafe | High - must add tests retroactively |
| Spec not updated after code change | New session implements wrong behavior | Medium - must audit all code |
| Tests not matching spec | False confidence in code correctness | Medium - must rewrite tests |
| "Not Supported" section missing | Features added that shouldn't exist | Very High - must remove code + tests |

## How to Apply

### Starting a New Feature

1. **Create spec file**: `specs/features/F###-feature-name.md`
2. **Write the spec**: Use template from `specs/README.md`
3. **Create test file**: `internal/core/.../feature_test.go`
4. **Write failing tests**: Based on spec's acceptance criteria
5. **Create code file**: `internal/core/.../feature.go`
6. **Implement**: Make tests pass
7. **Verify sync**: Spec ↔ Tests ↔ Code all match

### Modifying Existing Feature

1. **Read current spec**: Understand existing requirements
2. **Update spec first**: Add/change/remove requirements
3. **Update tests**: Reflect spec changes
4. **Update code**: Make tests pass
5. **Verify sync**: All three match

### New Session Onboarding

1. Read `CLAUDE.md` (points to key specs)
2. Read relevant specs for the task
3. Review existing tests to understand behavior
4. Continue following STC

## Verification

At any time, you can verify sync:

```bash
# 1. Check specs exist for implemented features
ls specs/domain/
ls specs/features/

# 2. Check tests exist
make test-unit

# 3. Check coverage (should be >90% for core)
make coverage
```

## References

- Test-Driven Development (TDD) by Kent Beck
- Specification by Example by Gojko Adzic
- Documentation-Driven Development
