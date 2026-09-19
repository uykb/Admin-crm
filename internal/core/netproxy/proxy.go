package netproxy

import (
	"net/http"
	"net/url"
	"sync"
)

// ProxyResolver 代理解析器接口，由 Tailscale 等组网插件动态实现
type ProxyResolver interface {
	// Name 返回解析器唯一标识名称
	Name() string
	// ResolveProxy 检查请求目标，若需走内网代理则返回代理 URL 和 true
	ResolveProxy(req *http.Request) (proxyURL *url.URL, shouldProxy bool, err error)
}

var (
	mu        sync.RWMutex
	resolvers []ProxyResolver
	once      sync.Once
)

// RegisterResolver 注册动态代理解析器（插件启动时调用）
func RegisterResolver(r ProxyResolver) {
	mu.Lock()
	defer mu.Unlock()

	for _, existing := range resolvers {
		if existing.Name() == r.Name() {
			return
		}
	}
	resolvers = append(resolvers, r)
	initGlobalTransport()
}

// UnregisterResolver 注销动态代理解析器（插件卸载/停用时调用）
func UnregisterResolver(name string) {
	mu.Lock()
	defer mu.Unlock()

	newResolvers := make([]ProxyResolver, 0, len(resolvers))
	for _, r := range resolvers {
		if r.Name() != name {
			newResolvers = append(newResolvers, r)
		}
	}
	resolvers = newResolvers
}

// GetResolvers 获取当前所有注册的代理解析器
func GetResolvers() []ProxyResolver {
	mu.RLock()
	defer mu.RUnlock()
	copied := make([]ProxyResolver, len(resolvers))
	copy(copied, resolvers)
	return copied
}

// initGlobalTransport 全局注入透明 HTTP Proxy Hook
func initGlobalTransport() {
	once.Do(func() {
		originalTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			originalTransport = &http.Transport{}
		}

		// 挂载代理决策钩子
		originalTransport.Proxy = func(req *http.Request) (*url.URL, error) {
			activeResolvers := GetResolvers()
			for _, r := range activeResolvers {
				proxyURL, shouldProxy, err := r.ResolveProxy(req)
				if err == nil && shouldProxy && proxyURL != nil {
					return proxyURL, nil
				}
			}
			// 默认使用系统环境变量代理
			return http.ProxyFromEnvironment(req)
		}

		http.DefaultTransport = originalTransport
	})
}

// NewClient 获取自带透明代理能力的 HTTP 客户端
func NewClient() *http.Client {
	initGlobalTransport()
	return &http.Client{
		Transport: http.DefaultTransport,
	}
}
