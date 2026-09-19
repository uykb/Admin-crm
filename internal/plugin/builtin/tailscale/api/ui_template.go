package api

const TailscaleUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Tailscale 虚拟组网与设备管控中心</title>
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
    .device-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; margin-top: 15px; }
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
    .code-preview { background: #282c34; color: #abb2bf; padding: 16px; border-radius: 6px; font-family: monospace; white-space: pre-wrap; word-break: break-all; max-height: 400px; overflow-y: auto; }
  </style>
</head>
<body>
  <div id="app">
    <div class="header">
      <h2>
        <el-icon color="#5A67F5"><connection /></el-icon>
        Tailscale 虚拟组网与内网设备管控中心
      </h2>
      <el-tag type="success" effect="dark" round>网络全打通 运行中</el-tag>
    </div>

    <!-- 概览指标卡 -->
    <div class="stat-row">
      <div class="stat-card">
        <div>
          <div class="stat-title">接入设备总数</div>
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
          <div class="stat-title">离线节点</div>
          <div class="stat-value" style="color: #9ca3af;">{{ devices.length - onlineCount }}</div>
        </div>
        <el-icon size="32" color="#9ca3af"><warning /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">所属 Tailnet</div>
          <div class="stat-value" style="font-size: 16px; word-break: break-all;">{{ configForm.tailnet || '未配置' }}</div>
        </div>
        <el-icon size="32" color="#8b5cf6"><share /></el-icon>
      </div>
    </div>

    <div class="card-box">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 标签页 1：设备与节点管理 -->
        <el-tab-pane label="组网节点列表" name="devices">
          <div class="filter-bar">
            <el-input v-model="deviceSearch" placeholder="搜索设备名/主机名/IP" style="width: 260px;" clearable @keyup.enter="loadDevices"></el-input>
            <el-button type="primary" @click="loadDevices">查询节点</el-button>
            <el-button type="success" :loading="syncing" @click="syncDevices">
              <el-icon><refresh /></el-icon> 同步 Tailnet 节点
            </el-button>
          </div>

          <div class="device-grid" v-loading="loadingDevices">
            <div v-for="d in devices" :key="d.device_id" class="device-card">
              <div class="device-header">
                <div>
                  <div class="device-name">{{ d.name || d.hostname }}</div>
                  <div class="device-user">所有者: {{ d.user || '全网共享' }} | 系统: {{ d.os }} (v{{ d.client_version }})</div>
                </div>
                <el-tag :type="d.online ? 'success' : 'info'" size="small">
                  <span class="status-dot" :class="d.online ? 'status-online' : 'status-offline'"></span>
                  {{ d.online ? '在线' : '离线' }}
                </el-tag>
              </div>

              <div style="margin-top: 10px;">
                <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px;">Tailscale 内网 IP (点击可复制):</div>
                <div>
                  <el-tag v-for="ip in parseIPs(d.ips)" :key="ip" class="ip-tag" size="small" type="primary" effect="plain" @click="copyText(ip)">
                    {{ ip }}
                  </el-tag>
                </div>
              </div>

              <div style="margin-top: 12px; display: flex; justify-content: space-between; align-items: center;">
                <span style="font-size: 12px; color: #9ca3af;">最后活跃: {{ formatTime(d.last_seen) }}</span>
                <el-popconfirm title="确定下线并注销该设备？" @confirm="deleteDevice(d.device_id)">
                  <template #reference>
                    <el-button type="danger" link size="small">解绑注销</el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 标签页 2：Auth Key 接入密钥生成 -->
        <el-tab-pane label="Auth Key 接入密钥" name="keys">
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
                  <el-input v-model="keyForm.tagsStr" placeholder="如 tag:server,tag:prod (多标签逗号分隔)"></el-input>
                </el-form-item>
                <el-form-item label="有效天数">
                  <el-input-number v-model="keyForm.expiry_days" :min="1" :max="90"></el-input-number>
                </el-form-item>
                <el-form-item label="使用用途">
                  <el-input v-model="keyForm.purpose" placeholder="如: 生产服务器接入/测试机连通"></el-input>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" :loading="creatingKey" @click="createAuthKey">立即生成 Key</el-button>
                </el-form-item>
              </form>
            </el-col>

            <el-col :span="14">
              <div v-if="newlyCreatedKey">
                <el-alert title="新 Auth Key 已成功生成！" type="success" :closable="false" show-icon></el-alert>
                <div class="key-box">
                  {{ newlyCreatedKey.key }}
                </div>
                <div style="margin-top: 10px;">
                  <el-button type="primary" size="small" @click="copyText(newlyCreatedKey.key)">复制密钥</el-button>
                  <el-button type="success" size="small" @click="copyText('tailscale up --authkey=' + newlyCreatedKey.key)">复制命令行命令 (tailscale up --authkey=...)</el-button>
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
              </el-table>
            </el-col>
          </row>
        </el-tab-pane>

        <!-- 标签页 3：ACL 安全策略查看 -->
        <el-tab-pane label="ACL 访问控制策略" name="acl">
          <div class="filter-bar">
            <el-button type="primary" @click="loadACL">刷新 ACL 策略</el-button>
            <el-text type="info">查看当前 Tailnet 组网使用的 HuJSON 安全访问策略文件</el-text>
          </div>
          <div class="code-preview" v-loading="loadingACL">{{ aclContent || '未加载 ACL 数据' }}</div>
        </el-tab-pane>

        <!-- 标签页 4：Tailscale API 密钥配置 -->
        <el-tab-pane label="Tailscale 密钥配置" name="config">
          <el-form :model="configForm" label-width="140px" style="max-width: 600px; margin-top: 10px;">
            <el-form-item label="Tailnet 名称">
              <el-input v-model="configForm.tailnet" placeholder="如 example.com 或 your-email@gmail.com"></el-input>
            </el-form-item>
            <el-form-item label="Personal API Key">
              <el-input v-model="configForm.api_key" type="password" show-password placeholder="tskey-api-xxxx (优先使用)"></el-input>
            </el-form-item>
            <el-form-item label="OAuth Client ID">
              <el-input v-model="configForm.client_id" placeholder="填写 OAuth Client ID"></el-input>
            </el-form-item>
            <el-form-item label="OAuth Secret">
              <el-input v-model="configForm.client_secret" type="password" show-password placeholder="填写 OAuth Client Secret"></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingConfig" @click="saveConfig">保存 Tailscale 密钥配置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
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

        const keyForm = reactive({ reusable: true, ephemeral: false, preauthorized: true, tagsStr: '', expiry_days: 30, purpose: '' });
        const creatingKey = ref(false);
        const newlyCreatedKey = ref(null);
        const keyLogs = ref([]);

        const aclContent = ref('');
        const loadingACL = ref(false);

        const configForm = reactive({ tailnet: '', api_key: '', client_id: '', client_secret: '' });
        const savingConfig = ref(false);

        const onlineCount = computed(() => devices.value.filter(d => d.online).length);

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

        const loadACL = async () => {
          loadingACL.value = true;
          try {
            const res = await fetch('/api/v1/tailscale/acl', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) aclContent.value = json.data || '';
          } catch(e) {}
          loadingACL.value = false;
        };

        const loadConfig = async () => {
          try {
            const res = await fetch('/api/v1/tailscale/config', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200 && json.data) {
              configForm.tailnet = json.data.tailnet || '';
              configForm.api_key = json.data.api_key || '';
              configForm.client_id = json.data.client_id || '';
              configForm.client_secret = json.data.client_secret || '';
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
            if (json.code === 200) ElementPlus.ElMessage.success('Tailscale 密钥配置保存成功');
          } catch(e) {}
          savingConfig.value = false;
        };

        const parseIPs = (ipsStr) => {
          if (!ipsStr) return [];
          try { return JSON.parse(ipsStr); } catch(e) { return [ipsStr]; }
        };

        const formatTime = (tStr) => {
          if (!tStr) return '-';
          return tStr.replace('T', ' ').slice(0, 19);
        };

        const copyText = (txt) => {
          navigator.clipboard.writeText(txt).then(() => {
            ElementPlus.ElMessage.success('已复制到剪贴板');
          });
        };

        const handleTabChange = (tabName) => {
          if (tabName === 'devices') loadDevices();
          if (tabName === 'keys') loadKeyLogs();
          if (tabName === 'acl') loadACL();
          if (tabName === 'config') loadConfig();
        };

        onMounted(() => {
          loadConfig();
          loadDevices();
        });

        return {
          activeTab, devices, loadingDevices, syncing, deviceSearch, onlineCount,
          keyForm, creatingKey, newlyCreatedKey, keyLogs, createAuthKey,
          aclContent, loadingACL, loadACL,
          configForm, savingConfig, saveConfig,
          loadDevices, syncDevices, deleteDevice, parseIPs, formatTime, copyText, handleTabChange
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
