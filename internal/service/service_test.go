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
	_, err := svc.Check(context.Background(), model.AuthAttempt{
		Login: "alice", Password: "x", IP: "invalid",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
