package redisstore

import (
	"antibf/internal/model"
	"context"
	"fmt"
	"net"

	"github.com/redis/go-redis/v9"
)

type NetworkRepository struct {
	client *redis.Client
}

func NewNetworkRepository(client *redis.Client) *NetworkRepository {
	return &NetworkRepository{client: client}
}

func (r *NetworkRepository) Add(ctx context.Context, listType model.NetworkListType, cidr string) error {
	if err := r.client.SAdd(ctx, listKey(listType), cidr).Err(); err != nil {
		return fmt.Errorf("add network to %s: %w", listType, err)
	}
	return nil
}

func (r *NetworkRepository) Remove(ctx context.Context, listType model.NetworkListType, cidr string) error {
	if err := r.client.SRem(ctx, listKey(listType), cidr).Err(); err != nil {
		return fmt.Errorf("remove network from %s: %w", listType, err)
	}
	return nil
}

func (r *NetworkRepository) ContainsIP(ctx context.Context, listType model.NetworkListType, ip net.IP) (bool, error) {
	cidrs, err := r.client.SMembers(ctx, listKey(listType)).Result()
	if err != nil {
		return false, fmt.Errorf("load networks from %s: %w", listType, err)
	}

	for _, cidr := range cidrs {
		_, network, errParse := net.ParseCIDR(cidr)
		if errParse != nil {
			continue
		}
		if network.Contains(ip) {
			return true, nil
		}
	}

	return false, nil
}

func listKey(listType model.NetworkListType) string {
	return "antibf:" + string(listType)
}
