package model

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Absolute personal-quota replacements use a monotonic cache version. The
// pending fence is published before the database write and outlives every
// user-cache hash. A delayed cache fill carrying an older version is rejected
// even when it arrives after the adjustment's final cache deletion.

var ErrUserQuotaCachePending = errors.New("user quota cache update is pending")

var ErrUserQuotaVersionConflict = errors.New("user quota version update conflicted")

func getUserQuotaFenceKey(userId int) string {
	return fmt.Sprintf("quota:user:fence:%d", userId)
}

func getUserQuotaVersionKey(userId int) string {
	return fmt.Sprintf("quota:user:version:%d", userId)
}

func userQuotaFenceTTLSeconds() int {
	cacheTTL := userCacheTTLSeconds()
	extra := cacheTTL
	if extra < 60 {
		extra = 60
	}
	return cacheTTL + extra
}

func getUserQuotaVersionFloor(userId int) (int64, error) {
	if !common.RedisEnabled {
		return 0, nil
	}
	values, err := common.RDB.MGet(
		context.Background(),
		getUserQuotaFenceKey(userId),
		getUserQuotaVersionKey(userId),
	).Result()
	if err != nil {
		return 0, err
	}
	var floor int64
	for _, value := range values {
		if value == nil {
			continue
		}
		parsed, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
		if err != nil {
			return 0, err
		}
		if parsed > floor {
			floor = parsed
		}
	}
	return floor, nil
}

func setUserQuotaVersionFence(userId int, quotaVersion int64) error {
	if !common.RedisEnabled {
		return nil
	}
	if userId <= 0 || quotaVersion <= 0 {
		return fmt.Errorf("invalid user quota fence")
	}
	const script = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local incoming = tonumber(ARGV[1])
if current < incoming then
  redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
elseif current == incoming then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
elseif redis.call('TTL', KEYS[1]) < 0 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
return 1`
	return common.RDB.Eval(
		context.Background(),
		script,
		[]string{getUserQuotaFenceKey(userId)},
		quotaVersion,
		userQuotaFenceTTLSeconds(),
	).Err()
}

func publishCommittedUserQuotaVersion(userId int, quotaVersion int64) error {
	if !common.RedisEnabled {
		return nil
	}
	if userId <= 0 || quotaVersion <= 0 {
		return fmt.Errorf("invalid committed user quota version")
	}
	const script = `
local incoming = tonumber(ARGV[1])
local committed = tonumber(redis.call('GET', KEYS[1]) or '0')
local pending = tonumber(redis.call('GET', KEYS[2]) or '0')
local cached = tonumber(redis.call('HGET', KEYS[3], 'QuotaVersion') or '0')
if committed < incoming then
  redis.call('SET', KEYS[1], ARGV[1])
end
if pending > 0 and pending <= incoming then
  redis.call('DEL', KEYS[2])
end
if cached < incoming then
  redis.call('DEL', KEYS[3])
end
return 1`
	return common.RDB.Eval(
		context.Background(),
		script,
		[]string{
			getUserQuotaVersionKey(userId),
			getUserQuotaFenceKey(userId),
			getUserCacheKey(userId),
		},
		quotaVersion,
	).Err()
}

func updateUserQuotaCacheAtVersion(userId int, quota int, quotaVersion int64) error {
	if !common.RedisEnabled {
		return nil
	}
	if userId <= 0 || quota < 0 || quotaVersion <= 0 {
		return fmt.Errorf("invalid versioned user quota cache update")
	}
	const script = `
local incoming = tonumber(ARGV[1])
local pending = tonumber(redis.call('GET', KEYS[2]) or '0')
local committed = tonumber(redis.call('GET', KEYS[3]) or '0')
local current = tonumber(redis.call('HGET', KEYS[1], 'QuotaVersion') or '0')
if pending > incoming or committed > incoming or current > incoming then
  return 0
end
if redis.call('EXISTS', KEYS[1]) == 0 then
  return 1
end
if committed < incoming then
  redis.call('SET', KEYS[3], ARGV[1])
end
if pending > 0 and pending <= incoming then
  redis.call('DEL', KEYS[2])
end
redis.call('HSET', KEYS[1], 'Quota', ARGV[2], 'QuotaVersion', ARGV[1])
redis.call('EXPIRE', KEYS[1], ARGV[3])
return 1`
	result, err := common.RDB.Eval(
		context.Background(),
		script,
		[]string{
			getUserCacheKey(userId),
			getUserQuotaFenceKey(userId),
			getUserQuotaVersionKey(userId),
		},
		quotaVersion,
		quota,
		userCacheTTLSeconds(),
	).Int()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrUserQuotaCachePending
	}
	return nil
}

func loadUserQuotaVersionTx(tx *gorm.DB, userId int, lock bool) (int64, error) {
	if tx == nil || userId <= 0 {
		return 0, fmt.Errorf("invalid user quota version lookup")
	}
	query := tx
	if lock {
		query = lockForUpdate(query)
	}
	var user User
	if err := query.Select("id", "quota_version").Where("id = ?", userId).First(&user).Error; err != nil {
		return 0, err
	}
	if user.QuotaVersion <= 0 {
		return 0, fmt.Errorf("invalid user quota version")
	}
	return user.QuotaVersion, nil
}

func incrementUserQuotaVersionWithTx(tx *gorm.DB, userId int) (int64, error) {
	current, err := loadUserQuotaVersionTx(tx, userId, true)
	if err != nil {
		return 0, err
	}
	if current == int64(^uint64(0)>>1) {
		return 0, ErrUserQuotaVersionConflict
	}
	next := current + 1
	if err := setUserQuotaVersionFence(userId, next); err != nil {
		return 0, err
	}
	result := tx.Model(&User{}).
		Where("id = ? AND quota_version = ?", userId, current).
		Update("quota_version", next)
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected != 1 {
		return 0, ErrUserQuotaVersionConflict
	}
	return next, nil
}

// InitializeUserQuotaVersions repairs rows created before the quota cache
// version column existed. It is idempotent across all supported databases.
func InitializeUserQuotaVersions() error {
	return DB.Model(&User{}).
		Where("quota_version IS NULL OR quota_version < ?", 1).
		Update("quota_version", 1).Error
}
