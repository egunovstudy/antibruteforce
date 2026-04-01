package memory

import (
	"antibf/internal/model"
	"context"
	"net"
	"testing"
)

func TestRepository(t *testing.T) {
	repo := New()
	ctx := context.Background()

	t.Run("add and contains", func(t *testing.T) {
		cidr := "192.168.1.0/24"
		err := repo.Add(ctx, model.ListTypeWhitelist, cidr)
		if err != nil {
			t.Fatalf("add: %v", err)
		}

		// duplicate add
		err = repo.Add(ctx, model.ListTypeWhitelist, cidr)
		if err != nil {
			t.Fatalf("duplicate add: %v", err)
		}

		ok, err := repo.ContainsIP(ctx, model.ListTypeWhitelist, net.ParseIP("192.168.1.10"))
		if err != nil || !ok {
			t.Fatal("expected to contain IP")
		}

		ok, err = repo.ContainsIP(ctx, model.ListTypeWhitelist, net.ParseIP("10.0.0.1"))
		if err != nil || ok {
			t.Fatal("expected NOT to contain IP")
		}
	})

	t.Run("remove", func(t *testing.T) {
		cidr := "10.0.0.0/8"
		_ = repo.Add(ctx, model.ListTypeBlacklist, cidr)

		err := repo.Remove(ctx, model.ListTypeBlacklist, cidr)
		if err != nil {
			t.Fatalf("remove: %v", err)
		}

		ok, err := repo.ContainsIP(ctx, model.ListTypeBlacklist, net.ParseIP("10.1.2.3"))
		if err != nil || ok {
			t.Fatal("expected NOT to contain IP after removal")
		}
	})

	t.Run("invalid cidr", func(t *testing.T) {
		err := repo.Add(ctx, model.ListTypeWhitelist, "invalid")
		if err == nil {
			t.Fatal("expected error for invalid CIDR in Add")
		}

		err = repo.Remove(ctx, model.ListTypeWhitelist, "invalid")
		if err == nil {
			t.Fatal("expected error for invalid CIDR in Remove")
		}
	})
}
