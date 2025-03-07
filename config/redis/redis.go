package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"record-traffic-press/config/conf"
	"sync"
	"time"
)

var (
	client *redis.Client
	one    sync.Once
	valid  bool
)

// InitRedis Redis初始化
func InitRedis() {
	one.Do(func() {
		conf.ReadFromLocal()
		var ss = conf.GetAppConf().RedisConfig
		redisClient := redis.NewClient(&redis.Options{
			Network:      "tcp",
			Addr:         fmt.Sprintf("%s:%d", ss.Host, conf.GetAppConf().RedisConfig.Port),
			Password:     conf.GetAppConf().RedisConfig.Password,
			DB:           conf.GetAppConf().RedisConfig.DB,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			PoolTimeout:  6 * time.Second,
			PoolSize:     conf.GetAppConf().RedisConfig.MaxPoolSize,
			MinIdleConns: conf.GetAppConf().RedisConfig.MinPoolSize,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			logrus.Errorf("init redis fail username: %s, err:%v", "root", err)
		} else {
			valid = true
			logrus.Infof("init redis success username: %+v", "root")
		}
		client = redisClient
	})
}

// IsValid 是否可用
func IsValid() bool {
	return valid
}

// RedisNotValid Redis不可用
var RedisNotValid = errors.New("redis不可用")

// ValNotExist Value不存在
var ValNotExist = errors.New("key不存在")

// Get Redis-Get方法
func Get(ctx context.Context, key string) (string, error) {
	fix := conf.GetAppConf().RedisConfig.Fix
	key = fmt.Sprintf(fix+"%s", key)
	cmd := client.Get(ctx, key)
	val, err := cmd.Result()
	if errors.Is(err, redis.Nil) {
		return val, ValNotExist
	}
	return val, nil
}

// Set Redis-Set方法
func Set(ctx context.Context, key string, object interface{}, expiration time.Duration) error {
	fix := conf.GetAppConf().RedisConfig.Fix
	if str, ok := object.(string); !ok {
		b, err := json.Marshal(object)
		if err != nil {
			logrus.Errorf(fmt.Sprintf("Redis marshal Object:%s, error:%s",
				key, err.Error()))
			return err
		}
		key = fmt.Sprintf(fix+"%s", key)
		_, err = client.Set(ctx, key, string(b), expiration).Result()
		if err != nil {
			logrus.Errorf(fmt.Sprintf("REDIS Set key:%s, value:%s, error:%s",
				key, string(b), err.Error()))
			return err
		}
	} else {
		key = fmt.Sprintf(fmt.Sprintf(fix+"%s", key), key)
		_, err := client.Set(ctx, key, str, expiration).Result()
		if err != nil {
			logrus.Errorf(fmt.Sprintf("REDIS Set key:%s, value:%s, error:%s",
				key, str, err.Error()))
			return err
		}
	}
	return nil
}

// SetNX Redis-SetNX方法
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	InitRedis()
	if !IsValid() {
		return false, RedisNotValid
	}
	cmd := client.SetNX(ctx, key, value, expiration)
	return cmd.Result()
}

// Del Redis-删除方法
func Del(ctx context.Context, key string) (int64, error) {
	fix := conf.GetAppConf().RedisConfig.Fix
	key = fmt.Sprintf(fix+"%s", key)

	intCmd := client.Del(ctx, key)
	return intCmd.Result()
}

// DelKeys Redis-删除方法
func DelKeys(ctx context.Context, keys ...string) (int64, error) {
	intCmd := client.Del(ctx, keys...)
	return intCmd.Result()
}

// Lock Redis-加锁方法
func Lock(ctx context.Context, key string, expiration time.Duration) (bool, string) {
	fix := conf.GetAppConf().RedisConfig.Fix
	key = fmt.Sprintf(fix+"%s", key)

	// 生成一个新的 UUID
	newUUID := uuid.New().String()

	result, err := SetNX(ctx, key, newUUID, expiration)
	if err != nil {
		logrus.Errorf("cache.Lock: SetNX err: %v", err)
		return false, ""
	}

	return result, newUUID
}

// UnLock Redis-解锁方法
func UnLock(ctx context.Context, key string, expectValue string) bool {

	actuallyValue, err := Get(ctx, key)

	if err != nil && !errors.Is(err, ValNotExist) {
		logrus.Errorf("cache.UnLock: 1. key is [%s]. err: %v", key, err)
		return false
	}
	if actuallyValue != expectValue {
		return false
	}

	_, err = Del(ctx, key)

	if err != nil {
		logrus.Errorf("cache.UnLock: 2. key is [%s]. err: %v", key, err)
		return false
	}

	return true
}
