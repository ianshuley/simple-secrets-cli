# TODO List

## CRITICAL - Fix Concurrency Race Condition

**Problem:** Opus AI testing framework discovered a race condition when multiple operations execute simultaneously.

**Details:** Under high concurrency (multiple goroutines performing operations like put/get/rotate simultaneously), the system exhibits non-deterministic behavior that could lead to data corruption or inconsistent state.

**Root Cause:** Likely shared state access without proper synchronization in core operations.

**Impact:** Could affect production deployments under heavy load or when automation tools perform parallel operations.

**Investigation Needed:**

- Identify which shared resources lack proper locking (secrets store, user store, file operations)
- Add appropriate mutexes or use Go's sync package for critical sections
- Test with high-concurrency scenarios to validate fix
- Consider atomic file operations vs in-memory locking strategies

**Priority:** HIGH - This affects data integrity under concurrent access patterns.



## Add clipboard flag

I'd like to add a --copy flag that pipes whatever information is being retrieved to the clipboard instead of to the console.



## Platform Command

Add `platform` command teasing what's coming next: The Simple Secrets Platform is coming soon! Visit <https://simple-secrets.io> to join the waitlist or learn more

## API Development

Start on API
