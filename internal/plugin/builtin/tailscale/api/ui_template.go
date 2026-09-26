package api

const TailscaleUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Tailscale 万物互联与设备管控中枢</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.css" />
  <script src="https://cdn.jsdelivr.net/npm/vue@3.4.27/dist/vue.global.prod.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.full.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/@element-plus/icons-vue@2.3.1/dist/index.iife.min.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f7fa; margin: 0; padding: 20px; color: #303133; }
    .header { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); margin-bottom: 20px; display: flex; justify-content: space-between; align-items: center; }
    .header h2 { margin: 0; font-size: 20px; color: #1f2937; display: flex; align-items: center; gap: 10px; }
    .stat-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 20px; }
    .stat-card { background: #fff; padding: 16px 20px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; display: flex; align-items: center; justify-content: space-between; }
    .stat-title { font-size: 13px; color: #6b7280; margin-bottom: 4px; }
    .stat-value { font-size: 24px; font-weight: 700; color: #111827; }
    .card-box { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); }
    .device-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 16px; margin-top: 15px; }
    .device-card { border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; background: #fafafa; transition: all 0.2s; position: relative; }
    .device-card:hover { border-color: #5A67F5; box-shadow: 0 4px 12px rgba(90, 103, 245, 0.12); background: #fff; }
    .device-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px; }
    .device-name { font-weight: 600; font-size: 16px; color: #111827; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .device-user { font-size: 12px; color: #6b7280; }
    .ip-tag { font-family: monospace; font-size: 12px; margin-right: 4px; margin-bottom: 4px; cursor: pointer; }
    .status-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 6px; }
    .status-online { background: #10b981; }
    .status-offline { background: #9ca3af; }
    .filter-bar { display: flex; gap: 12px; margin-bottom: 15px; align-items: center; flex-wrap: wrap; }
    .key-box { background: #111827; color: #10b981; font-family: monospace; padding: 12px 16px; border-radius: 6px; margin-top: 10px; word-break: break-all; position: relative; }
    .code-preview { background: #282c34; color: #abb2bf; padding: 16px; border-radius: 6px; font-family: monospace; white-space: pre-wrap; word-break: break-all; max-height: 450px; overflow-y: auto; }
    .tag-badge { margin-right: 4px; margin-bottom: 4px; }
    .diag-res-box { background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 6px; padding: 12px; margin-top: 15px; }
  </style>
</head>
<body>
  <div id="app">
    <div class="header">
      <h2>
        <el-icon color="#5A67F5"><connection /></el-icon>
        Tailscale 万物互联与边缘设备管控中枢
      </h2>
      <div>
        <el-tag type="success" effect="dark" round>P1/P2/P3 物联架构 就绪</el-tag>
      </div>
    </div>

    <!-- 概览指标卡 -->
    <div class="stat-row">
      <div class="stat-card">
        <div>
          <div class="stat-title">接入物联/服务器节点</div>
          <div class="stat-value">{{ devices.length }}</div>
        </div>
        <el-icon size="32" color="#5A67F5"><monitor /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">在线节点</div>
          <div class="stat-value" style="color: #10b981;">{{ onlineCount }}</div>
        </div>
        <el-icon size="32" color="#10b981"><circle-check /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">免密钥过期物联设备</div>
          <div class="stat-value" style="color: #f59e0b;">{{ expiryDisabledCount }}</div>
        </div>
        <el-icon size="32" color="#f59e0b"><key /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">所属 Tailnet 网络</div>
          <div class="stat-value" style="font-size: 15px; word-break: break-all;">{{ configForm.tailnet || '未配置' }}</div>
        </div>
        <el-icon size="32" color="#8b5cf6"><share /></el-icon>
      </div>
    </div>

    <div class="card-box">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 标签页 1：组网节点管控 (P1 + P2 + P3) -->
        <el-tab-pane label="组网节点管控" name="devices">
          <div class="filter-bar">
            <el-input v-model="deviceSearch" placeholder="搜索设备名/主机名/IP/Tag" style="width: 280px;" clearable @keyup.enter="loadDevices"></el-input>
            <el-button type="primary" @click="loadDevices">查询节点</el-button>
            <el-button type="success" :loading="syncing" @click="syncDevices">
              <el-icon><refresh /></el-icon> 同步 Tailnet 节点
            </el-button>
            <el-button type="warning" @click="openQuickDiag('')">
              <el-icon><aim /></el-icon> 全网物联协议连通性诊断
            </el-button>
          </div>

          <div class="device-grid" v-loading="loadingDevices">
            <div v-for="d in devices" :key="d.device_id" class="device-card">
              <div class="device-header">
                <div>
                  <div class="device-name" :title="d.name || d.hostname">
                    {{ d.name || d.hostname }}
                    <el-button type="primary" link size="small" @click="openRename(d)"><el-icon><edit /></el-icon></el-button>
                  </div>
                  <div class="device-user">所有者: {{ d.user || '全网共享' }} | 系统: {{ d.os }} (v{{ d.client_version || 'N/A' }})</div>
                </div>
                <el-tag :type="d.online ? 'success' : 'info'" size="small">
                  <span class="status-dot" :class="d.online ? 'status-online' : 'status-offline'"></span>
                  {{ d.online ? '在线' : '离线' }}
                </el-tag>
              </div>

              <!-- 内网 IP -->
              <div style="margin-top: 8px;">
                <div style="font-size: 12px; color: #6b7280; margin-bottom: 2px;">Tailscale 内网 IP:</div>
                <div>
                  <el-tag v-for="ip in parseJSON(d.ips)" :key="ip" class="ip-tag" size="small" type="primary" effect="plain" @click="copyText(ip)">
                    {{ ip }}
                  </el-tag>
                </div>
              </div>

              <!-- 标签 Tags (P3 零信任微隔离) -->
              <div style="margin-top: 6px;">
                <div style="font-size: 12px; color: #6b7280; margin-bottom: 2px;">安全标签 (Tags):</div>
                <div>
                  <el-tag v-for="tag in parseJSON(d.tags)" :key="tag" class="tag-badge" size="small" type="warning" effect="light">
                    {{ tag }}
                  </el-tag>
                  <el-button type="warning" link size="small" @click="openTags(d)">+ 配置标签</el-button>
                </div>
              </div>

              <!-- 子网路由 (P2) -->
              <div style="margin-top: 6px;" v-if="parseSubnetRoutes(d.subnet_routes).length > 0">
                <div style="font-size: 12px; color: #6b7280; margin-bottom: 2px;">广播子网路由:</div>
                <div>
                  <el-tag v-for="r in parseSubnetRoutes(d.subnet_routes)" :key="r" size="small" type="success" effect="plain" class="tag-badge">
                    {{ r }}
                  </el-tag>
                  <el-button type="success" link size="small" @click="openRoutes(d)">审批路由</el-button>
                </div>
              </div>

              <!-- 免过期策略 (P1 物联网必备) -->
              <div style="margin-top: 10px; display: flex; align-items: center; justify-content: space-between; background: #fff; padding: 6px 10px; border-radius: 6px; border: 1px dashed #d1d5db;">
                <span style="font-size: 12px; color: #4b5563;">免密钥过期 (IoT 永不掉线)</span>
                <el-switch :model-value="d.key_expiry_disabled" active-color="#10b981" @change="(val) => toggleKeyExpiry(d, val)"></el-switch>
              </div>

              <!-- 操作区 -->
              <div style="margin-top: 12px; display: flex; justify-content: space-between; align-items: center;">
                <div>
                  <el-button type="primary" link size="small" @click="openQuickDiag(parsePrimaryIP(d.ips))">协议诊断</el-button>
                  <el-button type="warning" link size="small" @click="openSSH(parsePrimaryIP(d.ips))">Web-SSH</el-button>
                  <el-button type="success" link size="small" @click="openRoutes(d)">子网路由</el-button>
                </div>
                <el-popconfirm title="确定下线并注销该设备？" @confirm="deleteDevice(d.device_id)">
                  <template #reference>
                    <el-button type="danger" link size="small">解绑注销</el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 标签页 2：Auth Key 接入密钥生成 (P1) -->
        <el-tab-pane label="Auth Key 密钥管理" name="keys">
          <el-row :gutter="20">
            <el-col :span="10">
              <h3 style="margin-top: 0;">生成设备接入密钥 (Auth Key)</h3>
              <el-form :model="keyForm" label-width="110px">
                <el-form-item label="单机/多机">
                  <el-switch v-model="keyForm.reusable" active-text="多机可复用 (Reusable)" inactive-text="一次性使用"></el-switch>
                </el-form-item>
                <el-form-item label="临时节点">
                  <el-switch v-model="keyForm.ephemeral" active-text="临时节点 (关机自动注销)" inactive-text="持久节点"></el-switch>
                </el-form-item>
                <el-form-item label="免手动核准">
                  <el-switch v-model="keyForm.preauthorized" active-text="自动批准接入 (Pre-authorized)" inactive-text="需管理员核准"></el-switch>
                </el-form-item>
                <el-form-item label="附加 Tags">
                  <el-input v-model="keyForm.tagsStr" placeholder="如 tag:iot-door,tag:camera (多标签逗号分隔)"></el-input>
                </el-form-item>
                <el-form-item label="有效天数">
                  <el-input-number v-model="keyForm.expiry_days" :min="1" :max="365"></el-input-number>
                </el-form-item>
                <el-form-item label="使用用途">
                  <el-input v-model="keyForm.purpose" placeholder="如: 深圳工厂门禁网关接入"></el-input>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" :loading="creatingKey" @click="createAuthKey">立即生成 Key</el-button>
                </el-form-item>
              </el-form>
            </el-col>

            <el-col :span="14">
              <div v-if="newlyCreatedKey">
                <el-alert title="新 Auth Key 已成功生成！" type="success" :closable="false" show-icon></el-alert>
                <div class="key-box">
                  {{ newlyCreatedKey.key }}
                </div>
                <div style="margin-top: 10px;">
                  <el-button type="primary" size="small" @click="copyText(newlyCreatedKey.key)">复制密钥</el-button>
                  <el-button type="success" size="small" @click="copyText('tailscale up --authkey=' + newlyCreatedKey.key)">复制一键接入命令</el-button>
                </div>
              </div>

              <h3 style="margin-top: 20px;">密钥生成历史记录</h3>
              <el-table :data="keyLogs" stripe style="width: 100%;" size="small">
                <el-table-column prop="purpose" label="用途"></el-table-column>
                <el-table-column label="属性" width="160">
                  <template #default="scope">
                    <el-tag size="small" :type="scope.row.reusable ? 'success' : 'info'">{{ scope.row.reusable ? '复用' : '一次性' }}</el-tag>
                    <el-tag size="small" type="warning" v-if="scope.row.ephemeral">临时</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="created_by" label="创建者" width="100"></el-table-column>
                <el-table-column prop="created_at" label="生成时间" width="150"></el-table-column>
                <el-table-column label="操作" width="90">
                  <template #default="scope">
                    <el-button type="danger" link size="small" @click="revokeAuthKey(scope.row.key_id)">撤销</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-col>
          </el-row>
        </el-tab-pane>

        <!-- 标签页 3：零信任 ACL 策略管控 (P3) -->
        <el-tab-pane label="零信任 ACL 策略管控" name="acl">
          <div class="filter-bar">
            <el-button type="primary" @click="loadACL">读取当前 ACL</el-button>
            <el-button type="success" @click="applyTemplate('iot')">应用万物互联微隔离模版</el-button>
            <el-button type="info" @click="applyTemplate('full')">应用全网互通模版</el-button>
            <el-button type="warning" @click="validateACL">语法校验</el-button>
            <el-button type="danger" :loading="savingACL" @click="saveACL">保存并推送生效</el-button>
          </div>
          <el-input type="textarea" :rows="18" v-model="aclContent" font-family="monospace" placeholder="Tailscale HuJSON ACL Policy 内容"></el-input>
        </el-tab-pane>

        <!-- 标签页 4：Webhook 实时遥测事件 (P1) -->
        <el-tab-pane label="Webhook 实时遥测" name="webhook">
          <el-alert title="Tailscale Webhook 事件回调端点已就绪" type="info" :closable="false" show-icon style="margin-bottom: 15px;">
            <template #default>
              <div>请在 Tailscale 控制台 Webhook 设置中配置此接收 URL: <code>/api/v1/tailscale/webhook</code></div>
              <div>当边缘物联节点上下线、密钥到期、子网路由变动时，系统将通过内部 EventBus 自动感知并同步。</div>
            </template>
          </el-alert>
          <div class="filter-bar">
            <el-button type="primary" @click="loadWebhookLogs">刷新遥测事件流</el-button>
          </div>
          <el-table :data="webhookLogs" stripe style="width: 100%;" size="small">
            <el-table-column prop="created_at" label="时间" width="160"></el-table-column>
            <el-table-column prop="event_type" label="事件类型" width="160">
              <template #default="scope">
                <el-tag size="small" type="primary">{{ scope.row.event_type }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="tailnet" label="所属 Tailnet" width="160"></el-table-column>
            <el-table-column prop="message" label="事件描述"></el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 标签页 5：连接与 tsnet 节点配置 (方案二) -->
        <el-tab-pane label="连接与嵌入式节点配置" name="config">
          <el-alert title="方案二：嵌入式 tsnet 用户态节点模式" type="success" :closable="false" show-icon style="margin-bottom: 15px;">
            <template #default>
              <div>只需填入一个预授权的 <b>Auth Key</b>，系统将直接在 Go 内存中拉起 WireGuard 组网引擎，云端容器无需任何系统权限即可实现内网直连与 Web-SSH 远程运维！</div>
            </template>
          </el-alert>
          <el-form :model="configForm" label-width="160px" style="max-width: 700px;">
            <el-form-item label="Tailnet 名称">
              <el-input v-model="configForm.tailnet" placeholder="如: mytailnet.ts.net 或 user@github"></el-input>
            </el-form-item>
            <el-form-item label="嵌入式节点 AuthKey">
              <el-input v-model="configForm.auth_key" type="password" show-password placeholder="tskey-auth-kxxxx (填入后自动激活方案二 tsnet 原生内网直连)"></el-input>
            </el-form-item>
            <el-form-item label="嵌入式节点名称">
              <el-input v-model="configForm.node_hostname" placeholder="默认: apeadmin-crm"></el-input>
            </el-form-item>
            <el-form-item label="tsnet 运行状态">
              <el-tag :type="configForm.tsnet_running ? 'success' : 'info'">
                {{ configForm.tsnet_running ? '● tsnet 嵌入式节点运行中 (直连就绪)' : '○ 尚未激活 (需配置 AuthKey)' }}
              </el-tag>
            </el-form-item>
            <el-divider content-position="left">API 管控凭证 (用于设备/ACL/密钥同步)</el-divider>
            <el-form-item label="API Key">
              <el-input v-model="configForm.api_key" type="password" show-password placeholder="tskey-api-xxx (API Key 或 OAuth 填一即可)"></el-input>
            </el-form-item>
            <el-form-item label="OAuth Client ID">
              <el-input v-model="configForm.client_id" placeholder="kxxxxxx"></el-input>
            </el-form-item>
            <el-form-item label="OAuth Client Secret">
              <el-input v-model="configForm.client_secret" type="password" show-password placeholder="tskey-client-xxxx"></el-input>
            </el-form-item>
            <el-form-item label="Webhook Secret">
              <el-input v-model="configForm.webhook_secret" type="password" show-password placeholder="Tailscale Webhook 签名密钥 (HMAC-SHA256)"></el-input>
            </el-form-item>
            <el-form-item label="备用代理 URL">
              <el-input v-model="configForm.proxy_url" placeholder="可选备用 SOCKS5 代理 (如 socks5://127.0.0.1:1055)"></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingConfig" @click="saveConfig">保存配置并启动节点</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- Web-SSH 终端执行弹窗 (方案二) -->
    <el-dialog v-model="sshDialogVisible" title="Web-SSH 远程运维终端 (tsnet 加密隧道直连)" width="680px">
      <el-form :model="sshForm" label-width="90px">
        <el-row :gutter="15">
          <el-col :span="12">
            <el-form-item label="目标主机">
              <el-input v-model="sshForm.target" placeholder="100.x.x.x 或 MagicDNS"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="端口">
              <el-input-number v-model="sshForm.port" :min="1" :max="65535" style="width: 100%;"></el-input-number>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="用户名">
              <el-input v-model="sshForm.user" placeholder="root"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="登录密码">
          <el-input v-model="sshForm.password" type="password" show-password placeholder="若节点开启 Tailscale SSH 免密可留空"></el-input>
        </el-form-item>
        <el-form-item label="执行命令">
          <el-input v-model="sshForm.command" placeholder="如: uptime, uname -a, df -h, systemctl status docker" @keyup.enter="execSSH"></el-input>
        </el-form-item>
      </el-form>
      <div v-if="sshOutput" style="margin-top: 10px;">
        <div style="font-weight: 600; font-size: 13px; margin-bottom: 4px; color: #4b5563;">终端输出 (Output):</div>
        <div class="code-preview" style="max-height: 260px;">{{ sshOutput }}</div>
      </div>
      <template #footer>
        <el-button @click="sshDialogVisible = false">关闭</el-button>
        <el-button type="primary" :loading="runningSSH" @click="execSSH">执行指令</el-button>
      </template>
    </el-dialog>

    <!-- 诊断弹窗 -->
    <el-dialog v-model="diagDialogVisible" title="物联设备协议与端口连通性诊断" width="550px">
      <el-form :model="diagForm" label-width="100px">
        <el-form-item label="目标地址">
          <el-input v-model="diagForm.target" placeholder="100.x.x.x / 域名 / 局域网 IP"></el-input>
        </el-form-item>
        <el-form-item label="常用协议预设">
          <el-radio-group v-model="diagForm.preset" @change="handlePresetChange">
            <el-radio-button label="Modbus (502)"></el-radio-button>
            <el-radio-button label="RTSP (554)"></el-radio-button>
            <el-radio-button label="MQTT (1883)"></el-radio-button>
            <el-radio-button label="HTTP (80)"></el-radio-button>
            <el-radio-button label="SSH (22)"></el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="目标端口">
          <el-input-number v-model="diagForm.port" :min="1" :max="65535"></el-input-number>
        </el-form-item>
      </el-form>
      <div v-if="diagResult" class="diag-res-box">
        <div style="font-weight: 600; margin-bottom: 6px;">
          诊断结果:
          <el-tag :type="diagResult.connected ? 'success' : 'danger'" size="small">
            {{ diagResult.connected ? '链路畅通 (Reachable)' : '无法连通 (Unreachable)' }}
          </el-tag>
        </div>
        <div style="font-size: 13px; color: #4b5563;">响应延迟: <b>{{ diagResult.latency_ms }} ms</b></div>
        <div style="font-size: 13px; color: #4b5563;">诊断通道: {{ diagResult.channel || '本地直连' }}</div>
        <div style="font-size: 13px; color: #4b5563;">目标服务: {{ diagResult.preset_hint }}</div>
        <div v-if="diagResult.error" style="font-size: 12px; color: #ef4444; margin-top: 4px;">错误详情: {{ diagResult.error }}</div>
        <div v-if="diagResult.tip" style="font-size: 12px; color: #d97706; margin-top: 6px; background: #fef3c7; padding: 6px 8px; border-radius: 4px;">{{ diagResult.tip }}</div>
      </div>
      <template #footer>
        <el-button @click="diagDialogVisible = false">关闭</el-button>
        <el-button type="primary" :loading="runningDiag" @click="runDiagnose">立即测试连通性</el-button>
      </template>
    </el-dialog>

    <!-- 改名弹窗 -->
    <el-dialog v-model="renameDialogVisible" title="修改设备名称 / MagicDNS 域名" width="400px">
      <el-form>
        <el-form-item label="设备名称">
          <el-input v-model="renameForm.name" placeholder="如: shenzhen-gateway-01"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRename">确认修改</el-button>
      </template>
    </el-dialog>

    <!-- 标签弹窗 -->
    <el-dialog v-model="tagsDialogVisible" title="配置设备安全标签 (Tags)" width="450px">
      <el-form>
        <el-form-item label="Tags (逗号分隔)">
          <el-input v-model="tagsForm.tagsStr" placeholder="如: tag:iot-door,tag:camera"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="tagsDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitTags">保存标签</el-button>
      </template>
    </el-dialog>

    <!-- 路由审批弹窗 -->
    <el-dialog v-model="routesDialogVisible" title="子网路由审批管理" width="500px">
      <div style="margin-bottom: 10px; font-size: 13px; color: #6b7280;">
        勾选允许该设备作为 Subnet Router 广播并打通的局域网 CIDR 网段：
      </div>
      <el-checkbox-group v-model="routesForm.selectedRoutes">
        <el-checkbox v-for="r in routesForm.allRoutes" :key="r" :label="r">{{ r }}</el-checkbox>
      </el-checkbox-group>
      <div style="margin-top: 15px;">
        <el-input v-model="routesForm.customRoute" placeholder="输入追加 CIDR (如 192.168.10.0/24)">
          <template #append>
            <el-button @click="addCustomRoute">添加</el-button>
          </template>
        </el-input>
      </div>
      <template #footer>
        <el-button @click="routesDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRoutes">确认生效路由</el-button>
      </template>
    </el-dialog>
  </div>

  <script>
    const { createApp, ref, reactive, computed, onMounted } = Vue;
    const app = createApp({
      setup() {
        const activeTab = ref('devices');
        const devices = ref([]);
        const loadingDevices = ref(false);
        const syncing = ref(false);
        const deviceSearch = ref('');

        const keyForm = reactive({ reusable: true, ephemeral: false, preauthorized: true, tagsStr: '', expiry_days: 90, purpose: '' });
        const creatingKey = ref(false);
        const newlyCreatedKey = ref(null);
        const keyLogs = ref([]);

        const aclContent = ref('');
        const loadingACL = ref(false);
        const savingACL = ref(false);

        const webhookLogs = ref([]);

        const configForm = reactive({ tailnet: '', api_key: '', client_id: '', client_secret: '', webhook_secret: '', proxy_url: '', auth_key: '', node_hostname: 'apeadmin-crm', tsnet_running: false });
        const savingConfig = ref(false);

        // SSH 弹窗状态
        const sshDialogVisible = ref(false);
        const sshForm = reactive({ target: '', port: 22, user: 'root', password: '', command: 'uptime' });
        const sshOutput = ref('');
        const runningSSH = ref(false);

        // 弹窗状态
        const diagDialogVisible = ref(false);
        const diagForm = reactive({ target: '', port: 80, preset: 'HTTP (80)' });
        const diagResult = ref(null);
        const runningDiag = ref(false);

        const renameDialogVisible = ref(false);
        const renameForm = reactive({ device_id: '', name: '' });

        const tagsDialogVisible = ref(false);
        const tagsForm = reactive({ device_id: '', tagsStr: '' });

        const routesDialogVisible = ref(false);
        const routesForm = reactive({ device_id: '', allRoutes: [], selectedRoutes: [], customRoute: '' });

        const onlineCount = computed(() => devices.value.filter(d => d.online).length);
        const expiryDisabledCount = computed(() => devices.value.filter(d => d.key_expiry_disabled).length);

        const getAuthHeader = () => {
          const urlParams = new URLSearchParams(window.location.search);
          const token = urlParams.get('token') || localStorage.getItem('apeadmin_token') || localStorage.getItem('token') || '';
          if (urlParams.get('token')) {
            try { localStorage.setItem('apeadmin_token', urlParams.get('token')); } catch(e) {}
          }
          return token ? { 'Authorization': 'Bearer ' + token } : {};
        };

        const loadDevices = async () => {
          loadingDevices.value = true;
          try {
            const res = await fetch('/api/v1/tailscale/devices?keyword=' + encodeURIComponent(deviceSearch.value), { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) devices.value = json.data || [];
          } catch(e) {}
          loadingDevices.value = false;
        };

        const syncDevices = async () => {
          syncing.value = true;
          try {
            const res = await fetch('/api/v1/tailscale/sync/devices', { method: 'POST', headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('成功同步 ' + (json.data?.synced_count || 0) + ' 个节点');
              await loadDevices();
            } else {
              ElementPlus.ElMessage.error(json.msg || '同步失败');
            }
          } catch(e) {}
          syncing.value = false;
        };

        const deleteDevice = async (deviceID) => {
          try {
            const res = await fetch('/api/v1/tailscale/devices/' + deviceID, { method: 'DELETE', headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('设备已解绑下线');
              await loadDevices();
            } else {
              ElementPlus.ElMessage.error(json.msg || '解绑失败');
            }
          } catch(e) {}
        };

        const toggleKeyExpiry = async (device, disabled) => {
          try {
            const res = await fetch('/api/v1/tailscale/devices/' + device.device_id + '/key-expiry', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ disabled: disabled })
            });
            const json = await res.json();
            if (json.code === 200) {
              device.key_expiry_disabled = disabled;
              ElementPlus.ElMessage.success(disabled ? '已开启免密钥过期 (IoT 永不掉线)' : '已恢复定期过期');
            } else {
              ElementPlus.ElMessage.error(json.msg || '更新失败');
            }
          } catch(e) {}
        };

        const openRename = (device) => {
          renameForm.device_id = device.device_id;
          renameForm.name = device.name || device.hostname;
          renameDialogVisible.value = true;
        };

        const submitRename = async () => {
          try {
            const res = await fetch('/api/v1/tailscale/devices/' + renameForm.device_id + '/name', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ name: renameForm.name })
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('设备重命名成功');
              renameDialogVisible.value = false;
              loadDevices();
            } else {
              ElementPlus.ElMessage.error(json.msg || '修改失败');
            }
          } catch(e) {}
        };

        const openTags = (device) => {
          tagsForm.device_id = device.device_id;
          const tags = parseJSON(device.tags);
          tagsForm.tagsStr = tags.join(',');
          tagsDialogVisible.value = true;
        };

        const submitTags = async () => {
          try {
            const tags = tagsForm.tagsStr.split(',').map(t => t.trim()).filter(Boolean);
            const res = await fetch('/api/v1/tailscale/devices/' + tagsForm.device_id + '/tags', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ tags: tags })
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('设备标签更新成功');
              tagsDialogVisible.value = false;
              loadDevices();
            } else {
              ElementPlus.ElMessage.error(json.msg || '更新标签失败');
            }
          } catch(e) {}
        };

        const openRoutes = async (device) => {
          routesForm.device_id = device.device_id;
          routesForm.allRoutes = [];
          routesForm.selectedRoutes = [];
          try {
            const res = await fetch('/api/v1/tailscale/devices/' + device.device_id + '/routes', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200 && json.data) {
              routesForm.allRoutes = json.data.advertisedRoutes || [];
              routesForm.selectedRoutes = json.data.enabledRoutes || [];
            }
          } catch(e) {}
          routesDialogVisible.value = true;
        };

        const addCustomRoute = () => {
          if (routesForm.customRoute && !routesForm.allRoutes.includes(routesForm.customRoute)) {
            routesForm.allRoutes.push(routesForm.customRoute);
            routesForm.selectedRoutes.push(routesForm.customRoute);
            routesForm.customRoute = '';
          }
        };

        const submitRoutes = async () => {
          try {
            const res = await fetch('/api/v1/tailscale/devices/' + routesForm.device_id + '/routes', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ routes: routesForm.selectedRoutes })
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('子网路由审批已生效');
              routesDialogVisible.value = false;
              loadDevices();
            } else {
              ElementPlus.ElMessage.error(json.msg || '审批路由失败');
            }
          } catch(e) {}
        };

        const openSSH = (target) => {
          sshForm.target = target || '';
          sshOutput.value = '';
          sshDialogVisible.value = true;
        };

        const execSSH = async () => {
          if (!sshForm.target || !sshForm.command) {
            ElementPlus.ElMessage.warning('请输入目标主机与执行命令');
            return;
          }
          runningSSH.value = true;
          sshOutput.value = '';
          try {
            const res = await fetch('/api/v1/tailscale/ssh/exec', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify(sshForm)
            });
            const json = await res.json();
            if (json.code === 200 && json.data) {
              sshOutput.value = json.data.output || (json.data.error ? '错误: ' + json.data.error : '(无返回内容)');
              if (json.data.status === 'success') {
                ElementPlus.ElMessage.success('命令执行完成');
              } else {
                ElementPlus.ElMessage.warning('执行完成但有异常');
              }
            } else {
              sshOutput.value = json.msg || '执行失败';
              ElementPlus.ElMessage.error(json.msg || '执行失败');
            }
          } catch(e) {
            sshOutput.value = '网络请求异常: ' + e.message;
          }
          runningSSH.value = false;
        };

        const openQuickDiag = (target) => {
          diagForm.target = target || '';
          diagResult.value = null;
          diagDialogVisible.value = true;
        };

        const handlePresetChange = (preset) => {
          if (preset.includes('502')) diagForm.port = 502;
          else if (preset.includes('554')) diagForm.port = 554;
          else if (preset.includes('1883')) diagForm.port = 1883;
          else if (preset.includes('80')) diagForm.port = 80;
          else if (preset.includes('22')) diagForm.port = 22;
        };

        const runDiagnose = async () => {
          if (!diagForm.target) {
            ElementPlus.ElMessage.warning('请输入目标地址');
            return;
          }
          runningDiag.value = true;
          diagResult.value = null;
          try {
            const res = await fetch('/api/v1/tailscale/diagnose', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ target: diagForm.target, port: diagForm.port, protocol: diagForm.preset })
            });
            const json = await res.json();
            if (json.code === 200) {
              diagResult.value = json.data;
            } else {
              ElementPlus.ElMessage.error(json.msg || '诊断执行失败');
            }
          } catch(e) {}
          runningDiag.value = false;
        };

        const createAuthKey = async () => {
          creatingKey.value = true;
          try {
            const tags = keyForm.tagsStr ? keyForm.tagsStr.split(',').map(t => t.trim()).filter(Boolean) : [];
            const reqData = {
              reusable: keyForm.reusable,
              ephemeral: keyForm.ephemeral,
              preauthorized: keyForm.preauthorized,
              tags: tags,
              expiry_days: keyForm.expiry_days,
              purpose: keyForm.purpose || '组网节点接入'
            };
            const res = await fetch('/api/v1/tailscale/keys', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify(reqData)
            });
            const json = await res.json();
            if (json.code === 200) {
              newlyCreatedKey.value = json.data;
              ElementPlus.ElMessage.success('Auth Key 生成成功');
              await loadKeyLogs();
            } else {
              ElementPlus.ElMessage.error(json.msg || '生成 Key 失败');
            }
          } catch(e) {}
          creatingKey.value = false;
        };

        const loadKeyLogs = async () => {
          try {
            const res = await fetch('/api/v1/tailscale/keys/logs', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) keyLogs.value = json.data || [];
          } catch(e) {}
        };

        const revokeAuthKey = async (keyID) => {
          try {
            const res = await fetch('/api/v1/tailscale/keys/' + keyID, { method: 'DELETE', headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('密钥已撤销');
              loadKeyLogs();
            } else {
              ElementPlus.ElMessage.error(json.msg || '撤销失败');
            }
          } catch(e) {}
        };

        const loadACL = async () => {
          loadingACL.value = true;
          try {
            const res = await fetch('/api/v1/tailscale/acl', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) aclContent.value = json.data || '';
          } catch(e) {}
          loadingACL.value = false;
        };

        const validateACL = async () => {
          try {
            const res = await fetch('/api/v1/tailscale/acl/validate', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ hujson: aclContent.value })
            });
            const json = await res.json();
            if (json.code === 200) ElementPlus.ElMessage.success('ACL 语法校验通过！');
            else ElementPlus.ElMessage.error(json.msg || '校验失败');
          } catch(e) {}
        };

        const saveACL = async () => {
          savingACL.value = true;
          try {
            const res = await fetch('/api/v1/tailscale/acl', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ hujson: aclContent.value })
            });
            const json = await res.json();
            if (json.code === 200) ElementPlus.ElMessage.success('ACL 策略已保存并成功生效！');
            else ElementPlus.ElMessage.error(json.msg || '保存失败');
          } catch(e) {}
          savingACL.value = false;
        };

        const applyTemplate = (type) => {
          if (type === 'iot') {
            aclContent.value = '{\\n  // Tailscale 零信任万物互联微隔离策略 (HuJSON)\\n  \"tagOwners\": {\\n    \"tag:crm-server\": [\"autogroup:admin\"],\\n    \"tag:iot-gateway\": [\"autogroup:admin\"],\\n    \"tag:iot-door\": [\"autogroup:admin\"],\\n    \"tag:camera\": [\"autogroup:admin\"]\\n  },\\n  \"acls\": [\\n    // 1. 允许 CRM 服务端访问所有边缘物联节点及摄像头 RTSP/Web\\n    { \"action\": \"accept\", \"src\": [\"tag:crm-server\", \"autogroup:admin\"], \"dst\": [\"*:*\"] },\\n    // 2. 允许边缘物联网设备向 CRM 后端上报数据 (8001/MQTT)\\n    { \"action\": \"accept\", \"src\": [\"tag:iot-gateway\", \"tag:iot-door\"], \"dst\": [\"tag:crm-server:8001,1883\"] }\\n  ]\\n}';
          } else {
            aclContent.value = '{\\n  // 全网互通开发策略\\n  \"acls\": [\\n    { \"action\": \"accept\", \"src\": [\"*\"], \"dst\": [\"*:*\"] }\\n  ]\\n}';
          }
        };

        const loadWebhookLogs = async () => {
          try {
            const res = await fetch('/api/v1/tailscale/webhook/logs', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) webhookLogs.value = json.data || [];
          } catch(e) {}
        };

        const loadConfig = async () => {
          try {
            const res = await fetch('/api/v1/tailscale/config', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200 && json.data) {
              Object.assign(configForm, json.data);
            }
          } catch(e) {}
        };

        const saveConfig = async () => {
          savingConfig.value = true;
          try {
            const res = await fetch('/api/v1/tailscale/config', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify(configForm)
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('配置保存成功');
            } else {
              ElementPlus.ElMessage.error(json.msg || '保存失败');
            }
          } catch(e) {}
          savingConfig.value = false;
        };

        const parseJSON = (str) => {
          if (!str) return [];
          try { return JSON.parse(str) || []; } catch(e) { return [str]; }
        };

        const parsePrimaryIP = (ipsStr) => {
          const ips = parseJSON(ipsStr);
          return ips.length > 0 ? ips[0] : '';
        };

        const parseSubnetRoutes = (routesStr) => {
          if (!routesStr) return [];
          try {
            const obj = JSON.parse(routesStr);
            return obj.enabledRoutes || obj.advertisedRoutes || [];
          } catch(e) { return []; }
        };

        const copyText = (txt) => {
          navigator.clipboard.writeText(txt).then(() => {
            ElementPlus.ElMessage.success('已复制: ' + txt);
          });
        };

        const handleTabChange = (tabName) => {
          if (tabName === 'devices') loadDevices();
          if (tabName === 'keys') loadKeyLogs();
          if (tabName === 'acl') loadACL();
          if (tabName === 'webhook') loadWebhookLogs();
          if (tabName === 'config') loadConfig();
        };

        onMounted(() => {
          loadConfig();
          loadDevices();
        });

        return {
          activeTab, devices, loadingDevices, syncing, deviceSearch, onlineCount, expiryDisabledCount,
          keyForm, creatingKey, newlyCreatedKey, keyLogs, createAuthKey, revokeAuthKey,
          aclContent, loadingACL, savingACL, loadACL, validateACL, saveACL, applyTemplate,
          webhookLogs, loadWebhookLogs,
          configForm, savingConfig, saveConfig,
          sshDialogVisible, sshForm, sshOutput, runningSSH, openSSH, execSSH,
          diagDialogVisible, diagForm, diagResult, runningDiag, openQuickDiag, handlePresetChange, runDiagnose,
          renameDialogVisible, renameForm, openRename, submitRename,
          tagsDialogVisible, tagsForm, openTags, submitTags,
          routesDialogVisible, routesForm, openRoutes, addCustomRoute, submitRoutes,
          loadDevices, syncDevices, deleteDevice, toggleKeyExpiry, parseJSON, parsePrimaryIP, parseSubnetRoutes, copyText, handleTabChange
        };
      }
    });

    for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
      app.component(key, component);
    }
    app.use(ElementPlus);
    app.mount('#app');
  </script>
</body>
</html>`
