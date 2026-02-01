package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	// DefaultTTL is the default time-to-live for pending commits
	DefaultTTL = 24 * time.Hour
)

// PendingCommit represents a commit that has been previewed but not yet approved
type PendingCommit struct {
	ID        string
	Message   string
	RepoPath  string
	CreatedAt time.Time
}

// Cache stores pending commits keyed by repo path
type Cache struct {
	mu      sync.RWMutex
	pending map[string]*PendingCommit // keyed by repo path
	byID    map[string]string         // maps ID to repo path for lookup
	ttl     time.Duration
}

// NewCache creates a new pending commit cache
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		pending: make(map[string]*PendingCommit),
		byID:    make(map[string]string),
		ttl:     ttl,
	}
}

// Store stores a pending commit, replacing any existing one for the same repo
func (c *Cache) Store(repoPath, message string) *PendingCommit {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove existing pending commit for this repo if any
	if existing, ok := c.pending[repoPath]; ok {
		delete(c.byID, existing.ID)
	}

	id := generateID()
	pc := &PendingCommit{
		ID:        id,
		Message:   message,
		RepoPath:  repoPath,
		CreatedAt: time.Now(),
	}

	c.pending[repoPath] = pc
	c.byID[id] = repoPath

	return pc
}

// Get retrieves a pending commit by ID, returns nil if not found or expired
func (c *Cache) Get(id string) *PendingCommit {
	c.mu.RLock()
	defer c.mu.RUnlock()

	repoPath, ok := c.byID[id]
	if !ok {
		return nil
	}

	pc, ok := c.pending[repoPath]
	if !ok {
		return nil
	}

	// Check if expired
	if time.Since(pc.CreatedAt) > c.ttl {
		return nil
	}

	return pc
}

// Remove removes a pending commit by ID
func (c *Cache) Remove(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	repoPath, ok := c.byID[id]
	if !ok {
		return
	}

	delete(c.byID, id)
	delete(c.pending, repoPath)
}

// generateID creates a random 8-character hex ID
func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
