package service

import (
	"antibf/internal/model"
	"context"
	"net"
	"testing"
)

type repoStub struct {
	whitelisted bool
	blacklisted bool
}

func (r repoStub) Add(context.Context, model.NetworkListType, string) error    { return nil }
func (r repoStub) Remove(context.Context, model.NetworkListType, string) error { return nil }
func (r repoStub) ContainsIP(_ context.Context, listType model.NetworkListType, _ net.IP) (bool, error) {
	if listType == model.ListTypeWhitelist {
		return r.whitelisted, nil
	}
	if listType == model.ListTypeBlacklist {
		return r.blacklisted, nil
	}
	return false, nil
}

type limiterStub struct {
	allow bool
}

func (l limiterStub) Allow(context.Context, string, string, string) (bool, error) {
	return l.allow, nil
}
func (l limiterStub) Reset(context.Context, string, string) error { return nil }

func TestService_WhitelistWins(t *testing.T) {
	svc := New(repoStub{whitelisted: true}, limiterStub{allow: false})
	ok, err := svc.Check(context.Background(), model.AuthAttempt{
		Login: "alice", Password: "x", IP: "192.168.1.10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected whitelisted ip to be allowed")
	}
}

func TestService_BlacklistWins(t *testing.T) {
	svc := New(repoStub{blacklisted: true}, limiterStub{allow: true})
	ok, err := svc.Check(context.Background(), model.AuthAttempt{
		Login: "alice", Password: "x", IP: "10.1.2.3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected blacklisted ip to be blocked")
	}
}

func TestService_ValidationError(t *testing.T) {
	svc := New(repoStub{}, limiterStub{allow: true})
	ctx := context.Background()

	t.Run("invalid ip", func(t *testing.T) {
		_, err := svc.Check(ctx, model.AuthAttempt{
			Login: "alice", Password: "x", IP: "invalid",
		})
		if err == nil {
			t.Fatal("expected validation error for invalid ip")
		}
	})

	t.Run("empty login", func(t *testing.T) {
		_, err := svc.Check(ctx, model.AuthAttempt{
			Login: "", Password: "x", IP: "1.1.1.1",
		})
		if err == nil {
			t.Fatal("expected validation error for empty login")
		}
	})

	t.Run("empty password", func(t *testing.T) {
		_, err := svc.Check(ctx, model.AuthAttempt{
			Login: "alice", Password: "", IP: "1.1.1.1",
		})
		if err == nil {
			t.Fatal("expected validation error for empty password")
		}
	})
}

func TestService_Reset(t *testing.T) {
	svc := New(repoStub{}, limiterStub{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		err := svc.Reset(ctx, model.ResetRequest{Login: "alice", IP: "1.1.1.1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty login", func(t *testing.T) {
		err := svc.Reset(ctx, model.ResetRequest{Login: " ", IP: "1.1.1.1"})
		if err == nil {
			t.Fatal("expected error for empty login")
		}
	})

	t.Run("invalid ip", func(t *testing.T) {
		err := svc.Reset(ctx, model.ResetRequest{Login: "alice", IP: "invalid"})
		if err == nil {
			t.Fatal("expected error for invalid ip")
		}
	})
}

func TestService_NetworkManagement(t *testing.T) {
	svc := New(repoStub{}, limiterStub{})
	ctx := context.Background()

	t.Run("add success", func(t *testing.T) {
		err := svc.AddNetwork(ctx, model.ListTypeWhitelist, "192.168.1.0/24")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("add invalid cidr", func(t *testing.T) {
		err := svc.AddNetwork(ctx, model.ListTypeWhitelist, "invalid")
		if err == nil {
			t.Fatal("expected error for invalid CIDR")
		}
	})

	t.Run("add ipv6 cidr", func(t *testing.T) {
		err := svc.AddNetwork(ctx, model.ListTypeWhitelist, "2001:db8::/32")
		if err == nil {
			t.Fatal("expected error for IPv6 CIDR")
		}
	})

	t.Run("remove success", func(t *testing.T) {
		err := svc.RemoveNetwork(ctx, model.ListTypeWhitelist, "192.168.1.0/24")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("remove invalid cidr", func(t *testing.T) {
		err := svc.RemoveNetwork(ctx, model.ListTypeWhitelist, "invalid")
		if err == nil {
			t.Fatal("expected error for invalid CIDR")
		}
	})
}
