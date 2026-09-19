<div align="center">
  <br/>
  <img src="assets/logo.png" width="130" alt="Admin-crm Logo" />
  <h1>Admin-crm (ApeAdmin-Gin)</h1>
  <p>面向云原生与高并发场景的 AI 智能体 + 物联网 (IoT) 后台管理框架</p>
</div>

<p align="center">
  <a href="https://crm.gzyaowei.cn/">线上管理后台 (Vercel)</a> ·
  <a href="https://handicapped-teodora-diyvv-ddf9b75d.koyeb.app/health">后端 API 健康监测 (Koyeb)</a> ·
  <a href="#功能特性">功能特性</a> ·
  <a href="#云原生部署">云原生部署</a> ·
  <a href="#海康互联插件-mvp">海康插件</a> ·
  <a href="#配置说明">配置说明</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-0.2.0-blue" alt="version">
  <img src="https://img.shields.io/badge/go-1.26%2B-00ADD8" alt="go">
  <img src="https://img.shields.io/badge/gin-1.12-brightgreen" alt="gin">
  <img src="https://img.shields.io/badge/database-PostgreSQL%20%7C%20MySQL%20%7C%20SQLite-blue" alt="database">
  <img src="https://img.shields.io/badge/cloud-Koyeb%20%7C%20Vercel-black" alt="cloud">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="license">
</p>

---

> **本项目由 AI 驱动开发，人工负责产品与质量。** 项目在 AI 辅助下高效迭代：人类开发者负责产品方向、架构评审、质量验证与最终决策，AI 负责加速实现、测试和文档工作。

---

**Admin-crm (ApeAdmin-Gin)** 是基于 **Go (Gin + GORM) + Vue 3 (Element Plus)** 打造的高性能后台管理与物联网插件底座，原生适配 **AI Agent MCP (Model Context Protocol)** 协议与云原生容器化部署。

框架天生适配高并发业务系统、多实例横向扩展与云端免费容器托管环境（如 Koyeb + Vercel + PostgreSQL）。具备三级插件化扩展能力、标准 RBAC 权限体系、海康威视（Hik-Connect）物联网门禁考勤插件及 AI 智能体工具调用网关。

---

## 🌟 演示环境

无需本地搭建，可直接访问云端部署演示：

- **线上后台地址**：[https://crm.gzyaowei.cn/](https://crm.gzyaowei.cn/)（部署于 Vercel，绑定独立域名）
- **后端 API 健康节点**：[https://handicapped-teodora-diyvv-ddf9b75d.koyeb.app/health](https://handicapped-teodora-diyvv-ddf9b75d.koyeb.app/health)（部署于 Koyeb）
- **测试账号**：`admin`
- **默认密码**：`admin123`

---

## ✨ 功能特性

### 1. 云原生与多数据库引擎驱动
- **PostgreSQL / MySQL / SQLite 全兼容**：原生接入 `gorm.io/driver/postgres`，支持 PostgreSQL 云数据库无缝自动迁移。
- **云平台零配置环境变量识别**：代码自动优先识别 `DATABASE_URL`（Koyeb / Neon / Heroku 自动注入）与 `PORT` 端口。
- **开箱即用多阶段 Dockerfile**：轻量级镜像打包，包含 SSL/TLS 证书及 CA 支持，适应任何容器平台。

### 2. 海康互联 (Hik-Connect) 物联网插件 (`hikiot`)
- **HMAC-SHA256 签名鉴权**：符合海康开放平台 `x-ca-key` / `x-ca-signature` 规范。
- **门禁点设备在线管控与远程控门**：设备在线/离线状态指示，提供一键远程开门、关门、常开指令下发。
- **组织架构与人员档案同步**：支持一键从海康云端拉取同步组织树与员工档案。
- **打卡考勤记录检索**：支持按人员姓名、日期区间查询打卡记录，区分人脸识别与刷卡通行模式。
- **内置沙箱 Mock 模式**：平台审核期或未配置密钥时自适应开启模拟沙箱，界面与 API 可全流程预览调测。
- **自动化菜单与权限注入**：插件启动时自动在数据库 `sys_menu` 中挂载“海康互联”管理面板及角色授权。

### 3. MCP AI Agent 网关 (Model Context Protocol)
- **管理能力工具化**：将底座及海康插件功能封装为标准 MCP Tools 供大模型调用：
  - `hikiot_door_control`: AI 大模型可通过自然语言（如“帮我开启一楼大门”）下发控门指令。
  - `hikiot_attendance_query`: AI 查询员工考勤打卡记录。
  - `hikiot_person_search`: AI 搜索员工档案与部门信息。
  - `system_health_check`: 系统健康运维检查。
- **RBAC 鉴权过滤与审计**：带权限限制的工具调用，全程写入审计日志 `sys_mcp_audit_log`。

### 4. 插件化架构与生态 (Plugins)
- **三级插件体系**：
  - **L1 编译内置插件**（如 `hikiot` 海康插件）：支持全量后端逻辑、数据库建表、HTTP 路由及 MCP 工具。
  - **L2 声明式 ZIP 插件**：通过后台动态上传安装（含安全 Zip Bomb 防护、`menu.json` 菜单注入与 `seed.sql` 预置数据）。
  - **L3 外部进程插件**（预留扩展）。

### 5. RBAC 权限与安全防护
- **标准五表模型**：用户 / 角色 / 菜单 / 部门，支持无限级树形结构。
- **四层权限粒度**：免登录 $\rightarrow$ 仅登录 $\rightarrow$ 规则鉴权（`RequirePermission`）$\rightarrow$ 数据范围（本人/本部门/全域）。
- **安全熔断与防护**：双令牌 JWT（Access/Refresh）、登录防爆破（Login Guard）、IP 级别 API 限流、生产环境 JWT Secret 强制校验熔断。

---

## 🛠️ 技术栈

| 模块 | 技术选型 | 说明 |
| :--- | :--- | :--- |
| **后端框架** | Go 1.26+ / Gin v1.12 / GORM v1.31 | 原生高并发，低内存占用 |
| **数据库** | PostgreSQL / MySQL / SQLite | 自动识别云端 `DATABASE_URL`，自动迁移建表 |
| **前端体系** | Vue 3.5 / TypeScript / Vite / Element Plus | 已部署至 Vercel (`crm.gzyaowei.cn`) |
| **云托管平台**| Koyeb (Go 服务 + Postgres) + Vercel (前端 CDN) | 纯云原生容器化部署 |
| **配置与日志**| Viper v1.21 / Zap v1.28 | 支持 `GA_` 前缀环境变量覆盖，异步队列日志 |
| **AI / 物联网**| MCP 协议 / 海康互联 Open API | 开放平台 HMAC-SHA256 签名，AI Agent 工具调用 |

---

## 🚀 云原生部署指南

项目采用前后端分离云部署（Koyeb 后端 + Vercel 前端）：

```
    【用户浏览器】
         │
         ├──► 访问前端：https://crm.gzyaowei.cn/ (Vercel 托管)
         │       │
         │       └──► /api/v1 反向代理 (vercel.json)
         │               │
         └───────────────┼────────► 访问后端：https://<your-koyeb-app>.koyeb.app (Koyeb 托管)
                                         │
                                         └──► 连接数据库：PostgreSQL (DATABASE_URL)
```

### 1. 后端部署（Koyeb 平台）

1. 在 Koyeb 中选择 **Create Service** $\rightarrow$ **PostgreSQL** 创建免费数据库。
2. 新建 **Web Service**，选择 GitHub 仓库 `Admin-crm`。
3. 构建方式选择 **Dockerfile**（系统会自动使用根目录的多阶段构建 `Dockerfile`）。
4. 在 **Environment Variables** 填入以下环境变量：

| 环境变量名 | 示例值 | 说明 |
| :--- | :--- | :--- |
| `DATABASE_URL` | `postgres://user:pass@ep-xxx.koyeb.app/koyebdb` | 选择关联 Postgres 或粘贴连接串（自动补全 `sslmode=require`） |
| `GA_JWT_SECRET` | `your-secure-random-jwt-secret-32chars!` | 生产模式必须设置 32 位以上字符串 |
| `GA_APP_DEBUG` | `false` | 生产模式关闭调试 |

5. 点击 **Deploy** 提交，Koyeb 会自动编译 Go 二进制镜像并启动服务。

### 2. 前端部署（Vercel 平台）

1. 在 Vercel 中导入 GitHub 仓库 `Admin-crm`。
2. 项目配置：
   - **Framework Preset**：选择 `Other`
   - **Build Command**：留空
   - **Output Directory**：填入 `release-package/frontend/dist`
3. 根目录已内置 `vercel.json`，会自动将所有 `/api/*` 接口请求反向代理到 Koyeb 后端，零跨域、零额外环境变量配置！

---

## 🔌 海康互联 (Hik-Connect) 插件开发与配置

海康插件代码位于 `internal/plugin/builtin/hikiot/`：

### 1. HTTP 路由 API

- `POST /api/v1/hikiot/config`：保存海康 AppKey / AppSecret 凭据
- `GET /api/v1/hikiot/doors`：获取门禁点设备在线列表
- `POST /api/v1/hikiot/doors/control`：远程控制门禁点（`command`: 0:关门, 1:开门, 2:常开）
- `GET /api/v1/hikiot/attendance/records`：打卡考勤记录列表
- `POST /api/v1/hikiot/sync/persons`：同步海康组织结构与人员列表

### 2. AI Agent MCP 工具调用

可在 AI 对话中直接调用以下工具：
```json
// 工具 1: 控门指令
{
  "name": "hikiot_door_control",
  "arguments": { "door_index_code": "D1001", "command": 1 }
}

// 工具 2: 考勤查询
{
  "name": "hikiot_attendance_query",
  "arguments": { "person_name": "张伟", "start_date": "2026-09-01" }
}
```

---

## 💻 本地开发调试

```bash
# 1. 运行本地后端 (默认 SQLite)
go run cmd/server/main.go

# 2. 指定 PostgreSQL 本地测试
GA_DATABASE_TYPE=postgres GA_DATABASE_URL="postgres://postgres:password@localhost:5432/apeadmin_gin?sslmode=disable" go run cmd/server/main.go

# 默认管理员账号密码
# 账号: admin  密码: admin123
```

---

## 📁 项目目录结构

```text
Admin-crm/
├── cmd/
│   └── server/          # 程序入口 main.go
├── configs/             # 配置文件 config.yaml
├── Dockerfile           # 多阶段容器化构建镜像
├── vercel.json          # Vercel 路由与 API 反向代理配置
├── internal/
│   ├── api/             # HTTP 路由与 Controller
│   ├── bootstrap/       # 应用启动编排逻辑 (app.go)
│   ├── config/          # Viper 配置与环境变量解析 (DATABASE_URL, PORT 识别)
│   ├── core/            # 基础核心 (PostgreSQL/MySQL/SQLite DB, JWT, AuditQueue)
│   ├── dal/             # 数据访问层
│   ├── mcp/             # MCP 协议网关与 AI 工具管理器
│   ├── middleware/      # 中间件 (JWT, CORS, 限流, 登录防护)
│   ├── model/           # GORM 实体定义
│   └── plugin/          # 插件系统核心与内置插件目录
│       └── builtin/
│           ├── hikiot/  # 海康互联门禁考勤物联网插件 (L1)
│           ├── hello/   # L1 示例插件
│           └── dev_example/
└── release-package/
    └── frontend/dist/   # 已打包构建完成的 Vue 3 前端静态产物
```

---

## 📄 开源协议

本项目采用 [MIT License](LICENSE) 开源协议。
