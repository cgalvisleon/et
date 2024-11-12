package cache

import (
	"context"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/redis/go-redis/v9"
)

/**
* SetCtx
* @params ctx context.Context
* @params key string
* @params val string
* @params second time.Duration
* @return error
**/
func SetCtx(ctx context.Context, key, val string, second time.Duration) error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.db.Set(ctx, key, val, second).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* GetCtx
* @params ctx context.Context
* @params key string
* @params def string
* @return string, error
**/
func GetCtx(ctx context.Context, key, def string) (string, error) {
	if conn == nil {
		return def, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	result, err := conn.db.Get(ctx, key).Result()
	switch {
	case err == redis.Nil:
		return def, nil
	case err != nil:
		return def, err
	default:
		return result, nil
	}
}

/**
* DeleteCtx
* @params ctx context.Context
* @params key string
* @return int64, error
**/
func DeleteCtx(ctx context.Context, key string) (int64, error) {
	if conn == nil {
		return 0, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	intCmd := conn.db.Del(ctx, key)

	return intCmd.Val(), intCmd.Err()
}

/**
* HSetCtx
* @params ctx context.Context
* @params key string
* @params val map[string]string
* @return error
**/
func HSetCtx(ctx context.Context, key string, val map[string]string) error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.db.HSet(ctx, key, val).Err()
	if err != nil {
		return err
	}

	return nil
}

/**
* HGetCtx
* @params ctx context.Context
* @params key string
* @return map[string]string, error
**/
func HGetCtx(ctx context.Context, key string) (map[string]string, error) {
	if conn == nil {
		return map[string]string{}, logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	result := conn.db.HGetAll(ctx, key).Val()

	return result, nil
}

/**
* HDeleteCtx
* @params ctx context.Context
* @params key string
* @params atr string
* @return error
**/
func HDeleteCtx(ctx context.Context, key, atr string) error {
	if conn == nil {
		return logs.Log(msg.ERR_NOT_CACHE_SERVICE)
	}

	err := conn.db.Do(ctx, "HDEL", key, atr).Err()
	if err != nil {
		return err
	}

	return nil
}
