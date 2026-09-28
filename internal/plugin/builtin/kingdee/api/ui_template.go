package api

const KingdeeUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>K3 Cloud & 飞书审批控制台</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.css" />
  <script src="https://cdn.jsdelivr.net/npm/vue@3.4.27/dist/vue.global.prod.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.full.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/@element-plus/icons-vue@2.3.1/dist/index.iife.min.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f7fa; margin: 0; padding: 20px; color: #303133; }
    .header { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); margin-bottom: 20px; display: flex; justify-content: space-between; align-items: center; }
    .header h2 { margin: 0; font-size: 20px; color: #1f2937; display: flex; align-items: center; gap: 10px; }
    .card-box { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); }
    .stat-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 16px; margin-bottom: 20px; }
    .stat-card { background: #fff; padding: 16px 20px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.04); border: 1px solid #e5e7eb; display: flex; align-items: center; justify-content: space-between; }
    .stat-title { font-size: 13px; color: #6b7280; margin-bottom: 4px; }
    .stat-value { font-size: 18px; font-weight: 700; color: #111827; }
    .toolbar { display: flex; justify-content: space-between; margin-bottom: 15px; }
  </style>
</head>
<body>
  <div id="app">
    <div class="header">
      <h2>
        <el-icon color="#409EFF"><box /></el-icon>
        金蝶云星空 ERP & 飞书集成审批控制台
      </h2>
      <el-tag type="primary" effect="dark" round>Tailscale & 飞书通道已就绪</el-tag>
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
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        
        <!-- 标签页 1：飞书审批实例监控 -->
        <el-tab-pane label="飞书审批实例监控" name="instances">
          <div class="toolbar">
            <el-button type="primary" icon="Refresh" @click="loadInstances">刷新实例列表</el-button>
          </div>
          <el-table :data="instances" stripe border style="width: 100%;">
            <el-table-column prop="id" label="ID" width="70"></el-table-column>
            <el-table-column prop="bill_no" label="金蝶单据编号" width="180"></el-table-column>
            <el-table-column prop="instance_id" label="飞书 Instance Code" width="220"></el-table-column>
            <el-table-column prop="approve_status" label="审批状态" width="140">
              <template #default="scope">
                <el-tag :type="getStatusTagType(scope.row.approve_status)">{{ scope.row.approve_status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="retry_count" label="重试次数" width="90"></el-table-column>
            <el-table-column prop="error_msg" label="错误日志/原因" show-overflow-tooltip></el-table-column>
            <el-table-column prop="created_at" label="发起时间" width="170"></el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="scope">
                <el-button size="small" type="success" @click="retryAudit(scope.row.id)" :disabled="scope.row.approve_status === 'APPROVED'">重试审核</el-button>
                <el-button size="small" type="danger" @click="cancelInstance(scope.row.id)" :disabled="scope.row.approve_status === 'APPROVED' || scope.row.approve_status === 'CANCELED'">撤销</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 标签页 2：审批流程配置 -->
        <el-tab-pane label="审批流程配置 (KdFlow)" name="flows">
          <div class="toolbar">
            <el-button type="primary" icon="Plus" @click="openFlowDialog()">新建流程配置</el-button>
            <el-button icon="Refresh" @click="loadFlows">刷新</el-button>
          </div>
          <el-table :data="flows" stripe border style="width: 100%;">
            <el-table-column prop="id" label="ID" width="70"></el-table-column>
            <el-table-column prop="flow_name" label="流程名称" width="180"></el-table-column>
            <el-table-column prop="kingdee_form_id" label="金蝶 FormID" width="180"></el-table-column>
            <el-table-column prop="feishu_approval_code" label="飞书审批 Code" width="220"></el-table-column>
            <el-table-column prop="poll_filter_status" label="过滤状态" width="100"></el-table-column>
            <el-table-column prop="enabled" label="启用状态" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.enabled ? 'success' : 'info'">{{ scope.row.enabled ? '已启用' : '已禁用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="title_template" label="标题模板"></el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="scope">
                <el-button size="small" type="primary" @click="openFlowDialog(scope.row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 标签页 3：用户与主管映射 -->
        <el-tab-pane label="用户/主管映射 (KdUserMap)" name="usermaps">
          <div class="toolbar">
            <el-button type="primary" icon="Plus" @click="openUserMapDialog()">新建映射</el-button>
            <el-button icon="Refresh" @click="loadUserMaps">刷新</el-button>
          </div>
          <el-table :data="userMaps" stripe border style="width: 100%;">
            <el-table-column prop="id" label="ID" width="70"></el-table-column>
            <el-table-column prop="kd_user_id" label="金蝶用户ID (FCreatorId)" width="180"></el-table-column>
            <el-table-column prop="kd_user_name" label="金蝶用户名" width="150"></el-table-column>
            <el-table-column prop="feishu_open_id" label="申请人飞书 OpenID" width="220"></el-table-column>
            <el-table-column prop="approver_open_id" label="二级审批人 OpenID" width="220"></el-table-column>
            <el-table-column prop="dept_name" label="部门名称" width="140"></el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="scope">
                <el-button size="small" type="primary" @click="openUserMapDialog(scope.row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 标签页 4：通用单据查询 -->
        <el-tab-pane label="通用单据查询 (ExecuteBillQuery)" name="customQuery">
          <el-form :model="queryForm" label-width="140px" style="max-width: 800px; margin-top: 10px;">
            <el-form-item label="单据 FormId" required>
              <el-input v-model="queryForm.form_id" placeholder="如 BD_MATERIAL, SAL_SALEORDER, PUR_PurchaseOrder"></el-input>
            </el-form-item>
            <el-form-item label="查询字段 FieldKeys" required>
              <el-input v-model="queryForm.field_keys" placeholder="逗号分隔，如 FMaterialId,FNumber,FName"></el-input>
            </el-form-item>
            <el-form-item label="过滤条件 FilterString">
              <el-input v-model="queryForm.filter_string" placeholder="如 FDocumentStatus = 'B'"></el-input>
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

        <!-- 标签页 5：金蝶与飞书基础配置 -->
        <el-tab-pane label="金蝶/飞书基础配置" name="config">
          <el-form :model="configForm" label-width="180px" style="max-width: 750px; margin-top: 10px;">
            
            <el-divider content-position="left">
              <el-icon><box /></el-icon> <strong>金蝶云星空 ERP 连接配置</strong>
            </el-divider>

            <el-form-item label="金蝶服务器地址" required>
              <el-input v-model="configForm.server_url" placeholder="如 http://183.6.98.164:9080"></el-input>
            </el-form-item>
            <el-form-item label="账套 ID (DbID)" required>
              <el-input v-model="configForm.db_id" placeholder="金蝶账套数据中心 ID，如 6a62c504f0138e"></el-input>
            </el-form-item>
            <el-form-item label="登录用户名" required>
              <el-input v-model="configForm.username" placeholder="金蝶登录账号，如 Administrator"></el-input>
            </el-form-item>
            <el-form-item label="登录密码">
              <el-input v-model="configForm.password" type="password" show-password placeholder="金蝶用户登录密码（密码认证模式）"></el-input>
            </el-form-item>
            <el-form-item label="金蝶 App ID">
              <el-input v-model="configForm.kd_app_id" placeholder="金蝶 Web API 应用授权 AppID (如 352877_...)"></el-input>
            </el-form-item>
            <el-form-item label="金蝶 App Secret">
              <el-input v-model="configForm.kd_app_secret" type="password" show-password placeholder="金蝶 Web API 应用授权 AppSecret"></el-input>
            </el-form-item>
            <el-form-item label="语言 ID (LCID)">
              <el-input-number v-model="configForm.lcid" :min="1000"></el-input-number>
            </el-form-item>

            <el-divider content-position="left">
              <el-icon><chat-dot-round /></el-icon> <strong>飞书开放平台集成配置</strong>
            </el-divider>

            <el-form-item label="飞书 App ID (自建应用)">
              <el-input v-model="configForm.feishu_app_id" placeholder="飞书开发者后台应用 App ID (如 cli_a1b2c3d4e5...)"></el-input>
            </el-form-item>
            <el-form-item label="飞书 App Secret">
              <el-input v-model="configForm.feishu_app_secret" type="password" show-password placeholder="飞书开发者后台应用 App Secret"></el-input>
            </el-form-item>
            <el-form-item label="Webhook 签名/加密 Key">
              <el-input v-model="configForm.feishu_encrypt_key" type="password" show-password placeholder="飞书事件订阅 Encrypt Key / X-Lark-Signature 签名密钥 (可选)"></el-input>
            </el-form-item>
            <el-form-item label="默认兜底审批人 OpenID">
              <el-input v-model="configForm.default_approver_openid" placeholder="无法自动匹配审批人时的兜底 OpenID (如 ou_...)"></el-input>
            </el-form-item>

            <el-form-item style="margin-top: 20px;">
              <el-button type="primary" @click="saveConfig">保存全套配置</el-button>
              <el-button type="success" @click="testConnection">测试金蝶连通性</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

      </el-tabs>
    </div>

    <!-- 流程配置 Modal -->
    <el-dialog v-model="flowDialogVisible" title="编辑流程配置">
      <el-form :model="flowForm" label-width="150px">
        <el-form-item label="流程名称" required>
          <el-input v-model="flowForm.flow_name" placeholder="如 销售订单审批"></el-input>
        </el-form-item>
        <el-form-item label="金蝶 FormID" required>
          <el-input v-model="flowForm.kingdee_form_id" placeholder="如 SAL_SaleOrder"></el-input>
        </el-form-item>
        <el-form-item label="飞书审批 Code" required>
          <el-input v-model="flowForm.feishu_approval_code" placeholder="如 46DE81B8-..."></el-input>
        </el-form-item>
        <el-form-item label="单号字段名">
          <el-input v-model="flowForm.bill_no_field" placeholder="默认 FBillNo"></el-input>
        </el-form-item>
        <el-form-item label="状态字段名">
          <el-input v-model="flowForm.status_field" placeholder="默认 FDocumentStatus"></el-input>
        </el-form-item>
        <el-form-item label="待审批过滤状态">
          <el-input v-model="flowForm.poll_filter_status" placeholder="默认 B (审核中)"></el-input>
        </el-form-item>
        <el-form-item label="标题模板">
          <el-input v-model="flowForm.title_template" placeholder="如 销售订单 - {FBillNo}"></el-input>
        </el-form-item>
        <el-form-item label="是否启用">
          <el-switch v-model="flowForm.enabled"></el-switch>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="flowDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveFlow">保存</el-button>
      </template>
    </el-dialog>

    <!-- 用户映射 Modal -->
    <el-dialog v-model="userMapDialogVisible" title="编辑用户/主管映射">
      <el-form :model="userMapForm" label-width="170px">
        <el-form-item label="金蝶用户ID (FCreatorId)" required>
          <el-input v-model="userMapForm.kd_user_id"></el-input>
        </el-form-item>
        <el-form-item label="金蝶用户名">
          <el-input v-model="userMapForm.kd_user_name"></el-input>
        </el-form-item>
        <el-form-item label="申请人飞书 OpenID">
          <el-input v-model="userMapForm.feishu_open_id" placeholder="ou_..."></el-input>
        </el-form-item>
        <el-form-item label="二级审批人 OpenID">
          <el-input v-model="userMapForm.approver_open_id" placeholder="ou_..."></el-input>
        </el-form-item>
        <el-form-item label="部门名称">
          <el-input v-model="userMapForm.dept_name" placeholder="如 销售一部"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userMapDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveUserMap">保存</el-button>
      </template>
    </el-dialog>

  </div>

  <script>
    const { createApp, ref, onMounted } = Vue;
    const { ElMessage } = ElementPlus;

    createApp({
      setup() {
        const activeTab = ref('instances');

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
          password: '',
          kd_app_id: '',
          kd_app_secret: '',
          lcid: 2052,
          feishu_app_id: '',
          feishu_app_secret: '',
          feishu_encrypt_key: '',
          default_approver_openid: ''
        });

        const instances = ref([]);
        const flows = ref([]);
        const userMaps = ref([]);

        const flowDialogVisible = ref(false);
        const flowForm = ref({ id: 0, flow_name: '', kingdee_form_id: '', feishu_approval_code: '', bill_no_field: 'FBillNo', status_field: 'FDocumentStatus', poll_filter_status: 'B', title_template: '', enabled: true });

        const userMapDialogVisible = ref(false);
        const userMapForm = ref({ id: 0, kd_user_id: '', kd_user_name: '', feishu_open_id: '', approver_open_id: '', dept_name: '' });

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
              configForm.value = { ...configForm.value, ...data.data };
            }
          } catch (e) {
            console.error('加载金蝶配置失败', e);
          }
        };

        const saveConfig = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/config', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...getAuthHeader() },
              body: JSON.stringify(configForm.value)
            });
            const data = await res.json();
            if (data.code === 200) {
              ElMessage.success('配置保存成功！');
              loadConfig();
            } else {
              ElMessage.error(data.msg || '保存失败');
            }
          } catch (e) {
            ElMessage.error('保存错误: ' + e.message);
          }
        };

        const testConnection = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/test', {
              method: 'POST',
              headers: { ...getAuthHeader() }
            });
            const data = await res.json();
            if (data.code === 200 && data.data.ok) {
              ElMessage.success(data.data.message);
            } else {
              ElMessage.error(data.data.error || '测试失败');
            }
          } catch (e) {
            ElMessage.error('测试异常: ' + e.message);
          }
        };

        const loadInstances = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/instances', { headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200) {
              instances.value = data.data || [];
            }
          } catch (e) { console.error('加载实例失败', e); }
        };

        const retryAudit = async (id) => {
          try {
            const res = await fetch('/api/v1/kingdee/instances/' + id + '/audit', { method: 'POST', headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200 && data.data.success) {
              ElMessage.success('重试审核成功，已反写金蝶！');
            } else {
              ElMessage.error('重试审核失败');
            }
            loadInstances();
          } catch (e) { ElMessage.error('请求失败: ' + e.message); }
        };

        const cancelInstance = async (id) => {
          try {
            const res = await fetch('/api/v1/kingdee/instances/' + id + '/cancel', { method: 'POST', headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200) {
              ElMessage.success('实例已撤销');
            }
            loadInstances();
          } catch (e) { ElMessage.error('撤销失败'); }
        };

        const loadFlows = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/flows', { headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200) {
              flows.value = data.data || [];
            }
          } catch (e) { console.error('加载流程失败', e); }
        };

        const openFlowDialog = (row) => {
          if (row) {
            flowForm.value = { ...row };
          } else {
            flowForm.value = { id: 0, flow_name: '', kingdee_form_id: 'SAL_SaleOrder', feishu_approval_code: '', bill_no_field: 'FBillNo', status_field: 'FDocumentStatus', poll_filter_status: 'B', title_template: '销售订单 - {FBillNo}', enabled: true };
          }
          flowDialogVisible.value = true;
        };

        const saveFlow = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/flows', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...getAuthHeader() },
              body: JSON.stringify(flowForm.value)
            });
            const data = await res.json();
            if (data.code === 200) {
              ElMessage.success('流程保存成功');
              flowDialogVisible.value = false;
              loadFlows();
            }
          } catch (e) { ElMessage.error('保存失败'); }
        };

        const loadUserMaps = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/usermaps', { headers: getAuthHeader() });
            const data = await res.json();
            if (data.code === 200) {
              userMaps.value = data.data || [];
            }
          } catch (e) { console.error('加载映射失败', e); }
        };

        const openUserMapDialog = (row) => {
          if (row) {
            userMapForm.value = { ...row };
          } else {
            userMapForm.value = { id: 0, kd_user_id: '', kd_user_name: '', feishu_open_id: '', approver_open_id: '', dept_name: '' };
          }
          userMapDialogVisible.value = true;
        };

        const saveUserMap = async () => {
          try {
            const res = await fetch('/api/v1/kingdee/usermaps', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json', ...getAuthHeader() },
              body: JSON.stringify(userMapForm.value)
            });
            const data = await res.json();
            if (data.code === 200) {
              ElMessage.success('映射保存成功');
              userMapDialogVisible.value = false;
              loadUserMaps();
            }
          } catch (e) { ElMessage.error('保存失败'); }
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

        const getStatusTagType = (status) => {
          switch (status) {
            case 'APPROVED': return 'success';
            case 'PENDING_AUDIT': return 'warning';
            case 'PENDING': return 'primary';
            case 'REJECTED': return 'danger';
            case 'CANCELED': return 'info';
            case 'AUDIT_FAILED': return 'danger';
            default: return '';
          }
        };

        const handleTabChange = (name) => {
          if (name === 'instances') loadInstances();
          if (name === 'flows') loadFlows();
          if (name === 'usermaps') loadUserMaps();
          if (name === 'config') loadConfig();
        };

        onMounted(() => {
          loadConfig();
          loadInstances();
        });

        return {
          activeTab, handleTabChange,
          queryForm, executingQuery, customResults, customResultCols, executeCustomQuery,
          configForm, saveConfig, testConnection,
          instances, loadInstances, retryAudit, cancelInstance, getStatusTagType,
          flows, loadFlows, flowDialogVisible, flowForm, openFlowDialog, saveFlow,
          userMaps, loadUserMaps, userMapDialogVisible, userMapForm, openUserMapDialog, saveUserMap
        };
      }
    }).use(ElementPlus).mount('#app');
  </script>
</body>
</html>`
