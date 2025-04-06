// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package nosql

import (
	"context"
	"strconv"
	"sync"
	"time"

	"forgejo.org/modules/process"

	"github.com/redis/go-redis/v9"
	"github.com/syndtr/goleveldb/leveldb"
)

var manager *Manager

// Manager is the nosql connection manager
type Manager struct {
	ctx      context.Context
	finished context.CancelFunc
	mutex    sync.Mutex

	RedisConnections   map[string]*redisClientHolder
	LevelDBConnections map[string]*levelDBHolder
}

// RedisClient is a subset of redis.UniversalClient, it exposes less methods
// to avoid generating machine code for unused methods. New method definitions
// should be copied from the definitions in the Redis library github.com/redis/go-redis.
type RedisClient interface {
	// redis.GenericCmdable
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd

	// redis.ListCmdable
	RPush(ctx context.Context, key string, values ...any) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
	LLen(ctx context.Context, key string) *redis.IntCmd

	// redis.StringCmdable
	Decr(ctx context.Context, key string) *redis.IntCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd

	// redis.HashCmdable
	HSet(ctx context.Context, key string, values ...any) *redis.IntCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HKeys(ctx context.Context, key string) *redis.StringSliceCmd

	// redis.SetCmdable
	SAdd(ctx context.Context, key string, members ...any) *redis.IntCmd
	SRem(ctx context.Context, key string, members ...any) *redis.IntCmd
	SIsMember(ctx context.Context, key string, member any) *redis.BoolCmd

	// redis.Cmdable
	DBSize(ctx context.Context) *redis.IntCmd
	FlushDB(ctx context.Context) *redis.StatusCmd
	Ping(ctx context.Context) *redis.StatusCmd

	// redis.UniversalClient
	Close() error
}

type redisClientHolder struct {
	RedisClient
	name  []string
	count int64
}

func (r *redisClientHolder) Close() error {
	return manager.CloseRedisClient(r.name[0])
}

type levelDBHolder struct {
	name  []string
	count int64
	db    *leveldb.DB
}

func init() {
	_ = GetManager()
}

// GetManager returns a Manager and initializes one as singleton is there's none yet
func GetManager() *Manager {
	if manager == nil {
		ctx, _, finished := process.GetManager().AddTypedContext(context.Background(), "Service: NoSQL", process.SystemProcessType, false)
		manager = &Manager{
			ctx:                ctx,
			finished:           finished,
			RedisConnections:   make(map[string]*redisClientHolder),
			LevelDBConnections: make(map[string]*levelDBHolder),
		}
	}
	return manager
}

func valToTimeDuration(vs []string) (result time.Duration) {
	var err error
	for _, v := range vs {
		result, err = time.ParseDuration(v)
		if err != nil {
			var val int
			val, err = strconv.Atoi(v)
			result = time.Duration(val)
		}
		if err == nil {
			return result
		}
	}
	return result
}
