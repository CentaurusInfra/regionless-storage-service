package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/constants"
)

var (
	pools    map[string]*redis.Pool
	initOnce sync.Once
)

func InitStorageInstancePool(stores []config.KVStore) {
	pools = make(map[string]*redis.Pool)
	for _, conf := range stores {
		url, pool := initPool(conf.Host, conf.Port)
		pools[url] = pool
	}
}

func initPool(host string, port int) (string, *redis.Pool) {
	url := fmt.Sprintf("%s:%d", host, port)
	if pools[url] != nil {
		return url, pools[url]
	}
	pool := &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", url)
			for retries := 0; err != nil && retries < constants.RedisRetryCount; retries++ {
				log.Printf("ERROR: failed to init the redis %s connection with error %v after %d times\n", url, err, retries+1)
				time.Sleep(constants.RedisRetryInterval << retries)
				if conn, err = redis.Dial("tcp", url); err == nil {
					if _, err = conn.Do("PING"); err != nil {
						log.Printf("ERROR: failed to ping redis %s: %v after %d retries\n", url, err, retries+1)
					}
				}
			}
			return conn, err
		},
	}
	fmt.Printf("The url is %s and the pool is %v\n", url, pool)
	return url, pool
}

type RedisDatabase struct {
	client  *redis.Pool
	latency time.Duration
}

func createRedisDatabase(databaseUrl string) (Database, error) {
	initOnce.Do(func() {
		InitStorageInstancePool(config.RKVConfig.Stores)
	})
	return &RedisDatabase{client: pools[databaseUrl], latency: 0}, nil
}

func (rd *RedisDatabase) Put(key, value string) (string, error) {
	conn, err := rd.client.Dial()
	if err != nil {
		return "", err
	}
	defer conn.Close()
	ret, err := conn.Do("Set", key, value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", ret), err
}

func (rd *RedisDatabase) Get(key string) (string, error) {
	conn, err := rd.client.Dial()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if resp, err := conn.Do("Get", key); err == nil {
		return fmt.Sprintf("%s", resp), nil
	} else {
		return "", err
	}
}

func (rd *RedisDatabase) Delete(key string) error {
	conn, err := rd.client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Do("Del", key); err == nil {
		return nil
	} else {
		return err
	}
}

func (rd *RedisDatabase) Close() error {
	return rd.client.Close()
}

func (rd *RedisDatabase) Latency() time.Duration {
	return rd.latency
}

func (rd *RedisDatabase) SetLatency(latency time.Duration) {
	rd.latency = latency
}
