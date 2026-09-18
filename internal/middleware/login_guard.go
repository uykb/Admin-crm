package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/pkg/response"
)

// loginGuardEntry 登录失败记录
type loginGuardEntry struct {
	attempts     int
	firstAttempt time.Time
	lockedUntil  time.Time
}

var (
	lgMu      sync.Mutex
	lgRecords = make(map[string]*loginGuardEntry) // key = ip（按 IP 维度防爆破）
)

func init() {
	// 定期清理过期的登录失败记录，防止内存无限增长
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			lgMu.Lock()
			for k, e := range lgRecords {
				if e.lockedUntil.IsZero() {
			// 无锁定：窗口超时后清除
					if now.Sub(e.firstAttempt) > 30*time.Minute {
						delete(lgRecords, k)
					}
				} else if now.After(e.lockedUntil) {
					// 锁定已过期：保留一段观察期后清除
					if now.Sub(e.lockedUntil) > 30*time.Minute {
						delete(lgRecords, k)
					}
				}
			}
			lgMu.Unlock()
		}
	}()
}

// LoginGuard 登录防爆破中间件（按 IP 维度计数锁定，username 维度作为二次约束）
func LoginGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := core.GetConfig().Security.LoginGuard
		if cfg.MaxAttempts <= 0 || cfg.WindowMinutes <= 0 {
			c.Next()
			return
		}
		ip := c.ClientIP()

		// 读取原始 body 并完整恢复，避免影响后续 handler 解析（限制 1MB 防内存耗尽）
		var username string
		bodyBytes, _ := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// JSON 请求：从 body 解析 username；表单请求：走 PostForm
		if ct := c.ContentType(); ct != "" && !strings.HasPrefix(ct, "application/x-www-form-urlencoded") && !strings.HasPrefix(ct, "multipart/form-data") {
			var body map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &body); err == nil {
				if u, ok := body["username"].(string); ok {
					username = u
				}
			}
		} else {
			username = c.PostForm("username")
		}

		// 滑动窗口重置：超过窗口时间则清零计数
		now := time.Now()
		lgMu.Lock()
		entry, exists := lgRecords[ip]
		if exists && now.Sub(entry.firstAttempt) > time.Duration(cfg.WindowMinutes)*time.Minute {
			entry.attempts = 0
			entry.firstAttempt = now
		}
		if exists && !entry.lockedUntil.IsZero() && now.After(entry.lockedUntil) {
			entry.lockedUntil = time.Time{}
			entry.attempts = 0
			entry.firstAttempt = now
		}
		if exists && now.Before(entry.lockedUntil) {
			lgMu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.Error(429, "登录失败次数过多，请稍后再试"))
			return
		}
		lgMu.Unlock()
		c.Next()

		// 请求后检查是否登录失败
		if c.Writer.Status() == http.StatusUnauthorized && username != "" {
			lgMu.Lock()
			if !exists {
				entry = &loginGuardEntry{firstAttempt: time.Now()}
				lgRecords[ip] = entry
			}
			entry.attempts++
			if entry.attempts >= cfg.MaxAttempts {
				entry.lockedUntil = time.Now().Add(time.Duration(cfg.LockMinutes) * time.Minute)
				entry.attempts = 0
			}
			lgMu.Unlock()
		} else if c.Writer.Status() == http.StatusOK {
			// 登录成功，清除记录
			lgMu.Lock()
			delete(lgRecords, ip)
			lgMu.Unlock()
		}
	}
}
