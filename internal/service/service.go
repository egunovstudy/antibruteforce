package service

import (
	"context"
	"fmt"
	"net"
	"strings"

	"antibf/internal/model"
	"antibf/internal/ratelimit"
)

type NetworkListRepository interface {
	Add(ctx context.Context, listType model.NetworkListType, cidr string) error
	Remove(ctx context.Context, listType model.NetworkListType, cidr string) error
	ContainsIP(ctx context.Context, listType model.NetworkListType, ip net.IP) (bool, error)
}

type AntiBruteforceService struct {
	repo    NetworkListRepository
	limiter ratelimit.Limiter
}

func New(repo NetworkListRepository, limiter ratelimit.Limiter) *AntiBruteforceService {
	return &AntiBruteforceService{repo: repo, limiter: limiter}
}

func (s *AntiBruteforceService) Check(ctx context.Context, attempt model.AuthAttempt) (bool, error) {
	ip, err := validateAttempt(attempt)
	if err != nil {
		return false, err
	}

	whitelisted, err := s.repo.ContainsIP(ctx, model.ListTypeWhitelist, ip)
	if err != nil {
		return false, fmt.Errorf("check whitelist: %w", err)
	}
	if whitelisted {
		return true, nil
	}

	blacklisted, err := s.repo.ContainsIP(ctx, model.ListTypeBlacklist, ip)
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}
	if blacklisted {
		return false, nil
	}

	return s.limiter.Allow(ctx, attempt.Login, attempt.Password, attempt.IP)
}

func (s *AntiBruteforceService) Reset(ctx context.Context, req model.ResetRequest) error {
	if strings.TrimSpace(req.Login) == "" {
		return fmt.Errorf("login is required")
	}
	if ip := net.ParseIP(req.IP); ip == nil || ip.To4() == nil {
		return fmt.Errorf("invalid IPv4 address")
	}
	return s.limiter.Reset(ctx, req.Login, req.IP)
}

func (s *AntiBruteforceService) AddNetwork(ctx context.Context, listType model.NetworkListType, cidr string) error {
	if err := validateCIDR(cidr); err != nil {
		return err
	}
	return s.repo.Add(ctx, listType, cidr)
}

func (s *AntiBruteforceService) RemoveNetwork(ctx context.Context, listType model.NetworkListType, cidr string) error {
	if err := validateCIDR(cidr); err != nil {
		return err
	}
	return s.repo.Remove(ctx, listType, cidr)
}

func validateAttempt(attempt model.AuthAttempt) (net.IP, error) {
	if strings.TrimSpace(attempt.Login) == "" {
		return nil, fmt.Errorf("login is required")
	}
	if strings.TrimSpace(attempt.Password) == "" {
		return nil, fmt.Errorf("password is required")
	}
	ip := net.ParseIP(attempt.IP)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("invalid IPv4 address")
	}
	return ip, nil
}

func validateCIDR(cidr string) error {
	ip, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return fmt.Errorf("invalid CIDR: %w", err)
	}
	if ip.To4() == nil || ipNet.IP.To4() == nil {
		return fmt.Errorf("only IPv4 is supported")
	}
	return nil
}
