package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"apeadmin-gin/internal/pkg/response"
)

// tokenBucket 令牌桶
type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

var (
	tbMu      sync.Mutex
	tbBuckets = make(map[string]*tokenBucket)
)

// StartBucketCleanup 定期清理长期未使用的令牌桶，防止内存无限增长
func StartBucketCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			tbMu.Lock()
			for k, b := range tbBuckets {
				// 超过 2 个窗口未活动的桶视为过期
				if now.Sub(b.lastTime) > 2*time.Minute {
					delete(tbBuckets, k)
				}
			}
			tbMu.Unlock()
		}
	}()
}

// RateLimit API 限流中间件（令牌桶，按 IP）
// name: 限流标识；rpm: 每分钟请求数；window: 时间窗口
func RateLimit(name string, rpm int, window time.Duration) gin.HandlerFunc {
	refillRate := float64(rpm) / window.Seconds()
	burst := rpm
	if burst < 1 {
		burst = 1
	}
	return func(c *gin.Context) {
		key := name + ":" + c.ClientIP()
		tbMu.Lock()
		bucket, exists := tbBuckets[key]
		if !exists {
			bucket = &tokenBucket{tokens: float64(burst), lastTime: time.Now()}
			tbBuckets[key] = bucket
		}
		// 补充令牌
		now := time.Now()
		elapsed := now.Sub(bucket.lastTime).Seconds()
		bucket.tokens += elapsed * refillRate
		if bucket.tokens > float64(burst) {
			bucket.tokens = float64(burst)
		}
		bucket.lastTime = now
		if bucket.tokens < 1 {
			tbMu.Unlock()
			retryAfter := int(1 / refillRate)
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.Error(429, "请求过于频繁，请稍后再试"))
			return
		}
		bucket.tokens--
		tbMu.Unlock()
		c.Next()
	}
}
