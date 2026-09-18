package mcp

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"apeadmin-gin/internal/config"
	"apeadmin-gin/internal/model"
	"gorm.io/gorm"
)

// Manager MCP 管理器
type Manager struct {
	db        *gorm.DB
	cfg       *config.Config
	mu        sync.RWMutex
	tools     map[string]*ToolEntry
	resources map[string]*ResourceEntry
	prompts   map[string]*PromptEntry

	semMu sync.Mutex // 保护 sem 的惰性初始化
	sem   chan struct{}
}

// 工具调用错误定义
var (
	ErrToolNotFound = errors.New("工具不存在")
	ErrToolTimeout  = errors.New("工具调用超时")
	ErrToolBusy     = errors.New("工具并发调用数已达上限")
)

// Registrar 插件注册 MCP 工具的入口（等价于 *Manager）
type Registrar = Manager

// ToolEntry MCP 工具注册条目
type ToolEntry struct {
	Name                string
	Description         string
	InputSchema         map[string]interface{}
	PluginName          string
	Category            string
	RequiredPermissions []string
	Handler             ToolHandler
}

// ResourceEntry MCP 资源注册条目
type ResourceEntry struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	ReadHandler func() (string, error)
}

// PromptEntry MCP 提示词注册条目
type PromptEntry struct {
	Name        string
	Description string
	Arguments   []string
	Template    string // 支持 {arg} 占位符
}

// ToolHandler MCP 工具处理函数
type ToolHandler func(args map[string]interface{}) (interface{}, error)

// toolResult 工具调用结果（用于超时 goroutine 传递）
type toolResult struct {
	data interface{}
	err  error
}

// ExecuteTool 带超时与并发控制的工具调用。
// 并发上限取自 cfg.MCP.MaxConcurrency（全局信号量，防止整体打满）；
// 单次调用超时取自 cfg.MCP.Timeout（秒）。
// 注意：超时后底层 goroutine 仍可能继续执行直到自行返回（同步 Handler 无法强制中断），
// 通过带缓冲的 done channel 保证其完成后不会泄漏阻塞。
func (m *Manager) ExecuteTool(name string, args map[string]interface{}) (interface{}, error) {
	t, ok := m.GetTool(name)
	if !ok {
		return nil, ErrToolNotFound
	}
	timeoutSec := 30
	maxConc := 10
	if m.cfg != nil {
		if m.cfg.MCP.Timeout > 0 {
			timeoutSec = m.cfg.MCP.Timeout
		}
		if m.cfg.MCP.MaxConcurrency > 0 {
			maxConc = m.cfg.MCP.MaxConcurrency
		}
	}

	// 信号量（惰性初始化，大小 = maxConc）
	m.semMu.Lock()
	if m.sem == nil {
		m.sem = make(chan struct{}, maxConc)
	}
	sem := m.sem
	m.semMu.Unlock()

	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	default:
		return nil, ErrToolBusy
	}

	// 超时控制：同步 Handler 放入 goroutine + select 双通道
	done := make(chan toolResult, 1)
	go func() {
		r, err := t.Handler(args)
		done <- toolResult{data: r, err: err}
	}()
	select {
	case res := <-done:
		return res.data, res.err
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		return nil, ErrToolTimeout
	}
}

// NewManager 创建 MCP 管理器
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		db:        db,
		tools:     make(map[string]*ToolEntry),
		resources: make(map[string]*ResourceEntry),
		prompts:   make(map[string]*PromptEntry),
	}
}

// SetConfig 注入配置引用（bootstrap 调用）
func (m *Manager) SetConfig(cfg *config.Config) {
	m.cfg = cfg
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *config.Config {
	return m.cfg
}

// GetDB 获取数据库
func (m *Manager) GetDB() *gorm.DB {
	return m.db
}

// RegisterTool 注册 MCP 工具
func (m *Manager) RegisterTool(entry *ToolEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.Category == "" {
		entry.Category = "system"
	}
	m.tools[entry.Name] = entry
}

// UnregisterPluginTools 注销某插件的所有工具
func (m *Manager) UnregisterPluginTools(pluginName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, t := range m.tools {
		if t.PluginName == pluginName {
			delete(m.tools, name)
		}
	}
}

// ListTools 列出所有工具（按名称排序）
func (m *Manager) ListTools() []*ToolEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*ToolEntry, 0, len(m.tools))
	for _, t := range m.tools {
		result = append(result, t)
	}
	// 简单排序保证顺序稳定
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Name < result[i].Name {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// GetTool 按名称查找工具
func (m *Manager) GetTool(name string) (*ToolEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tools[name]
	return t, ok
}

// ListCategories 工具分类统计
func (m *Manager) ListCategories() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counter := make(map[string]int)
	for _, t := range m.tools {
		cat := t.Category
		if cat == "" {
			cat = "system"
		}
		counter[cat]++
	}
	result := make([]map[string]interface{}, 0, len(counter))
	for name, count := range counter {
		result = append(result, map[string]interface{}{"name": name, "count": count})
	}
	return result
}

// RegisterResource 注册资源
func (m *Manager) RegisterResource(entry *ResourceEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources[entry.URI] = entry
}

// ListResources 列出所有资源
func (m *Manager) ListResources() []*ResourceEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*ResourceEntry, 0, len(m.resources))
	for _, r := range m.resources {
		result = append(result, r)
	}
	return result
}

// ReadResource 读取资源内容
func (m *Manager) ReadResource(uri string) (string, bool) {
	m.mu.RLock()
	r, ok := m.resources[uri]
	m.mu.RUnlock()
	if !ok {
		return "", false
	}
	content, err := r.ReadHandler()
	if err != nil {
		return "", false
	}
	return content, true
}

// RegisterPrompt 注册提示词
func (m *Manager) RegisterPrompt(entry *PromptEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prompts[entry.Name] = entry
}

// ListPrompts 列出所有提示词
func (m *Manager) ListPrompts() []*PromptEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*PromptEntry, 0, len(m.prompts))
	for _, p := range m.prompts {
		result = append(result, p)
	}
	return result
}

// RenderPrompt 渲染提示词
func (m *Manager) RenderPrompt(name string, args map[string]string) (string, bool) {
	m.mu.RLock()
	p, ok := m.prompts[name]
	m.mu.RUnlock()
	if !ok {
		return "", false
	}
	rendered := p.Template
	for k, v := range args {
		rendered = strings.ReplaceAll(rendered, "{"+k+"}", v)
	}
	return rendered, true
}

// WriteAuditLog 写入 MCP 审计日志
func (m *Manager) WriteAuditLog(userID uint, username, actionType, targetName, arguments, result, status string, durationMs int64) {
	if m.db == nil {
		return
	}
	truncate := func(s string, n int) string {
		if len(s) > n {
			return s[:n]
		}
		return s
	}
	log := model.SysMcpAuditLog{
		UserID:     userID,
		Username:   username,
		ActionType: actionType,
		TargetName: targetName,
		Arguments:  truncate(arguments, 2000),
		Result:     truncate(result, 2000),
		Status:     status,
		DurationMs: durationMs,
	}
	m.db.Create(&log)
}

// MarshalSchema 序列化 input_schema（辅助函数）
func MarshalSchema(schema map[string]interface{}) string {
	if schema == nil {
		return ""
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return ""
	}
	return string(b)
}
