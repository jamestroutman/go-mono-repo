# Spec-Driven Development Guide

## Overview

This monorepo follows a spec-driven development approach where features are thoroughly specified before implementation. This ensures alignment between stakeholders, reduces rework, and provides clear documentation.

## Documentation Structure

```
go-mono-repo/
├── docs/                                    # Cross-cutting documentation
│   ├── ARCHITECTURE.md                     # Overall system architecture
│   ├── PROTOBUF_PATTERNS.md               # Shared protobuf conventions
│   ├── SERVICE_DEVELOPMENT.md             # Service creation guide
│   ├── SPEC_DRIVEN_DEVELOPMENT.md         # This guide
│   └── SPEC_TEMPLATE.md                   # Template for new specs
│
└── services/
    └── {domain}/
        └── {service-name}/
            ├── docs/
            │   ├── README.md               # Service-specific documentation
            │   ├── specs/                  # Feature specifications
            │   │   ├── 001-feature-a.md
            │   │   ├── 002-feature-b.md
            │   │   └── 003-feature-c.md
            │   ├── adrs/                   # Architecture Decision Records
            │   │   ├── 001-storage-choice.md
            │   │   └── 002-api-design.md
            │   └── runbooks/               # Operational guides
            │       ├── deployment.md
            │       └── troubleshooting.md
            └── ...
```

## Specification Lifecycle

### 1. Draft Phase
- Create spec using [SPEC_TEMPLATE.md](./SPEC_TEMPLATE.md)
- Number specs sequentially (001, 002, 003...)
- Include problem statement and success metrics
- Define scope clearly (in/out of scope)

### 2. Review Phase
- Technical review by team members
- Stakeholder review for requirements
- Update spec based on feedback
- Document decisions in decision log

### 3. Approved Phase
- Final sign-off from reviewers
- Spec becomes source of truth
- No implementation without approved spec
- Changes require spec updates

### 4. Implemented Phase
- Code matches specification
- Tests verify acceptance criteria
- Documentation updated
- Spec marked as implemented

## Creating a New Specification

### Step 1: Identify the Need
Before creating a spec, ensure:
- The feature provides clear value
- It aligns with service boundaries
- No existing spec covers this need

### Step 2: Create Spec File
```bash
# Navigate to service docs
cd services/{domain}/{service-name}/docs/specs

# Copy template
cp ../../../../docs/SPEC_TEMPLATE.md {number}-{feature-name}.md

# Example:
cp ../../../../docs/SPEC_TEMPLATE.md 001-account-management.md
```

### Step 3: Fill Out Sections

#### Essential Sections
1. **Executive Summary** - What and why in 2-3 sentences
2. **Problem Statement** - Current vs desired state
3. **User Stories** - Who needs what and why
4. **Technical Design** - How it will work
5. **Acceptance Criteria** - Definition of done

#### Supporting Sections
- Implementation plan with phases
- Dependencies and risks
- Performance requirements
- Security considerations
- Testing strategy

### Step 4: Review Process

1. **Self Review**
   - Is the problem clearly stated?
   - Are success metrics measurable?
   - Is the technical design complete?
   - Are edge cases considered?

2. **Peer Review**
   - Create a pull request with the spec
   - Tag relevant reviewers
   - Address feedback in the spec
   - Update decision log

3. **Stakeholder Review**
   - Share with product/business stakeholders
   - Validate requirements and priorities
   - Confirm success metrics
   - Get approval to proceed

## Specification Standards

### Naming Convention
```
{number}-{feature-name}.md

Examples:
001-account-management.md
002-transaction-processing.md
003-audit-logging.md
```

### Numbering Rules
- Use 3-digit numbers (001, 002, etc.)
- Sequential within each service
- Never reuse numbers
- Deprecated specs keep their numbers

### Content Guidelines

#### Focus on What, Not How
Specs should define requirements and interfaces, not implementation details:

❌ "Use a HashMap to store accounts"
✅ "Provide O(1) account lookup by ID"

#### Be Specific
❌ "The system should be fast"
✅ "Response time < 100ms for 99th percentile"

#### Be Measurable
❌ "Improve user experience"
✅ "Reduce error rate from 5% to < 1%"

#### Be Complete
❌ "Handle errors appropriately"
✅ "Return INVALID_ARGUMENT for missing fields, NOT_FOUND for unknown IDs"

#### Be Realistic
❌ "Support infinite scale"
✅ "Support 10,000 requests/second with horizontal scaling"

## Linking Specs to Code

### Required Code References

Every implementation MUST reference its governing specification in code comments. This creates traceability between requirements and implementation.

### In Service Methods
```go
// CreateAccount implements account creation
// Spec: docs/specs/001-account-management.md
func (s *server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
    // Implementation...
}

// GetAccount retrieves account details
// Spec: docs/specs/001-account-management.md#story-2-query-account-balance
func (s *server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
    // Implementation...
}
```

### In Proto Files
```protobuf
// Account management service
// Spec: docs/specs/001-account-management.md
service Ledger {
    // Account operations
    rpc CreateAccount (CreateAccountRequest) returns (CreateAccountResponse) {}
    rpc GetAccount (GetAccountRequest) returns (GetAccountResponse) {}
}
```

### In Tests
```go
// TestCreateAccount verifies account creation
// Spec: docs/specs/001-account-management.md#story-1-create-account
func TestCreateAccount(t *testing.T) {
    // Test validates acceptance criteria from story 1
    // Test implementation...
}

// TestGetAccountBalance verifies balance query
// Spec: docs/specs/001-account-management.md#story-2-query-account-balance
func TestGetAccountBalance(t *testing.T) {
    // Test validates acceptance criteria from story 2
    // Test implementation...
}
```

### Reference Format

Use this consistent format for spec references:
```
// Spec: docs/specs/{number}-{feature}.md[#section]
```

Examples:
- `// Spec: docs/specs/001-account-management.md`
- `// Spec: docs/specs/001-account-management.md#user-stories`
- `// Spec: docs/specs/002-transaction-processing.md#error-handling`

### Benefits of Code References

1. **Traceability**: Direct link from code to requirements
2. **Context**: Developers understand the "why" behind the code
3. **Navigation**: Easy to find the governing specification
4. **Validation**: Can verify implementation matches spec
5. **Documentation**: Self-documenting code

## Architecture Decision Records (ADRs)

For significant technical decisions, create ADRs:

```markdown
# ADR-001: Use In-Memory Storage for MVP

## Status
Accepted

## Context
We need to decide on storage for the ledger service MVP.

## Decision
Use in-memory storage with mutex protection.

## Consequences
- Pros: Fast development, easy testing
- Cons: No persistence, limited scale

## Alternatives Considered
- PostgreSQL: Too heavy for MVP
- Redis: Additional infrastructure
```

## Spec Review Checklist

### Technical Review
- [ ] API design follows protobuf patterns
- [ ] Error handling is comprehensive
- [ ] Performance targets are realistic
- [ ] Security concerns addressed
- [ ] Testing strategy is complete

### Business Review
- [ ] Problem statement is clear
- [ ] Success metrics align with goals
- [ ] User stories capture requirements
- [ ] Scope is well-defined
- [ ] Timeline is realistic

### Implementation Review
- [ ] Spec matches implementation
- [ ] All acceptance criteria met
- [ ] Tests verify requirements
- [ ] Documentation updated
- [ ] Monitoring in place

## Documentation Requirements

Specs should reference local project documentation only:

### Local Documentation
- ✅ Service README.md files
- ✅ CLAUDE.md for AI assistance patterns
- ✅ Project docs in /docs folder
- ✅ Service docs in service folders

### External Documentation
- ❌ External wikis (use Confluence link in header instead)
- ❌ External design docs (link in references section)
- ❌ Third-party documentation (link in references)

The Confluence link in the spec header serves as the bridge to external documentation.

## Benefits of Spec-Driven Development

### For Developers
- Clear requirements before coding
- Fewer mid-development changes
- Better technical decisions
- Reduced rework

### For Teams
- Shared understanding
- Parallel development possible
- Better estimation
- Knowledge documentation

### For Business
- Predictable delivery
- Measurable outcomes
- Risk identification
- Quality assurance

## Common Pitfalls

### Over-Specification
Don't specify implementation details that might change:
- Specific algorithms (unless critical)
- Internal data structures
- Private methods

### Under-Specification
Don't leave critical details undefined:
- API contracts
- Error conditions
- Performance requirements
- Security constraints

### Spec Abandonment
Keep specs updated:
- Update when requirements change
- Mark deprecated sections
- Link to superseding specs
- Maintain decision log

## Tools and Automation

### Spec Validation
Consider tools for:
- Markdown linting
- Link checking
- Template compliance
- Review automation

### Code Generation
Where applicable:
- Generate proto from specs
- Generate test cases
- Generate documentation
- Generate monitoring

## Examples

### Good Specifications
- [Account Management](../services/treasury-services/ledger-service/docs/specs/001-account-management.md) - Complete example with all sections

### Templates
- [Spec Template](./SPEC_TEMPLATE.md) - Starting point for new specs

## References

- [Architecture Overview](./ARCHITECTURE.md)
- [Service Development Guide](./SERVICE_DEVELOPMENT.md)
- [Protobuf Patterns](./PROTOBUF_PATTERNS.md)