package proxy

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	tsmodel "apeadmin-gin/internal/plugin/builtin/tailscale/model"

	"gorm.io/gorm"
)

// TailscaleResolver 实现 netproxy.ProxyResolver 接口
type TailscaleResolver struct {
	db       *gorm.DB
	mu       sync.RWMutex
	proxyURL *url.URL
}

func NewTailscaleResolver(db *gorm.DB) *TailscaleResolver {
	r := &TailscaleResolver{db: db}
	r.RefreshConfig()
	return r
}

func (r *TailscaleResolver) Name() string {
	return "tailscale"
}

// RefreshConfig 从数据库刷新代代理配置
func (r *TailscaleResolver) RefreshConfig() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db == nil {
		return
	}

	var cfgs []tsmodel.TsConfig
	if err := r.db.Where("key = ?", "tailscale_proxy_url").Limit(1).Find(&cfgs).Error; err == nil && len(cfgs) > 0 && cfgs[0].Value != "" {
		if u, err := url.Parse(cfgs[0].Value); err == nil {
			r.proxyURL = u
		}
	}
}

// ResolveProxy 检查 HTTP 请求的目标地址是否为 Tailscale 内网节点或声明的子网
func (r *TailscaleResolver) ResolveProxy(req *http.Request) (*url.URL, bool, error) {
	if req == nil || req.URL == nil {
		return nil, false, nil
	}

	hostname := req.URL.Hostname()
	if hostname == "" {
		return nil, false, nil
	}

	// 1. 匹配 MagicDNS 域名 (*.ts.net)
	if strings.HasSuffix(strings.ToLower(hostname), ".ts.net") {
		return r.getProxyURL(), true, nil
	}

	// 2. 匹配 100.64.0.0/10 (Tailscale 预留 CGNAT 网段)
	ip := net.ParseIP(hostname)
	if ip != nil {
		_, tsCIDR, _ := net.ParseCIDR("100.64.0.0/10")
		if tsCIDR != nil && tsCIDR.Contains(ip) {
			return r.getProxyURL(), true, nil
		}
	}

	// 3. 匹配数据库镜像表中的设备 Hostname/Name 或 广播的 Subnet Routes
	if r.db != nil {
		var devices []tsmodel.TsDeviceCache
		if err := r.db.Select("name, hostname, ips, subnet_routes").Find(&devices).Error; err == nil {
			for _, dev := range devices {
				// 匹配设备名或主机名
				if strings.EqualFold(dev.Name, hostname) || strings.EqualFold(dev.Hostname, hostname) {
					return r.getProxyURL(), true, nil
				}

				// 匹配设备 IP
				if dev.IPs != "" && strings.Contains(dev.IPs, hostname) {
					return r.getProxyURL(), true, nil
				}

				// 匹配广播的局域网子网 (Subnet Routes)
				if ip != nil && dev.SubnetRoutes != "" {
					var routes struct {
						EnabledRoutes []string `json:"enabledRoutes"`
					}
					if err := json.Unmarshal([]byte(dev.SubnetRoutes), &routes); err == nil {
						for _, routeCIDR := range routes.EnabledRoutes {
							_, cidr, err := net.ParseCIDR(routeCIDR)
							if err == nil && cidr.Contains(ip) {
								return r.getProxyURL(), true, nil
							}
						}
					}
				}
			}
		}
	}

	return nil, false, nil
}

func (r *TailscaleResolver) getProxyURL() *url.URL {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.proxyURL
}
