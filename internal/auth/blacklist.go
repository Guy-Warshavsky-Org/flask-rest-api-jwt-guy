package auth

import "sync"

// Blacklist provides a thread-safe in-memory set for storing revoked JWT token
// identifiers (JTIs). This mirrors Flask's module-level jwt_token_blacklist set(),
// but uses a sync.RWMutex for concurrent safety since Gin serves requests
// concurrently (unlike Flask's typical single-threaded dev server).
//
// Like the Flask implementation, this blacklist is volatile — revoked tokens
// are lost on server restart.
type Blacklist struct {
	mu     sync.RWMutex
	tokens map[string]struct{}
}

// NewBlacklist creates a new empty Blacklist.
func NewBlacklist() *Blacklist {
	return &Blacklist{
		tokens: make(map[string]struct{}),
	}
}

// Add marks a token JTI as revoked.
func (b *Blacklist) Add(jti string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens[jti] = struct{}{}
}

// Contains checks whether a token JTI has been revoked.
func (b *Blacklist) Contains(jti string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, exists := b.tokens[jti]
	return exists
}

// Reset clears all revoked tokens. This is primarily used in tests
// to ensure a clean state between test cases.
func (b *Blacklist) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens = make(map[string]struct{})
}

// TokenBlacklist is the global blacklist instance, analogous to
// Flask's module-level jwt_token_blacklist = set().
var TokenBlacklist = NewBlacklist()
