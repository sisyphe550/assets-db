// Package redislock Redis 分布式锁
// 用于盘点协同写入的并发控制（01-desgin.md §5.3）
package redislock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// TryLock 尝试获取分布式锁
// key: 锁键（如 fams:lock:inventory:${asset_no}）
// owner: 持有者标识（operator_id）
// ttl: 锁过期时间
// 返回 (true, nil) 表示获取成功；(false, nil) 表示被他人持有
func TryLock(ctx context.Context, rds *redis.Client, key, owner string, ttl time.Duration) (bool, error) {
	ok, err := rds.SetNX(ctx, key, owner, ttl).Result()
	if err != nil {
		return false, errx.ErrServiceUnavailable
	}
	if !ok {
		return false, errx.ErrConflict
	}
	return true, nil
}

// Unlock 安全释放锁（Lua 脚本校验 owner 再 DEL）
// 防止误删他人持有的锁
const unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`

func Unlock(ctx context.Context, rds *redis.Client, key, owner string) error {
	result, err := rds.Eval(ctx, unlockScript, []string{key}, owner).Result()
	if err != nil {
		return errx.ErrServiceUnavailable
	}
	// result == 0 表示锁已被他人持有或已过期（非错误，仅不删除）
	_ = result
	return nil
}
