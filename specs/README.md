# Specifications

This directory contains the source of truth for the Hoster project. All features, domain models, and architectural decisions are documented here before implementation.

## STC Methodology (Spec → Test → Code)

Every feature follows this flow:

```
1. SPEC   → Write the specification first
2. TEST   → Write tests that verify the spec
3. CODE   → Implement code that passes tests
```

**Rules:**
- Never write code without a spec
- Never write code without tests
- If the spec changes, update tests first, then code
- The trifecta (Spec ↔ Test ↔ Code) must always be in sync

## Directory Structure

```
specs/
├── README.md           # This file
├── domain/             # Domain model specifications
│   ├── template.md     # Template entity
│   ├── deployment.md   # Deployment entity
│   ├── node.md         # Node entity
│   └── customer.md     # Customer entity
├── features/           # Feature specifications
│   ├── F001-*.md       # Feature specs with ID prefix
│   └── ...
└── decisions/          # Architecture Decision Records
    ├── ADR-001-*.md    # ADR with ID prefix
    └── ...
```

## Spec File Format

### Domain Specs (`domain/*.md`)

```markdown
# Entity Name

## Overview
Brief description of what this entity represents.

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id    | UUID | Yes      | Unique identifier |
| ...   | ...  | ...      | ... |

## Invariants
- Rules that must always be true
- Validation constraints
- Business rules

## Behaviors
- State transitions
- Computed properties
- Side effects (if any)

## Not Supported
- What this entity does NOT do (and why)

## Examples
Concrete examples of valid/invalid instances.
```

### Feature Specs (`features/F###-*.md`)

```markdown
# F001: Feature Name

## User Story
As a [role], I want [capability] so that [benefit].

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2
- [ ] ...

## Inputs
What data/parameters the feature accepts.

## Outputs
What the feature produces.

## Edge Cases
- Case 1: ...
- Case 2: ...

## Error Handling
How errors are handled and reported.

## Not Supported
What this feature explicitly does NOT do.

## Dependencies
Other features or components this depends on.
```

### Architecture Decision Records (`decisions/ADR-###-*.md`)

```markdown
# ADR-001: Decision Title

## Status
Accepted | Proposed | Deprecated | Superseded by ADR-XXX

## Context
What is the issue that we're seeing that motivates this decision?

## Decision
What is the change that we're proposing or accepting?

## Consequences

### Positive
- Benefit 1
- Benefit 2

### Negative
- Tradeoff 1
- Tradeoff 2

### Neutral
- Observation 1

## Alternatives Considered
Other options that were evaluated and why they were rejected.
```

## Linking Specs to Tests

Each spec should reference its corresponding test file:

```markdown
## Tests
- `internal/core/domain/template_test.go` - Unit tests
- `tests/e2e/template_test.go` - E2E tests
```

## Versioning

Specs are versioned with the code. When a spec changes:
1. Update the spec document
2. Update corresponding tests
3. Update the code
4. Commit all three together

## Review Process

Before a feature is implemented:
1. Spec is written and reviewed
2. Edge cases are identified
3. "Not Supported" section is explicit
4. Dependencies are documented
