# Copilot Instructions

## 🚨 CRITICAL VERSIONING REMINDER 🚨

### ⛔ NEVER SUGGEST VERSION CHANGES FOR DEVELOPMENT WORK ⛔
- **User is doing DEVELOPMENT work** - NOT releasing software to the public
- **Official releases created**: 0 (ZERO)
- **When to suggest versioning**: ONLY when user explicitly says "I want to release this to the public"
- **Development workflow**: ALWAYS use `make dev` (shows dev-abc123)
- **DO NOT suggest**: Version number changes, git tags, "release" commands during normal development
- **User understands**: Versioning is for official releases only, not PRs or daily development

### 📋 User's Versioning Workflow (ALREADY ESTABLISHED):
- **Daily work**: `make dev` → shows `dev-abc123`
- **PRs/features**: No version changes needed
- **Official release** (when ready): `make release VERSION=vX.Y.Z` + git tag
- **Current status**: Still in development phase, no releases yet

**STOP CONFUSING THE USER WITH VERSIONING DURING DEVELOPMENT!**

---

## Code Style Guidelines

### Avoid `else` in Garbage Collected Languages
- **Principle**: If you're trying to use `else` in a GC language, you're likely overlooking a better way to solve the problem
- **Rationale**: GC languages offer more flexible control flow patterns that often eliminate the need for explicit `else` branches
- **Preferred patterns**:
  - Early returns to reduce nesting
  - Default values with conditional overrides
  - Guard clauses for validation
  - Pattern matching where available

### Extract Methods and Variables Frequently
- **Principle**: Extract methods and variables frequently to take control of vernacular back from the language's syntax
- **Goal**: Make code read more book-like at higher levels of abstraction
- **Abstraction Level Rule**: Objects used in a method should exist on the same level of abstraction
  - If working with primitives, only use primitive manipulation
  - If working with domain concepts, only use domain-level operations
  - Don't mix low-level string manipulation with high-level business logic in the same method

**✅ Extract meaningful operations:**
```go
// Instead of inline primitive manipulation
func ProcessUser(rawData string) error {
    parts := strings.Split(rawData, ",")
    if len(parts) != 2 {
        return errors.New("invalid format")
    }
    name := strings.TrimSpace(parts[0])
    email := strings.TrimSpace(parts[1])
    if !strings.Contains(email, "@") {
        return errors.New("invalid email")
    }
    // ... business logic
}

// Extract to maintain abstraction levels
func ProcessUser(rawData string) error {
    userData, err := parseUserData(rawData)
    if err != nil {
        return err
    }
    return createUser(userData)
}

func parseUserData(rawData string) (UserData, error) {
    name, email := extractNameAndEmail(rawData)
    if !isValidEmail(email) {
        return UserData{}, errors.New("invalid email")
    }
    return UserData{Name: name, Email: email}, nil
}
```

**✅ Extract meaningful variables:**
```go
// Instead of magic values and unclear intent
if len(users) > 5 && time.Since(lastCheck) > 3600*time.Second {
    processUsers(users)
}

// Extract to express intent clearly
maxUsersBeforeProcessing := 5
hourInSeconds := 3600 * time.Second
shouldProcessUsers := len(users) > maxUsersBeforeProcessing &&
                      time.Since(lastCheck) > hourInSeconds
if shouldProcessUsers {
    processUsers(users)
}
```

### Examples

**❌ Avoid this pattern:**
```go
if condition {
    doSomething()
} else {
    doSomethingElse()
}
```

**✅ Prefer these patterns:**
```go
// Early return pattern
if condition {
    doSomething()
    return
}
doSomethingElse()

// Default with override pattern
value := defaultValue
if condition {
    value = specialValue
}
useValue(value)

// Guard clause pattern
if !isValid(input) {
    return fmt.Errorf("invalid input")
}
processInput(input)
```

### Comments Are an Admission of Failure
- **Principle**: A comment is an admission of failure to properly express yourself in code
- **Rationale**: If you need a comment, it usually means something needs to be abstracted and you're likely mixing abstraction layers
- **When comments are acceptable**:
  - Explaining complex regex patterns that are inherently non-intuitive
  - Documenting external API requirements or constraints
  - Explaining business rules that come from external requirements documents
- **When to extract instead of comment**:
  - If you're explaining what the code does → extract to a well-named method
  - If you're explaining why variables exist → extract to meaningful variable names
  - If you're explaining complex logic → break into smaller, self-documenting methods

**❌ Instead of commenting what code does:**
```go
// Check if user has admin privileges and token is not expired
if user.Role == "admin" && time.Now().Before(user.TokenExpiry) {
    // Allow access to admin features
    return true
}
```

**✅ Make the code self-documenting:**
```go
func canAccessAdminFeatures(user User) bool {
    return user.hasAdminPrivileges() && user.hasValidToken()
}

func (u User) hasAdminPrivileges() bool {
    return u.Role == "admin"
}

func (u User) hasValidToken() bool {
    return time.Now().Before(u.TokenExpiry)
}
```

### Other Style Guidelines
- Use meaningful variable names that express intent
- Prefer composition over inheritance
- Keep functions small and focused on a single responsibility
- Use early returns to reduce cognitive load
- Favor explicit error handling over silent failures
- Extract methods to maintain consistent abstraction levels within functions
- Extract variables to replace magic numbers and express business intent
- Prioritize readability: code should read like well-written prose at the appropriate abstraction level

## Go-Specific Guidelines
- Use `gofmt` formatting consistently
- Follow Go naming conventions (PascalCase for exports, camelCase for internal)
- Prefer interfaces for testability
- Use context.Context for cancellation and timeouts
- Handle errors explicitly, don't ignore them

## Architecture Principles
- Single Responsibility Principle
- Dependency injection for testability
- Clear separation between business logic and I/O operations
- Immutable data structures where possible

---

## 🧪 Testing & Validation Frameworks

This repository includes comprehensive AI-driven testing and validation frameworks designed to ensure code quality and merge-readiness. These frameworks are stored as dotfiles and should be used by AI assistants when testing, validating, or preparing code for merge.

### Framework Files Overview

#### `.opus-testing-framework.md` - Persona-Based Testing
**Purpose**: AI simulates different user personas testing their perception and experience with the application
**When to use**:
- User requests "run opus testing" or "test different personas"
- Need comprehensive user experience validation
- Want to discover edge cases through creative, persona-based exploration

**Key Features**:
- Multiple testing personas (New User, Power User, Malicious Actor, DevOps Engineer, etc.)
- Creative destruction and edge case discovery
- Security-focused adversarial testing
- User experience and intuition validation
- Pattern recognition and behavioral exploration

**Example Usage**:
```
"Run the opus testing framework focusing on the security auditor persona"
"Use opus methodology to test the backup/restore user experience"
```

#### `.testing-framework.md` - Systematic Functional Testing
**Purpose**: AI simulates rigorous human testing for semantic correctness and comprehensive functionality validation
**When to use**:
- User requests "run testing framework" or "validate functionality"
- Need systematic regression testing
- Want comprehensive coverage of all features and edge cases

**Key Features**:
- Systematic test coverage of all major functionality
- Regression testing protocols
- Error handling validation
- Performance and reliability testing
- Integration testing scenarios

**Example Usage**:
```
"Execute the testing framework to validate all CRUD operations"
"Run full testing framework before merge"
```

#### `.copilot-consistency-checklist.md` - Documentation Synchronization
**Purpose**: Ensures all documentation, tests, code, and examples remain synchronized after changes
**When to use**:
- After making code changes that affect functionality
- Before merging any feature or bug fix
- User requests "consistency check" or "sync documentation"

**Key Features**:
- Pre-change analysis of scope and impact
- Documentation synchronization validation
- Code quality review against coding standards
- Cross-reference validation between files
- Test coverage verification

**Example Usage**:
```
"Run consistency checklist after adding new CLI command"
"Validate documentation sync before merge"
```

#### `.pre-merge-checklist.md` - Complete Validation Process
**Purpose**: Comprehensive pre-merge validation combining all frameworks into a single, repeatable process
**When to use**:
- User requests "pre-merge validation" or "run pre-merge checklist"
- Before any code merge to ensure complete validation
- Need full quality assurance process

**Key Features**:
- Sequential execution of all validation frameworks
- SOLID principles and code quality review
- Cross-reference validation between all framework files
- Clear go/no-go merge decision criteria

### Framework Usage Guidelines

#### For AI Assistants

**Sequential Framework Execution**:
1. **Development Phase**: Use individual frameworks as needed during development
2. **Pre-Merge Phase**: Always use `.pre-merge-checklist.md` for complete validation
3. **Post-Change Phase**: Always use `.copilot-consistency-checklist.md` to sync documentation

**When User Requests Testing**:
- "run opus testing" → Use `.opus-testing-framework.md`
- "test functionality" or "run tests" → Use `.testing-framework.md`
- "consistency check" → Use `.copilot-consistency-checklist.md`
- "pre-merge" or "ready to merge" → Use `.pre-merge-checklist.md`

**Framework Integration**:
- Each framework file cross-references the others appropriately
- Use frameworks in combination for comprehensive validation
- Report results from each framework clearly
- Block merges if any framework reports critical issues

#### Example AI Assistant Workflow

```
User: "I've added a new CLI command, can you validate it's ready to merge?"

AI Response:
1. Execute .pre-merge-checklist.md (which includes):
   - .opus-testing-framework.md (persona-based testing)
   - .testing-framework.md (systematic validation)
   - .copilot-consistency-checklist.md (documentation sync)
   - SOLID principles review
   - Cross-reference validation

2. Report clear pass/fail for each phase
3. Provide go/no-go merge recommendation
```

### Framework Maintenance

- **Keep frameworks synchronized**: Changes to one should be reflected in cross-references
- **Update examples**: Ensure all framework files have current, relevant examples
- **Validate cross-references**: All framework files should properly reference each other
- **Maintain consistency**: Coding standards in this file should align with validation in frameworks

---

## 🎯 Quality Assurance Workflow

**For every significant change**:
1. Follow coding standards above during development
2. Run appropriate testing framework(s) during development
3. Execute `.copilot-consistency-checklist.md` after changes
4. Execute `.pre-merge-checklist.md` before merge
5. Only merge if all frameworks report success

This ensures every merge maintains the highest standards of quality, consistency, and maintainability.
