# ApeAdmin-Gin 部署说明

ApeAdmin-Gin 是基于 Go + Gin + GORM 的插件化后台框架（后端），前端为 Vue3 管理后台（已内置于本包，无需单独部署）。

## 一、目录结构

```
ApeAdmin-Gin/
├── bin/
│   ├── ApeAdmin-Gin_windows_amd64.exe   # Windows 版
│   └── ApeAdmin-Gin_linux_amd64         # Linux 版
├── configs/
│   └── config.yaml                      # 配置文件
├── frontend/
│   └── dist/                            # 前端构建产物（SPA）
├── README-DEPLOY.md                     # 本说明
└── LICENSE
```

## 二、快速开始（Windows）

1. 解压本包到任意目录（路径建议不含空格与中文）。
2. 双击运行 `bin\ApeAdmin-Gin_windows_amd64.exe`，或命令行执行：
   ```powershell
   cd ApeAdmin-Gin
   .\bin\ApeAdmin-Gin_windows_amd64.exe
   ```
3. 浏览器访问 `http://127.0.0.1:8001/admin/`，默认账号 `admin` / `admin123`。

## 三、快速开始（Linux）

```bash
cd ApeAdmin-Gin
chmod +x bin/ApeAdmin-Gin_linux_amd64
./bin/ApeAdmin-Gin_linux_amd64
```

浏览器访问 `http://服务器IP:8001/admin/`，默认账号 `admin` / `admin123`。

> 注意：程序会自动在工作目录下创建 `apeadmin_gin.db`（SQLite 数据库）、`logs/`、`uploads/` 目录。**首次启动即自动初始化（建表、种子数据、默认菜单）**。

## 四、关键配置说明（configs/config.yaml）

| 配置项 | 默认值 | 说明 |
|---|---|---|
| `app.port` | `8001` | HTTP 监听端口 |
| `app.admin_path` | `/admin` | 前端后台路径前缀（前端构建产物已绑定该前缀，勿改） |
| `app.spa_dir` | `frontend/dist` | 前端构建产物目录（相对工作目录） |
| `app.debug` | `true` | 生产环境必须改 `false`（会强制校验 jwt.secret） |
| `database.type` | `sqlite` | `sqlite` 或 `mysql`；MySQL 需填写 host/port/user/password/dbname |
| `jwt.secret` | 默认值 | **生产必须修改**为 32+ 位随机字符串（debug=false 时强制） |
| `redis.enabled` | `false` | 多实例部署 / 需要重启后 token 仍有效时开启（需 Redis 服务） |
| `mcp.timeout` | `30` | 单次 MCP 工具调用超时（秒） |
| `mcp.max_concurrency` | `10` | MCP 工具并发上限 |
| `file.storage_dir` | `uploads/files` | 上传文件存储目录 |
| `security.rate_limit` | 启用 | API 限流：每 IP 每分钟 300 次 |
| `security.login_guard` | 启用 | 登录防爆破：15 分钟内错 5 次锁 30 分钟 |

## 五、环境变量覆盖

所有配置项均支持环境变量覆盖，规则：`GA_` 前缀 + 配置路径下划线连接。例如：

```
GA_APP_PORT=9000
GA_DATABASE_TYPE=mysql
GA_DATABASE_HOST=192.168.1.100
GA_JWT_SECRET=your-production-secret-xxxxxxxx
```

## 六、生产部署建议

1. **修改密钥**：`jwt.secret` 换成 32+ 位随机值；`app.debug` 改为 `false`。
2. **数据库**：生产建议 MySQL（`database.type: mysql`），并关闭 `database.auto_migrate`。
3. **反向代理**：建议前面挂 Nginx 做 HTTPS，代理到 `127.0.0.1:8001`。
4. **更新**：管理后台「系统设置 → 检查更新」支持上传更新包（zip）。
5. **防火墙**：如非本机访问，需放行 8001 端口（或反代后仅放行 80/443）。

## 七、常见问题

**Q：启动报 "生产环境（debug=false）必须修改 jwt.secret"？**
A：生产配置必须自定义 JWT 密钥，编辑 `configs/config.yaml` 中 `jwt.secret` 后重启。

**Q：页面空白？**
A：确认 `frontend/dist` 目录与二进制同级（即解压后目录结构未被破坏），且访问路径是 `/admin/`。

**Q：MySQL 连不上？**
A：检查 `database.*` 配置项与 MySQL 账号权限；首次连接会自动建库表。

**Q：忘记 admin 密码？**
A：删除 `apeadmin_gin.db` 会重建（**数据全部丢失**），或联系管理员通过数据库修改。生产环境请提前绑定邮箱/手机。

**Q：多实例/重启后登录失效？**
A：开启 `redis.enabled: true`（多实例必须，token 状态集中存储），并在环境变量注入 `GA_REDIS_URL`。
