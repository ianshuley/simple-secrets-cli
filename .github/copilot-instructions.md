# Copilot Instructions

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
