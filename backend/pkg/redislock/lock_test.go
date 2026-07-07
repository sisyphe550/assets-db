//go:build integration
// +build integration

package redislock

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	return rdb
}

func TestTryLock(t *testing.T) {
	rdb := newTestClient(t)
	defer rdb.Close()
	ctx := context.Background()

	key := "fams:test:lock:001"
	owner := "operator_10003"

	// First lock
	ok, err := TryLock(ctx, rdb, key, owner, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Skip("First lock failed (conflict)")
	}

	// Second lock by another owner
	ok2, _ := TryLock(ctx, rdb, key, "operator_10004", 10*time.Second)
	if ok2 {
		t.Error("second lock should have failed")
	}

	// Safe unlock
	if err := Unlock(ctx, rdb, key, owner); err != nil {
		t.Fatal(err)
	}

	// Unlock by wrong owner
	// Re-lock first
	TryLock(ctx, rdb, key, owner, 10*time.Second)
	if err := Unlock(ctx, rdb, key, "someone_else"); err != nil {
		t.Fatal(err)
	}
	// Lock should still exist
	val, _ := rdb.Get(ctx, key).Result()
	if val != owner {
		t.Errorf("lock should not be released by wrong owner, got %q", val)
	}

	// Cleanup
	rdb.Del(ctx, key)
}

func TestUnlockLuaScript(t *testing.T) {
	rdb := newTestClient(t)
	defer rdb.Close()
	ctx := context.Background()

	key := "fams:test:lock:002"
	owner := "test_user"

	// Set manually
	rdb.Set(ctx, key, owner, 30*time.Second)

	// Wrong owner can't delete
	if err := Unlock(ctx, rdb, key, "wrong_user"); err != nil {
		t.Fatal(err)
	}
	exists, _ := rdb.Exists(ctx, key).Result()
	if exists == 0 {
		t.Error("wrong owner should not be able to delete the lock")
	}

	// Correct owner can delete
	if err := Unlock(ctx, rdb, key, owner); err != nil {
		t.Fatal(err)
	}
	exists, _ = rdb.Exists(ctx, key).Result()
	if exists != 0 {
		t.Error("correct owner should have deleted the lock")
	}
}
