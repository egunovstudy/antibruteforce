package memory

import (
	"antibf/internal/model"
	"context"
	"fmt"
	"net"
	"sync"
)

// Repository is an in-memory repo mainly for tests and MVP runtime.
type Repository struct {
	mu    sync.RWMutex
	items map[model.NetworkListType][]*net.IPNet
}

// New creates repository.
func New() *Repository {
	return &Repository{items: map[model.NetworkListType][]*net.IPNet{
		model.ListTypeWhitelist: {},
		model.ListTypeBlacklist: {},
	}}
}

// Add stores a CIDR.
func (r *Repository) Add(_ context.Context, listType model.NetworkListType, cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("parse CIDR: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.items[listType] {
		if existing.String() == network.String() {
			return nil
		}
	}

	r.items[listType] = append(r.items[listType], network)

	return nil
}

// Remove deletes a CIDR.
func (r *Repository) Remove(_ context.Context, listType model.NetworkListType, cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("parse CIDR: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := r.items[listType][:0]
	for _, existing := range r.items[listType] {
		if existing.String() != network.String() {
			filtered = append(filtered, existing)
		}
	}
	copied := make([]*net.IPNet, len(filtered))
	copy(copied, filtered)
	r.items[listType] = copied

	return nil
}

// ContainsIP checks list membership.
func (r *Repository) ContainsIP(_ context.Context, listType model.NetworkListType, ip net.IP) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, network := range r.items[listType] {
		if network.Contains(ip) {
			return true, nil
		}
	}

	return false, nil
}
