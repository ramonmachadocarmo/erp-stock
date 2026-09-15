package redisadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"erp/pkg/redisx"
	"erp/services/stock-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) SetBalance(ctx context.Context, b domain.Balance) error {
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key(b.ProductID, b.WarehouseID), raw, 10*time.Minute).Err()
}

func (c *Cache) GetBalance(ctx context.Context, productID, warehouseID string) (domain.Balance, bool, error) {
	raw, err := c.client.Get(ctx, key(productID, warehouseID)).Bytes()
	if err == redis.Nil {
		return domain.Balance{}, false, nil
	}
	if err != nil {
		return domain.Balance{}, false, err
	}
	var b domain.Balance
	if err := json.Unmarshal(raw, &b); err != nil {
		return domain.Balance{}, false, err
	}
	return b, true, nil
}

type Locker struct {
	client *redis.Client
}

func NewLocker(client *redis.Client) *Locker {
	return &Locker{client: client}
}

func (l *Locker) Lock(ctx context.Context, k string) (func(context.Context), error) {
	return redisx.Lock(ctx, l.client, k, 8*time.Second)
}

func key(productID, warehouseID string) string {
	return fmt.Sprintf("balance:%s:%s", productID, warehouseID)
}
