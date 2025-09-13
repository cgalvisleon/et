package cache

import (
	"context"
	"errors"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/redis/go-redis/v9"
)

/**
* SetCtx
* @params ctx context.Context, key string, val string, expiration time.Duration
* @return string
**/
func SetCtx(ctx context.Context, key, val string, expiration time.Duration) string {
	if conn == nil {
		return val
	}

	err := conn.Set(ctx, key, val, expiration).Err()
	if err != nil {
		return val
	}

	return val
}

/**
* ExpireCtx
* @params ctx context.Context, key string, second time.Duration
* @return error
**/
func ExpireCtx(ctx context.Context, key string, second time.Duration) error {
	if conn == nil {
		return errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	return conn.Expire(ctx, key, second).Err()
}

/**
* IncrCtx
* @params ctx context.Context, key string, expiration time.Duration
* @return int64
**/
func IncrCtx(ctx context.Context, key string, expiration time.Duration) int64 {
	if conn == nil {
		return 0
	}

	result, err := conn.Incr(ctx, key).Result()
	if err != nil {
		return 0
	}

	if result == 1 {
		conn.Expire(ctx, key, expiration)
	}

	return result
}

/**
* DecrCtx
* @params ctx context.Context, key string
* @return int64
**/
func DecrCtx(ctx context.Context, key string) int64 {
	if conn == nil {
		return 0
	}

	result, err := conn.Decr(ctx, key).Result()
	if err != nil {
		return 0
	}

	return result
}

/**
* GetCtx
* @params ctx context.Context, key string, def string
* @return string, error
**/
func GetCtx(ctx context.Context, key, def string) (string, error) {
	if conn == nil {
		return def, errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	result, err := conn.Get(ctx, key).Result()
	if err == redis.Nil {
		return def, nil
	} else if err != nil {
		return def, err
	}

	return result, nil
}

/**
* ExistsCtx
* @params ctx context.Context, key string
* @return bool
**/
func ExistsCtx(ctx context.Context, key string) bool {
	if conn == nil {
		logs.Alertm(msg.ERR_NOT_CACHE_SERVICE)
		return false
	}

	result, err := conn.Exists(ctx, key).Result()
	if err != nil {
		logs.Alertm(msg.ERR_NOT_CACHE_SERVICE)
		return false
	}

	return result == 1
}

/**
* DeleteCtx
* @params ctx context.Context, key string
* @return int64, error
**/
func DeleteCtx(ctx context.Context, key string) (int64, error) {
	if conn == nil {
		return 0, errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	intCmd := conn.Del(ctx, key)

	return intCmd.Val(), intCmd.Err()
}

/**
* LPushCtx
* @params ctx context.Context, key string, val string
* @return error
**/
func LPushCtx(ctx context.Context, key string, val string) error {
	if conn == nil {
		return errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.RPush(ctx, key, val).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* LPushCtx
* @params ctx context.Context, key string, val string
* @return error
**/
func LRemCtx(ctx context.Context, key string, val string) error {
	if conn == nil {
		return errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.LRem(ctx, key, 1, val).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* LRangeCtx
* @params ctx context.Context, key string, start int64, stop int64
* @return []string, error
**/
func LRangeCtx(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	if conn == nil {
		return []string{}, errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	result, err := conn.LRange(ctx, key, start, stop).Result()

	return result, err
}

/**
* LTrimCtx
* @params ctx context.Context, key string, start int64, stop int64
* @return error
**/
func LTrimCtx(ctx context.Context, key string, start int64, stop int64) error {
	if conn == nil {
		return errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.LTrim(ctx, key, start, stop).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* HSetCtx
* @params ctx context.Context, key string, val map[string]string
* @return error
**/
func HSetCtx(ctx context.Context, key string, val map[string]string) error {
	if conn == nil {
		return errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.HSet(ctx, key, val).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* HGetCtx
* @params ctx context.Context, key string
* @return map[string]string, error
**/
func HGetCtx(ctx context.Context, key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	result := conn.HGetAll(ctx, key).Val()

	return result, nil
}

/**
* HDeleteCtx
* @params ctx context.Context, key string, atr string
* @return error
**/
func HDeleteCtx(ctx context.Context, key, atr string) error {
	if conn == nil {
		return errors.New(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.Do(ctx, "HDEL", key, atr).Err()
	if err != nil {
		return err
	}

	return nil
}
