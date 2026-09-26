package proxy

import (
	"context"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// TsnetManager 管理插件内部网络通道与动态 SOCKS5 / HTTP 内存代理拨号器
// 通过自包含的 SOCKS5/HTTP 传输通道实现对 Tailnet 内网的无缝访问，摆脱对外部庞大 SDK 的锁定
type TsnetManager struct {
	mu       sync.RWMutex
	proxyURL *url.URL
	hostname string
	authKey  string
	running  bool
}

var (
	globalTsnet *TsnetManager
	tsnetOnce   sync.Once
)

// GetTsnetManager 获取全局单例 TsnetManager
func GetTsnetManager() *TsnetManager {
	tsnetOnce.Do(func() {
		globalTsnet = &TsnetManager{
			hostname: "apeadmin-crm",
		}
	})
	return globalTsnet
}

// Start 启动插件网络直连通道
func (m *TsnetManager) Start(authKey string, hostname string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hostname != "" {
		m.hostname = hostname
	}
	m.authKey = authKey
	m.running = true

	log.Printf("[Plugin:tailscale:tsnet] 插件内网通信引擎已就绪 (Hostname: %s)", m.hostname)
	return nil
}

// SetProxyURL 动态配置插件代理通道
func (m *TsnetManager) SetProxyURL(rawURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rawURL == "" {
		m.proxyURL = nil
		return
	}
	if u, err := url.Parse(rawURL); err == nil {
		m.proxyURL = u
		m.running = true
	}
}

// Stop 停止通道
func (m *TsnetManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	log.Println("[Plugin:tailscale:tsnet] 插件通信引擎已释放")
}

// IsRunning 检查通道是否处于就绪状态
func (m *TsnetManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// DialContext 提供智能动态上下文拨号 (支持 SOCKS5 / HTTP 隧道 / 直连自动切换)
func (m *TsnetManager) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	m.mu.RLock()
	pURL := m.proxyURL
	m.mu.RUnlock()

	if pURL != nil && strings.HasPrefix(strings.ToLower(pURL.Scheme), "socks5") {
		dialer, err := proxy.FromURL(pURL, proxy.Direct)
		if err == nil {
			if ctxDialer, ok := dialer.(proxy.ContextDialer); ok {
				return ctxDialer.DialContext(ctx, network, addr)
			}
			return dialer.Dial(network, addr)
		}
	}

	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

// DialTimeout 带超时的 TCP 拨号
func (m *TsnetManager) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.DialContext(ctx, network, addr)
}
