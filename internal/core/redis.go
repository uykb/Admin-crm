package core

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdbClient   *redis.Client
	redisOnce   sync.Once
	redisEnable bool
)

// InitRedis 初始化 Redis 客户端（支持 Upstash rediss:// 及标准 redis://）
func InitRedis() {
	redisOnce.Do(func() {
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			redisURL = os.Getenv("GA_REDIS_URL")
		}

		redisURL = strings.TrimSpace(redisURL)
		if redisURL == "" {
			log.Println("[Redis] 未配置 REDIS_URL 环境变量，已平滑降级为内置内存缓存模式")
			return
		}

		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("[Redis] 解析 REDIS_URL 失败: %v，将降级为内存模式", err)
			return
		}

		// 针对 Upstash / 云端 TLS 自动开启 TLS 校验
		if opt.TLSConfig == nil && strings.HasPrefix(redisURL, "rediss://") {
			opt.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}

		client := redis.NewClient(opt)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			log.Printf("[Redis] 连通性 Ping 测试失败: %v，将降级为内存模式", err)
			return
		}

		rdbClient = client
		redisEnable = true
		log.Printf("[Redis] ✅ Redis 客户端连接成功！已完美接入 Serverless/Upstash Redis (%s)", opt.Addr)
	})
}

// GetRedis 获取全局 Redis 客户端（未配置或连接失败时返回 nil）
func GetRedis() *redis.Client {
	if !redisEnable {
		return nil
	}
	return rdbClient
}

// IsRedisEnabled 是否开启了 Redis 缓存
func IsRedisEnabled() bool {
	return redisEnable
}
