package api

const KingdeeUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>金蝶云星空 ERP 业务控制台</title>
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
    .stat-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 20px; }
    .stat-card { background: #fff; padding: 16px 20px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; display: flex; align-items: center; justify-content: space-between; }
    .stat-title { font-size: 13px; color: #6b7280; margin-bottom: 4px; }
    .stat-value { font-size: 22px; font-weight: 700; color: #111827; }
  </style>
</head>
<body>
  <div id="app">
    <div class="header">
      <h2>
        <el-icon color="#409EFF"><box /></el-icon>
        金蝶云星空 ERP 业务对接控制台 (K3 Cloud)
      </h2>
      <el-tag type="primary" effect="dark" round>Tailscale 内网子网直连中</el-tag>
    </div>

    <!-- 指标卡片 -->
    <div class="stat-row">
      <div class="stat-card">
        <div>
          <div class="stat-title">物料/商品查询数</div>
          <div class="stat-value" style="color: #409EFF;">{{ materials.length }}</div>
        </div>
        <el-icon size="32" color="#409EFF"><goods /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">客户总数</div>
          <div class="stat-value" style="color: #67C23A;">{{ customers.length }}</div>
        </div>
        <el-icon size="32" color="#67C23A"><user /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">销售订单加载数</div>
          <div class="stat-value" style="color: #E6A23C;">{{ salesOrders.length }}</div>
        </div>
        <el-icon size="32" color="#E6A23C"><document /></el-icon>
      </div>
      <div class="stat-card">
        <div>
          <div class="stat-title">金蝶服务器</div>
          <div class="stat-value" style="font-size: 14px; word-break: break-all;">{{ configForm.server_url || '未配置' }}</div>
        </div>
        <el-icon size="32" color="#909399"><connection /></el-icon>
      </div>
    </div>

    <div class="card-box">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 标签页 1：物料/商品查询 -->
        <el-tab-pane label="物料/商品列表 (BD_MATERIAL)" name="materials">
          <div class="filter-bar">
            <el-input v-model="materialSearch" placeholder="搜索物料编码/名称/规格" style="width: 280px;" clearable @keyup.enter="loadMaterials"></el-input>
            <el-button type="primary" :loading="loadingMaterials" @click="loadMaterials">
              <el-icon><search /></el-icon> 查询物料
            </el-button>
          </div>
          <el-table :data="materials" v-loading="loadingMaterials" stripe border style="width: 100%;">
            <el-table-column prop="id" label="物料内码 (ID)" width="120"></el-table-column>
            <el-table-column prop="number" label="物料编码" width="180"></el-table-column>
            <el-table-column prop="name" label="物料名称" min-width="200"></el-table-column>
            <el-table-column prop="specification" label="规格型号" min-width="200"></el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 标签页 2：客户列表 -->
        <el-tab-pane label="客户列表 (BD_Customer)" name="customers">
          <div class="filter-bar">
            <el-input v-model="customerSearch" placeholder="搜索客户编码/名称" style="width: 280px;" clearable @keyup.enter="loadCustomers"></el-input>
            <el-button type="success" :loading="loadingCustomers" @click="loadCustomers">
              <el-icon><search /></el-icon> 查询客户
            </el-button>
          </div>
          <el-table :data="customers" v-loading="loadingCustomers" stripe border style="width: 100%;">
            <el-table-column prop="id" label="客户 ID" width="120"></el-table-column>
            <el-table-column prop="number" label="客户编码" width="180"></el-table-column>
            <el-table-column prop="name" label="客户名称" min-width="240"></el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 标签页 3：销售订单 -->
        <el-tab-pane label="销售订单 (SAL_SALEORDER)" name="salesOrders">
          <div class="filter-bar">
            <el-input v-model="orderSearch" placeholder="搜索订单号/客户名称" style="width: 280px;" clearable @keyup.enter="loadSalesOrders"></el-input>
            <el-button type="warning" :loading="loadingOrders" @click="loadSalesOrders">
              <el-icon><search /></el-icon> 查询销售订单
            </el-button>
          </div>
          <el-table :data="salesOrders" v-loading="loadingOrders" stripe border style="width: 100%;">
            <el-table-column prop="id" label="订单 ID" width="100"></el-table-column>
            <el-table-column prop="bill_no" label="单据编号" width="200"></el-table-column>
            <el-table-column prop="date" label="订货日期" width="160"></el-table-column>
            <el-table-column prop="customer_name" label="客户名称" min-width="200"></el-table-column>
            <el-table-column prop="document_status" label="审核状态" width="120">
              <template #default="scope">
                <el-tag :type="scope.row.document_status === 'C' ? 'success' : 'info'">
                  {{ scope.row.document_status === 'C' ? '已审核' : scope.row.document_status || '暂存' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 标签页 4：自定义单据 SQL 条件查询 -->
        <el-tab-pane label="通用单据查询 (ExecuteBillQuery)" name="customQuery">
          <el-form :model="queryForm" label-width="120px" style="max-width: 800px; margin-top: 10px;">
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

        <!-- 标签页 5：金蝶账套配置 -->
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
              <el-button type="success" :loading="testingConfig" @click="testConfig">测试内网 API 连通性</el-button>
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
        const activeTab = ref('materials');

        const materialSearch = ref('');
        const loadingMaterials = ref(false);
        const materials = ref([]);

        const customerSearch = ref('');
        const loadingCustomers = ref(false);
        const customers = ref([]);

        const orderSearch = ref('');
        const loadingOrders = ref(false);
        const salesOrders = ref([]);

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

        const loadMaterials = async () => {
          loadingMaterials.value = true;
          try {
            const url = '/api/v1/kingdee/materials?keyword=' + encodeURIComponent(materialSearch.value);
            const res = await fetch(url, { headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200) {
              materials.value = data.data || [];
            } else {
              ElMessage.error(data.msg || '查询物料失败');
            }
          } catch (e) {
            ElMessage.error('查询报错: ' + e.message);
          } finally {
            loadingMaterials.value = false;
          }
        };

        const loadCustomers = async () => {
          loadingCustomers.value = true;
          try {
            const url = '/api/v1/kingdee/customers?keyword=' + encodeURIComponent(customerSearch.value);
            const res = await fetch(url, { headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200) {
              customers.value = data.data || [];
            } else {
              ElMessage.error(data.msg || '查询客户失败');
            }
          } catch (e) {
            ElMessage.error('查询报错: ' + e.message);
          } finally {
            loadingCustomers.value = false;
          }
        };

        const loadSalesOrders = async () => {
          loadingOrders.value = true;
          try {
            const url = '/api/v1/kingdee/sales-orders?keyword=' + encodeURIComponent(orderSearch.value);
            const res = await fetch(url, { headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200) {
              salesOrders.value = data.data || [];
            } else {
              ElMessage.error(data.msg || '查询销售订单失败');
            }
          } catch (e) {
            ElMessage.error('查询报错: ' + e.message);
          } finally {
            loadingOrders.value = false;
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

        const handleTabChange = (tabName) => {
          if (tabName === 'materials' && materials.value.length === 0) loadMaterials();
          if (tabName === 'customers' && customers.value.length === 0) loadCustomers();
          if (tabName === 'salesOrders' && salesOrders.value.length === 0) loadSalesOrders();
        };

        onMounted(() => {
          loadConfig();
          loadMaterials();
        });

        return {
          activeTab,
          materialSearch, loadingMaterials, materials, loadMaterials,
          customerSearch, loadingCustomers, customers, loadCustomers,
          orderSearch, loadingOrders, salesOrders, loadSalesOrders,
          queryForm, executingQuery, customResults, customResultCols, executeCustomQuery,
          configForm, savingConfig, testingConfig, loadConfig, saveConfig, testConfig,
          handleTabChange
        };
      }
    }).use(ElementPlus).mount('#app');
  </script>
</body>
</html>`
