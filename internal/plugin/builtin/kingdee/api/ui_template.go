package api

const KingdeeUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>K3 Cloud</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.css" />
  <script src="https://cdn.jsdelivr.net/npm/vue@3.4.27/dist/vue.global.prod.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.full.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/@element-plus/icons-vue@2.3.1/dist/index.iife.min.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f7fa; margin: 0; padding: 20px; color: #303133; }
    .header { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); margin-bottom: 20px; display: flex; justify-content: space-between; align-items: center; }
    .header h2 { margin: 0; font-size: 20px; color: #1f2937; display: flex; align-items: center; gap: 10px; }
    .card-box { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); }
    .filter-bar { display: flex; gap: 12px; margin-bottom: 15px; align-items: center; flex-wrap: wrap; }
    .stat-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 16px; margin-bottom: 20px; }
    .stat-card { background: #fff; padding: 16px 20px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; display: flex; align-items: center; justify-content: space-between; }
    .stat-title { font-size: 13px; color: #6b7280; margin-bottom: 4px; }
    .stat-value { font-size: 18px; font-weight: 700; color: #111827; }
  </style>
</head>
<body>
  <div id="app">
    <div class="header">
      <h2>
        <el-icon color="#409EFF"><box /></el-icon>
        K3 Cloud
      </h2>
      <el-tag type="primary" effect="dark" round>Tailscale 内网通道已就绪</el-tag>
    </div>

    <!-- 指标卡片 -->
    <div class="stat-row">
      <div class="stat-card">
        <div>
          <div class="stat-title">金蝶服务器</div>
          <div class="stat-value" style="font-size: 14px; word-break: break-all;">{{ configForm.server_url || '未配置' }}</div>
        </div>
        <el-icon size="32" color="#409EFF"><connection /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">数据中心 (账套 ID)</div>
          <div class="stat-value" style="font-size: 15px; color: #67C23A;">{{ configForm.db_id || '未配置' }}</div>
        </div>
        <el-icon size="32" color="#67C23A"><data-analysis /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">集成登录账号</div>
          <div class="stat-value" style="font-size: 15px; color: #E6A23C;">{{ configForm.username || '未配置' }}</div>
        </div>
        <el-icon size="32" color="#E6A23C"><user /></el-icon>
      </div>
    </div>

    <div class="card-box">
      <el-tabs v-model="activeTab">
        <!-- 标签页 1：通用单据 SQL 条件查询 -->
        <el-tab-pane label="通用单据查询 (ExecuteBillQuery)" name="customQuery">
          <el-form :model="queryForm" label-width="140px" style="max-width: 800px; margin-top: 10px;">
            <el-form-item label="单据 FormId" required>
              <el-input v-model="queryForm.form_id" placeholder="如 BD_MATERIAL, SAL_SALEORDER, AR_RECEIVEBILL"></el-input>
            </el-form-item>
            <el-form-item label="查询字段 FieldKeys" required>
              <el-input v-model="queryForm.field_keys" placeholder="逗号分隔，如 FMaterialId,FNumber,FName"></el-input>
            </el-form-item>
            <el-form-item label="过滤条件 FilterString">
              <el-input v-model="queryForm.filter_string" placeholder="如 FForbidStatus = 'A' AND FName LIKE '%测试%'"></el-input>
            </el-form-item>
            <el-form-item label="返回条数 Limit">
              <el-input-number v-model="queryForm.limit" :min="1" :max="500"></el-input-number>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="executingQuery" @click="executeCustomQuery">执行单据查询</el-button>
            </el-form-item>
          </el-form>

          <div v-if="customResults.length > 0" style="margin-top: 20px;">
            <h4>查询结果 (共 {{ customResults.length }} 条记录):</h4>
            <el-table :data="customResults" stripe border style="width: 100%;">
              <el-table-column v-for="(col, index) in customResultCols" :key="index" :label="'列 ' + (index + 1)">
                <template #default="scope">
                  {{ scope.row[index] }}
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 标签页 2：金蝶账套配置 -->
        <el-tab-pane label="金蝶账套 API 配置" name="config">
          <el-form :model="configForm" label-width="160px" style="max-width: 650px; margin-top: 15px;">
            <el-form-item label="金蝶服务地址" required>
              <el-input v-model="configForm.server_url" placeholder="内网地址如 http://192.168.1.50:8000/k3cloud/"></el-input>
            </el-form-item>
            <el-form-item label="账套 ID (数据中心)" required>
              <el-input v-model="configForm.db_id" placeholder="如 60d1xxxxxx 或 db2024"></el-input>
            </el-form-item>
            <el-form-item label="登录用户" required>
              <el-input v-model="configForm.username" placeholder="金蝶集成专用账号或管理员"></el-input>
            </el-form-item>
            <el-form-item label="应用 ID (AppID)">
              <el-input v-model="configForm.app_id" placeholder="Web API 注册分配的 AppID"></el-input>
            </el-form-item>
            <el-form-item label="应用密钥 (AppSecret)">
              <el-input v-model="configForm.app_secret" type="password" show-password placeholder="应用 Secret（优先使用）"></el-input>
            </el-form-item>
            <el-form-item label="登录密码 (Password)">
              <el-input v-model="configForm.password" type="password" show-password placeholder="未填 AppSecret 时使用账号密码"></el-input>
            </el-form-item>
            <el-form-item label="语言代码 (Lcid)">
              <el-input-number v-model="configForm.lcid" :min="1000" :max="9999" placeholder="2052"></el-input-number>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingConfig" @click="saveConfig">保存配置</el-button>
              <el-button type="success" :loading="testingConfig" @click="testConfig">测试 API 连通性</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>

  <script>
    const { createApp, ref, onMounted } = Vue;
    const { ElMessage } = ElementPlus;

    createApp({
      setup() {
        const activeTab = ref('customQuery');

        const queryForm = ref({
          form_id: 'BD_MATERIAL',
          field_keys: 'FMaterialId,FNumber,FName,FSpecification',
          filter_string: "FForbidStatus = 'A'",
          limit: 20
        });
        const executingQuery = ref(false);
        const customResults = ref([]);
        const customResultCols = ref(0);

        const configForm = ref({
          server_url: '',
          db_id: '',
          username: '',
          app_id: '',
          app_secret: '',
          password: '',
          lcid: 2052
        });
        const savingConfig = ref(false);
        const testingConfig = ref(false);

        const getToken = () => {
          const urlParams = new URLSearchParams(window.location.search);
          let token = urlParams.get('token');
          if (token) {
            localStorage.setItem('apeadmin_token', token);
          } else {
            token = localStorage.getItem('apeadmin_token');
          }
          return token || '';
        };

        const getAuthHeader = () => {
          const token = getToken();
          return token ? { 'Authorization': 'Bearer ' + token } : {};
        };

        const loadConfig = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/config', { headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200 && data.data) {
              configForm.value = data.data;
            }
          } catch (e) {
            console.error('加载金蝶配置失败', e);
          }
        };

        const saveConfig = async () => {
          savingConfig.value = true;
          try {
            const res = await fetch('/api/v1/kingdee/config', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...getAuthHeader() },
              body: JSON.stringify(configForm.value)
            });
            const data = await res.json();
            if (data.code === 200) {
              ElMessage.success('金蝶云星空配置保存成功');
            } else {
              ElMessage.error(data.msg || '保存失败');
            }
          } catch (e) {
            ElMessage.error('请求错误: ' + e.message);
          } finally {
            savingConfig.value = false;
          }
        };

        const testConfig = async () => {
          testingConfig.value = true;
          try {
            const res = await fetch('/api/v1/kingdee/test', {
              method: 'POST',
              headers: getAuthHeader()
            });
            const data = await res.json();
            if (data.code === 200 && data.data.ok) {
              ElMessage.success(data.data.message);
            } else {
              ElMessage.error((data.data && data.data.error) || data.msg || '测试连通失败');
            }
          } catch (e) {
            ElMessage.error('网络请求失败: ' + e.message);
          } finally {
            testingConfig.value = false;
          }
        };

        const executeCustomQuery = async () => {
          executingQuery.value = true;
          customResults.value = [];
          try {
            const res = await fetch('/api/v1/kingdee/query', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...getAuthHeader() },
              body: JSON.stringify(queryForm.value)
            });
            const data = await res.json();
            if (data.code === 200 && Array.isArray(data.data)) {
              customResults.value = data.data;
              if (data.data.length > 0 && Array.isArray(data.data[0])) {
                customResultCols.value = data.data[0].length;
              }
              ElMessage.success('执行通用查询成功，返回 ' + data.data.length + ' 条数据');
            } else {
              ElMessage.error(data.msg || '执行失败');
            }
          } catch (e) {
            ElMessage.error('查询错误: ' + e.message);
          } finally {
            executingQuery.value = false;
          }
        };

        onMounted(() => {
          loadConfig();
        });

        return {
          activeTab,
          queryForm, executingQuery, customResults, customResultCols, executeCustomQuery,
          configForm, savingConfig, testingConfig, loadConfig, saveConfig, testConfig
        };
      }
    }).use(ElementPlus).mount('#app');
  </script>
</body>
</html>`
