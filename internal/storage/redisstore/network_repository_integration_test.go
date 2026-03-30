package redisstore

import (
	"antibf/internal/model"
	"context"
	"net"
	"os"
	"testing"

	redis "github.com/redis/go-redis/v9"
)

func TestNetworkRepository_AddContainsRemove(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR is not set")
	}

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: addr})
	defer client.Close()

	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush db: %v", err)
	}

	repo := NewNetworkRepository(client)
	if err := repo.Add(ctx, model.ListTypeWhitelist, "192.168.1.0/24"); err != nil {
		t.Fatalf("add network: %v", err)
	}

	ok, err := repo.ContainsIP(ctx, model.ListTypeWhitelist, net.ParseIP("192.168.1.55"))
	if err != nil {
		t.Fatalf("contains ip: %v", err)
	}
	if !ok {
		t.Fatal("expected IP to be found in whitelist")
	}

	if err := repo.Remove(ctx, model.ListTypeWhitelist, "192.168.1.0/24"); err != nil {
		t.Fatalf("remove network: %v", err)
	}

	ok, err = repo.ContainsIP(ctx, model.ListTypeWhitelist, net.ParseIP("192.168.1.55"))
	if err != nil {
		t.Fatalf("contains ip after remove: %v", err)
	}
	if ok {
		t.Fatal("expected IP to be absent after remove")
	}
}
