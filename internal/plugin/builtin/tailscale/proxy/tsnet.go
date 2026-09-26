package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

// TsnetManager 管理插件内部嵌入式 Tailscale 节点生命周期与内存拨号
type TsnetManager struct {
	mu       sync.RWMutex
	server   *tsnet.Server
	hostname string
	authKey  string
	dir      string
	running  bool
}

var (
	globalTsnet *TsnetManager
	tsnetOnce   sync.Once
)

// GetTsnetManager 获取全局单例 TsnetManager
func GetTsnetManager() *TsnetManager {
	tsnetOnce.Do(func() {
		stateDir := filepath.Join(os.TempDir(), "apeadmin_tsnet")
		_ = os.MkdirAll(stateDir, 0700)
		globalTsnet = &TsnetManager{
			hostname: "apeadmin-crm",
			dir:      stateDir,
		}
	})
	return globalTsnet
}

// Start 启动内存中的 Tailscale 节点
func (m *TsnetManager) Start(authKey string, hostname string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if authKey == "" {
		return fmt.Errorf("未配置 AuthKey")
	}

	if m.running && m.server != nil && m.authKey == authKey {
		return nil
	}

	// 若已有旧实例先关闭
	if m.server != nil {
		_ = m.server.Close()
		m.server = nil
		m.running = false
	}

	if hostname != "" {
		m.hostname = hostname
	}
	m.authKey = authKey

	s := &tsnet.Server{
		Dir:      m.dir,
		Hostname: m.hostname,
		AuthKey:  m.authKey,
		Logf:     func(format string, args ...any) {}, // 静默底层调试日志
	}

	m.server = s
	m.running = true
	log.Printf("[Plugin:tailscale:tsnet] 嵌入式用户态节点已初始化 (Hostname: %s)", m.hostname)
	return nil
}

// Stop 停止嵌入式节点
func (m *TsnetManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.server != nil {
		_ = m.server.Close()
		m.server = nil
	}
	m.running = false
	log.Println("[Plugin:tailscale:tsnet] 嵌入式用户态节点已停止并释放资源")
}

// IsRunning 检查当前 tsnet 节点是否已启用
func (m *TsnetManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running && m.server != nil
}

// DialContext 原生使用 WireGuard 内存网络进行 TCP 拨号
func (m *TsnetManager) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	m.mu.RLock()
	s := m.server
	running := m.running
	m.mu.RUnlock()

	if !running || s == nil {
		return nil, fmt.Errorf("tsnet 节点尚未启动 (请先在配置中填入 Auth Key)")
	}

	return s.Dial(ctx, network, addr)
}

// DialTimeout 带超时的 TCP 拨号
func (m *TsnetManager) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.DialContext(ctx, network, addr)
}
