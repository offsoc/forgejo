// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package queue

import (
	"context"
	"sync"
	"time"

	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/nosql"

	"github.com/redis/go-redis/v9"
)

type baseRedis struct {
	client   nosql.RedisClient
	isUnique bool
	cfg      *BaseConfig
	prefix   string

	mu sync.Mutex // the old implementation is not thread-safe, the queue operation and set operation should be protected together
}

var _ baseQueue = (*baseRedis)(nil)

func newBaseRedisGeneric(cfg *BaseConfig, unique bool, client nosql.RedisClient) (baseQueue, error) {
	if client == nil {
		client = nosql.GetManager().GetRedisClient(cfg.ConnStr)
	}

	prefix := ""
	uri := nosql.ToRedisURI(cfg.ConnStr)

	for key, value := range uri.Query() {
		switch key {
		case "prefix":
			if len(value) > 0 {
				prefix = value[0]

				// As we are not checking any other values, if we found this one, we can
				// exit from the loop.
				// If a new key check is required, remove this break.
				break
			}
		}
	}

	var err error
	for i := 0; i < 10; i++ {
		err = client.Ping(graceful.GetManager().ShutdownContext()).Err()
		if err == nil {
			break
		}
		log.Warn("Redis is not ready, waiting for 1 second to retry: %v", err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	return &baseRedis{cfg: cfg, client: client, isUnique: unique, prefix: prefix}, nil
}

func newBaseRedisSimple(cfg *BaseConfig) (baseQueue, error) {
	return newBaseRedisGeneric(cfg, false, nil)
}

func newBaseRedisUnique(cfg *BaseConfig) (baseQueue, error) {
	return newBaseRedisGeneric(cfg, true, nil)
}

func (q *baseRedis) prefixedName(name string) string {
	return q.prefix + name
}

func (q *baseRedis) PushItem(ctx context.Context, data []byte) error {
	return backoffErr(ctx, backoffBegin, backoffUpper, time.After(pushBlockTime), func() (retry bool, err error) {
		q.mu.Lock()
		defer q.mu.Unlock()

		cnt, err := q.client.LLen(ctx, q.prefixedName(q.cfg.QueueFullName)).Result()
		if err != nil {
			return false, err
		}
		if int(cnt) >= q.cfg.Length {
			return true, nil
		}

		if q.isUnique {
			added, err := q.client.SAdd(ctx, q.prefixedName(q.cfg.SetFullName), data).Result()
			if err != nil {
				return false, err
			}
			if added == 0 {
				return false, ErrAlreadyInQueue
			}
		}
		return false, q.client.RPush(ctx, q.prefixedName(q.cfg.QueueFullName), data).Err()
	})
}

func (q *baseRedis) PopItem(ctx context.Context) ([]byte, error) {
	return backoffRetErr(ctx, backoffBegin, backoffUpper, infiniteTimerC, func() (retry bool, data []byte, err error) {
		q.mu.Lock()
		defer q.mu.Unlock()

		data, err = q.client.LPop(ctx, q.prefixedName(q.cfg.QueueFullName)).Bytes()
		if err == redis.Nil {
			return true, nil, nil
		}
		if err != nil {
			return true, nil, nil
		}
		if q.isUnique {
			// the data has been popped, even if there is any error we can't do anything
			_ = q.client.SRem(ctx, q.prefixedName(q.cfg.SetFullName), data).Err()
		}
		return false, data, err
	})
}

func (q *baseRedis) HasItem(ctx context.Context, data []byte) (bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.isUnique {
		return false, nil
	}
	return q.client.SIsMember(ctx, q.prefixedName(q.cfg.SetFullName), data).Result()
}

func (q *baseRedis) Len(ctx context.Context) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	cnt, err := q.client.LLen(ctx, q.prefixedName(q.cfg.QueueFullName)).Result()
	return int(cnt), err
}

func (q *baseRedis) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.client.Close()
}

func (q *baseRedis) RemoveAll(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	c1 := q.client.Del(ctx, q.prefixedName(q.cfg.QueueFullName))
	// the "set" must be cleared after the "list" because there is no transaction.
	// it's better to have duplicate items than losing items.
	c2 := q.client.Del(ctx, q.prefixedName(q.cfg.SetFullName))
	if c1.Err() != nil {
		return c1.Err()
	}
	if c2.Err() != nil {
		return c2.Err()
	}
	return nil // actually, checking errors doesn't make sense here because the state could be out-of-sync
}
