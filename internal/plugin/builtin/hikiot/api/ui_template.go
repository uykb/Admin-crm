package api

const HikUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>海康互联 (Hik-Connect) 门禁与考勤控制台</title>
  <!-- Element Plus CSS & Vue 3 -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.css" />
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/theme-chalk/dark/css-vars.css" />
  <script src="https://cdn.jsdelivr.net/npm/vue@3.4.27/dist/vue.global.prod.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.full.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/@element-plus/icons-vue@2.3.1/dist/index.iife.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/xlsx@0.18.5/dist/xlsx.full.min.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: var(--el-bg-color-page); margin: 0; padding: 20px; color: var(--el-text-color-primary); }
    .header-box { background: var(--el-bg-color-overlay); padding: 20px; border-radius: 8px; box-shadow: var(--el-box-shadow-light); margin-bottom: 20px; }
    .card-box { background: var(--el-bg-color-overlay); padding: 20px; border-radius: 8px; box-shadow: var(--el-box-shadow-light); }
    .door-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 20px; margin-top: 15px; }
    .door-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
    .door-title { font-weight: 600; font-size: 16px; color: var(--el-text-color-primary); }
    .door-code { font-size: 12px; color: var(--el-text-color-regular); font-family: monospace; }
    .door-actions { display: flex; gap: 8px; justify-content: flex-end; padding-top: 12px; border-top: 1px solid var(--el-border-color-lighter); margin-top: 15px; }
    .filter-bar { display: flex; gap: 12px; margin-bottom: 15px; align-items: center; flex-wrap: wrap; }
    .el-table .cell { padding: 0 4px !important; }

    /* 📱 手机移动端响应式自适应优化 */
    @media (max-width: 768px) {
      body { padding: 6px; }
      .header-box { display: none; } /* 移动端隐藏重复顶部 Header 节省 80px 纵向空间 */
      .card-box { padding: 10px; border-radius: 6px; }
      .filter-bar { gap: 8px; flex-direction: column; align-items: stretch; }
      .filter-bar .el-form-item { margin-bottom: 8px; margin-right: 0 !important; width: 100%; }
      .filter-bar .el-input, .filter-bar .el-date-picker, .filter-bar .el-select { width: 100% !important; }
      .filter-bar .el-button { width: 100%; margin-left: 0 !important; margin-top: 4px; min-height: 38px; }
      .door-grid { grid-template-columns: 1fr; gap: 12px; }
      .door-actions { justify-content: space-between; gap: 6px; }
      .door-actions .el-button { flex: 1; margin-left: 0 !important; }
      .el-dialog { width: 94% !important; max-width: 480px; margin: 10px auto !important; }
      .el-table { font-size: 11px; }
      .el-table .cell { padding: 0 2px !important; }
      .el-tag { padding: 0 2px; font-size: 11px; }
    }
  </style>
</head>
<body>
  <div id="app">
    <div class="header-box">
      <el-page-header @back="goBack" :icon="null">
        <template #content>
          <div style="display: flex; align-items: center; gap: 10px;">
            <el-icon color="#4f46e5" size="24"><video-camera /></el-icon>
            <span style="font-size: 18px; font-weight: 600;">{{ getTitle() }}</span>
          </div>
        </template>
        <template #extra>
          <el-tag type="success" effect="dark" round>MVP 插件模式运行中</el-tag>
        </template>
      </el-page-header>
    </div>

    <div class="card-box">
      
      <!-- 页面 1：门禁设备管控 -->
      <div v-if="activeTab === 'doors'">
        <el-form :inline="true" class="filter-bar">
          <el-form-item>
            <el-button type="primary" :loading="syncingDoors" @click="syncDoors">
              <el-icon><refresh /></el-icon> 同步海康门禁点
            </el-button>
          </el-form-item>
          <el-form-item>
            <el-text type="info" style="font-size: 13px;">包含远程一键开门、关门控制，自动推送动作至设备点</el-text>
          </el-form-item>
        </el-form>

        <div v-if="!loadingDoors && doors.length === 0" style="padding: 40px 0;">
          <el-empty description="暂无设备数据，请点击同步"></el-empty>
        </div>

        <div class="door-grid" v-loading="loadingDoors">
          <el-card shadow="hover" v-for="d in doors" :key="d.door_index_code" :body-style="{ padding: '16px' }">
            <div class="door-header">
              <div class="door-title">{{ d.door_name }}</div>
              <el-tag :type="d.status === 1 ? 'success' : 'info'" effect="light" size="small">
                {{ d.status === 1 ? '设备在线' : '设备离线' }}
              </el-tag>
            </div>
            <div class="door-code">编号: {{ d.door_index_code }} (通道 {{ d.channel_no }})</div>

            <div class="door-actions">
              <el-popconfirm title="确定要远程开启此门吗？" @confirm="controlDoor(d.door_index_code, 1)">
                <template #reference>
                  <el-button type="success" text size="small" :loading="controlling[d.door_index_code + '_1']">远程开门</el-button>
                </template>
              </el-popconfirm>
              <el-button type="info" text size="small" :loading="controlling[d.door_index_code + '_0']" @click="controlDoor(d.door_index_code, 0)">关门</el-button>
              <el-button type="warning" text size="small" :loading="controlling[d.door_index_code + '_2']" @click="controlDoor(d.door_index_code, 2)">常开模式</el-button>
            </div>
          </el-card>
        </div>
      </div>

      <!-- 页面 2：考勤刷卡记录 -->
      <div v-if="activeTab === 'attendance'">
        <el-form :inline="true" class="filter-bar">
          <el-form-item>
            <el-input v-model="attQuery.person_name" placeholder="检索姓名/编码/工号" style="width: 180px;" clearable></el-input>
          </el-form-item>
          <el-form-item>
            <el-date-picker v-model="attQuery.start_date" type="date" value-format="YYYY-MM-DD" placeholder="开始日期" style="width: 140px;"></el-date-picker>
          </el-form-item>
          <el-form-item>
            <el-date-picker v-model="attQuery.end_date" type="date" value-format="YYYY-MM-DD" placeholder="结束日期" style="width: 140px;"></el-date-picker>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadAttendance">查询记录</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="attendance" stripe v-loading="loadingAtt" style="width: 100%;">
          <template #empty><el-empty description="暂无打卡数据，请尝试切换日期"></el-empty></template>
          <el-table-column prop="job_no" label="编码" width="100"></el-table-column>
          <el-table-column prop="person_name" label="姓名" width="100"></el-table-column>
          <el-table-column prop="door_name" label="通行门禁位置"></el-table-column>
          <el-table-column label="打卡方式" width="130">
            <template #default="scope">
              <el-tag :type="scope.row.verify_mode === 1 ? 'primary' : 'success'" size="small">
                {{ scope.row.verify_mode === 1 ? '人脸识别' : '刷卡通行' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="clock_time" label="打卡时间" width="200"></el-table-column>
        </el-table>
      </div>

      <!-- 页面 3：排班考勤汇总 -->
      <div v-if="activeTab === 'matrix'">
        <el-form :inline="true" class="filter-bar">
          <el-form-item>
            <el-date-picker
              v-model="matrixMonth"
              type="month"
              placeholder="选择月份"
              value-format="YYYY-MM"
              @change="loadMatrix">
            </el-date-picker>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="calculatingMatrix" @click="calculateMatrix">
              <el-icon><cpu /></el-icon> 一键智能排班判定
            </el-button>
            <el-button type="success" :loading="syncingPersons" @click="syncPersons" style="margin-left:8px;">
              <el-icon><user /></el-icon> 同步部门人员
            </el-button>
            <el-button type="danger" :loading="clearingResults" @click="clearResults" style="margin-left:8px;">
              <el-icon><delete /></el-icon> 清空核算数据
            </el-button>
            <el-button type="info" :loading="exportingMatrix" @click="exportMatrix" style="margin-left:8px;">
              <el-icon><download /></el-icon> 导出 Excel
            </el-button>
          </el-form-item>
        </el-form>

        <el-table :data="matrixData" stripe v-loading="loadingMatrix" style="width: 100%;" border>
          <template #empty><el-empty description="暂无排班数据，请点击一键智能判定"></el-empty></template>
          <el-table-column prop="person_name" label="姓名" width="70" fixed="left" align="center"></el-table-column>
          <el-table-column prop="job_no" label="编码" width="75" fixed="left" align="center"></el-table-column>
          
          <el-table-column v-for="day in daysInMonth" :key="day.num" width="55" align="center">
            <template #header>
              <div style="line-height: 1.2; font-size: 12px; color: #606266;">
                <div>{{ day.week }}</div>
                <div>{{ parseInt(day.num) }}</div>
              </div>
            </template>
            <template #default="scope">
              <div @click="handleCellClick(scope.row, day.num)" style="cursor: pointer;">
                <el-tag v-if="scope.row.days[day.num] === '白班'" type="success" effect="light" size="small">白班</el-tag>
                <el-tag v-else-if="scope.row.days[day.num] === '夜班'" type="primary" effect="dark" size="small">夜班</el-tag>
                <el-tag v-else-if="scope.row.days[day.num] === '请假'" type="warning" effect="light" size="small">请假</el-tag>
                <el-tag v-else-if="scope.row.days[day.num] === '休息'" type="info" effect="light" size="small">休息</el-tag>
                <el-tag v-else-if="scope.row.days[day.num]" type="danger" effect="plain" size="small">{{ scope.row.days[day.num] }}</el-tag>
                <span v-else style="color: #ccc;">-</span>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 手工调整与打卡明细穿透弹窗 -->
      <el-dialog v-model="editDialogVisible" title="考勤明细与手工调整" width="500px">
        <el-tabs v-model="editTab">
          <el-tab-pane label="状态调整" name="form">
            <el-form :model="editForm" label-width="80px" style="margin-top: 15px;">
              <el-form-item label="员工">
                <el-input v-model="editForm.personName" disabled></el-input>
              </el-form-item>
              <el-form-item label="日期">
                <el-input v-model="editForm.date" disabled></el-input>
              </el-form-item>
              <el-form-item label="判定状态">
                <el-select v-model="editForm.shiftType" style="width: 100%">
                  <el-option label="白班" value="白班"></el-option>
                  <el-option label="夜班" value="夜班"></el-option>
                  <el-option label="异常" value="异常"></el-option>
                  <el-option label="缺卡" value="缺卡"></el-option>
                  <el-option label="请假" value="请假"></el-option>
                  <el-option label="休息" value="休息"></el-option>
                </el-select>
              </el-form-item>
              <el-form-item label="备注">
                <el-input v-model="editForm.remark" type="textarea"></el-input>
              </el-form-item>
            </el-form>
          </el-tab-pane>
          <el-tab-pane label="当日打卡明细" name="records">
            <el-table :data="cellRecords" stripe v-loading="loadingCellRecords" style="width: 100%; margin-top: 10px;" height="240">
              <template #empty><el-empty description="当日无打卡记录" :image-size="60"></el-empty></template>
              <el-table-column prop="clock_time" label="打卡时间" width="160"></el-table-column>
              <el-table-column prop="door_name" label="通行位置"></el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="editDialogVisible = false">取消</el-button>
            <el-button type="primary" :loading="savingEdit" @click="saveEdit">保存</el-button>
          </span>
        </template>
      </el-dialog>

      <!-- 智能判定分段进度弹窗 -->
      <el-dialog title="一键智能排班判定中" v-model="progressVisible" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false" width="420px">
        <div style="text-align: center; padding: 20px 0;">
          <el-progress type="circle" :percentage="calcPercent" status="success" :stroke-width="10" :width="140"></el-progress>
          <div style="margin-top: 20px; font-size: 15px; font-weight: bold; color: #409EFF;">{{ calcStatusText }}</div>
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">正在分段请求海康 API 并执行智能会话判定...</div>
        </div>
      </el-dialog>
    </div>
  </div>

  <script>
    const { createApp, ref, reactive, onMounted } = Vue;
    const app = createApp({
      setup() {
        const activeTab = ref('doors');
        const getTitle = () => {
          if (activeTab.value === 'doors') return '门禁设备管控';
          if (activeTab.value === 'attendance') return '打卡考勤记录';
          return '排班考勤汇总';
        };

        const doors = ref([]);
        const loadingDoors = ref(false);
        const syncingDoors = ref(false);
        const controlling = reactive({});

        const attendance = ref([]);
        const loadingAtt = ref(false);
        const attQuery = reactive({ person_name: '', start_date: '', end_date: '' });

        const getAuthHeader = () => {
          const urlParams = new URLSearchParams(window.location.search);
          const token = urlParams.get('token') || localStorage.getItem('apeadmin_token') || localStorage.getItem('token') || '';
          if (urlParams.get('token')) {
            try { localStorage.setItem('apeadmin_token', urlParams.get('token')); } catch(e) {}
          }
          return token ? { 'Authorization': 'Bearer ' + token } : {};
        };

        const loadDoors = async () => {
          loadingDoors.value = true;
          try {
            const res = await fetch('/api/v1/hikiot/doors', { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) doors.value = json.data || [];
          } catch(e) {}
          loadingDoors.value = false;
        };

        const syncDoors = async () => {
          syncingDoors.value = true;
          try {
            await fetch('/api/v1/hikiot/sync/doors', { method: 'POST', headers: getAuthHeader() });
            ElementPlus.ElMessage.success('海康门禁点同步完成');
            await loadDoors();
          } catch(e) {}
          syncingDoors.value = false;
        };

        const controlDoor = async (doorCode, cmd) => {
          const key = doorCode + '_' + cmd;
          controlling[key] = true;
          try {
            const res = await fetch('/api/v1/hikiot/doors/control', {
              method: 'POST',
              headers: { ...getAuthHeader(), 'Content-Type': 'application/json' },
              body: JSON.stringify({ door_index_code: doorCode, command: cmd })
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success(json.data || '指令下发成功');
              await loadDoors();
            } else {
              ElementPlus.ElMessage.error(json.msg || '指令下发失败');
            }
          } catch(e) {}
          controlling[key] = false;
        };

        const loadAttendance = async () => {
          loadingAtt.value = true;
          try {
            const params = new URLSearchParams();
            if (attQuery.person_name) params.append('person_name', attQuery.person_name);
            if (attQuery.start_date) params.append('start_date', attQuery.start_date);
            if (attQuery.end_date) params.append('end_date', attQuery.end_date);
            const res = await fetch('/api/v1/hikiot/attendance/records?' + params.toString(), { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) attendance.value = json.data || [];
          } catch(e) {}
          loadingAtt.value = false;
        };

        // --- 排班矩阵视图逻辑 ---
        const matrixMonth = ref(new Date().toISOString().slice(0, 7)); // 默认当前月
        const matrixData = ref([]);
        const loadingMatrix = ref(false);
        const calculatingMatrix = ref(false);
        const exportingMatrix = ref(false);
        const daysInMonth = ref([]);
        
        const editDialogVisible = ref(false);
        const editTab = ref('form');
        const cellRecords = ref([]);
        const loadingCellRecords = ref(false);
        const savingEdit = ref(false);
        const editForm = reactive({ personId: '', personName: '', date: '', shiftType: '', remark: '' });

        const loadMatrix = async () => {
          if (!matrixMonth.value) return;
          loadingMatrix.value = true;
          try {
            const [y, m] = matrixMonth.value.split('-');
            const days = new Date(y, m, 0).getDate();
            const cols = [];
            const weekDays = ['日', '一', '二', '三', '四', '五', '六'];
            for (let i = 1; i <= days; i++) {
              const d = new Date(y, parseInt(m) - 1, i);
              cols.push({
                num: i.toString().padStart(2, '0'),
                week: weekDays[d.getDay()]
              });
            }
            daysInMonth.value = cols;

            const res = await fetch('/api/v1/hikiot/attendance/matrix?month=' + matrixMonth.value, { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) {
              matrixData.value = json.data || [];
            }
          } catch(e) {}
          loadingMatrix.value = false;
        };

        const exportMatrix = () => {
          if (!matrixData.value || matrixData.value.length === 0) {
            ElementPlus.ElMessage.warning('没有可导出的数据');
            return;
          }
          exportingMatrix.value = true;
          try {
            const header = ['姓名', '编码'];
            daysInMonth.value.forEach(d => {
               header.push(d.week + '(' + parseInt(d.num) + ')');
            });
            
            const data = [header];
            matrixData.value.forEach(row => {
              const rowData = [row.person_name, row.job_no];
              daysInMonth.value.forEach(d => {
                rowData.push(row.days[d.num] || '');
              });
              data.push(rowData);
            });
            
            const ws = XLSX.utils.aoa_to_sheet(data);
            const wb = XLSX.utils.book_new();
            XLSX.utils.book_append_sheet(wb, ws, "排班考勤汇总");
            XLSX.writeFile(wb, '排班考勤汇总_' + matrixMonth.value + '.xlsx');
            
            ElementPlus.ElMessage.success('导出成功');
          } catch(e) {
            ElementPlus.ElMessage.error('导出失败: ' + e.message);
          }
          exportingMatrix.value = false;
        };

        const progressVisible = Vue.ref(false);
        const calcPercent = Vue.ref(0);
        const calcStatusText = Vue.ref('');

        const calculateMatrix = async () => {
          if (!matrixMonth.value) return;
          calculatingMatrix.value = true;
          progressVisible.value = true;
          calcPercent.value = 0;
          calcStatusText.value = '准备开始分段判定...';

          try {
            const yearMonth = matrixMonth.value; // "2026-09"
            const parts = yearMonth.split('-');
            const year = parseInt(parts[0]);
            const month = parseInt(parts[1]);
            const totalDays = new Date(year, month, 0).getDate();

            const chunks = [];
            const chunkSize = 5;
            for (let d = 1; d <= totalDays; d += chunkSize) {
              const startDay = d;
              const endDay = Math.min(d + chunkSize - 1, totalDays);
              const startStr = yearMonth + '-' + (startDay < 10 ? '0' + startDay : String(startDay));
              const endStr = yearMonth + '-' + (endDay < 10 ? '0' + endDay : String(endDay));
              chunks.push({ startStr, endStr, startDay, endDay });
            }

            for (let i = 0; i < chunks.length; i++) {
              const c = chunks[i];
              calcStatusText.value = '正在核算: ' + c.startDay + '日 ~ ' + c.endDay + '日';
              
              const res = await fetch('/api/v1/hikiot/attendance/calculate_range', {
                method: 'POST',
                headers: getAuthHeader(),
                body: JSON.stringify({ start_date: c.startStr, end_date: c.endStr })
              });
              const json = await res.json();
              if (json.code !== 200) {
                ElementPlus.ElMessage.warning('区间 ' + c.startStr + ' 响应: ' + (json.msg || '未知'));
              }
              calcPercent.value = Math.round(((i + 1) / chunks.length) * 100);
            }

            ElementPlus.ElMessage.success('一键智能排班判定全量完成！');
          } catch(e) {
            ElementPlus.ElMessage.error('判定过程异常: ' + e);
          } finally {
            progressVisible.value = false;
            calculatingMatrix.value = false;
            await loadMatrix();
          }
        };

        const syncingPersons = Vue.ref(false);
        const syncPersons = async () => {
          syncingPersons.value = true;
          try {
            const res = await fetch('/api/v1/hikiot/sync/persons', {
              method: 'POST',
              headers: getAuthHeader()
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('人员同步成功，共 ' + (json.data || 0) + ' 人');
            } else {
              ElementPlus.ElMessage.error(json.msg || '同步失败');
            }
          } catch(e) {
            ElementPlus.ElMessage.error('同步请求异常');
          }
          await loadMatrix();
          syncingPersons.value = false;
        };

        const clearingResults = Vue.ref(false);
        const clearResults = async () => {
          if (!matrixMonth.value) return;
          try {
            await ElementPlus.ElMessageBox.confirm(
              '确定要清空 ' + matrixMonth.value + ' 月的全部核算数据吗?清空后需重新点击一键智能排班判定。',
              '清空核算数据',
              { confirmButtonText: '确定清空', cancelButtonText: '取消', type: 'warning' }
            );
          } catch(e) { return; }
          clearingResults.value = true;
          try {
            const res = await fetch('/api/v1/hikiot/attendance/results?month=' + matrixMonth.value, {
              method: 'DELETE',
              headers: getAuthHeader(),
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success(json.data || '清空成功');
            } else {
              ElementPlus.ElMessage.error(json.msg || '清空失败');
            }
          } catch(e) {
            ElementPlus.ElMessage.error('请求异常');
          }
          await loadMatrix();
          clearingResults.value = false;
        };

        const handleCellClick = (row, day) => {
          editForm.personId = row.person_id;
          editForm.personName = row.person_name;
          editForm.date = matrixMonth.value + '-' + day;
          editForm.shiftType = row.days[day] || '空白';
          editForm.remark = row.details[day] ? row.details[day].remark : '';
          editTab.value = 'form';
          cellRecords.value = [];
          editDialogVisible.value = true;

          const fetchRecords = async () => {
            loadingCellRecords.value = true;
            try {
              const url = '/api/v1/hikiot/attendance/records?person_name=' + encodeURIComponent(row.person_name) + '&start_date=' + editForm.date + '&end_date=' + editForm.date;
              const res = await fetch(url, { headers: getAuthHeader() });
              const json = await res.json();
              if (json.code === 200) {
                cellRecords.value = json.data || [];
              }
            } catch(e) {}
            loadingCellRecords.value = false;
          };
          fetchRecords();
        };

        const saveEdit = async () => {
          savingEdit.value = true;
          try {
            const res = await fetch('/api/v1/hikiot/attendance/result', {
              method: 'PUT',
              headers: getAuthHeader(),
              body: JSON.stringify({
                person_id: editForm.personId,
                date: editForm.date,
                shift_type: editForm.shiftType,
                remark: editForm.remark
              })
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('调整保存成功');
              editDialogVisible.value = false;
              await loadMatrix();
            } else {
              ElementPlus.ElMessage.error(json.msg || '保存失败');
            }
          } catch(e) {}
          savingEdit.value = false;
        };



        onMounted(() => {
          const syncTheme = () => {
            try {
              if (window.parent && window.parent.document.documentElement.classList.contains('dark')) {
                document.documentElement.classList.add('dark');
              } else {
                document.documentElement.classList.remove('dark');
              }
            } catch(e) {}
          };
          syncTheme();
          try {
            if (window.parent) {
              const observer = new MutationObserver(syncTheme);
              observer.observe(window.parent.document.documentElement, { attributes: true, attributeFilter: ['class'] });
            }
          } catch(e) {
            setInterval(syncTheme, 500);
          }

          const path = window.location.pathname;
          if (path.includes('ui_records') || path.includes('records')) {
            activeTab.value = 'attendance';
          } else if (path.includes('ui_matrix') || path.includes('matrix')) {
            activeTab.value = 'matrix';
            loadMatrix();
          } else {
            activeTab.value = 'doors';
            loadDoors();
          }
        });

        const goBack = () => {
          window.history.back();
        };

        return {
          activeTab, getTitle,
          doors, loadingDoors, syncingDoors, controlling,
          attendance, loadingAtt, attQuery, loadAttendance,
          loadDoors, syncDoors, controlDoor,
          matrixMonth, matrixData, loadingMatrix, calculatingMatrix, exportingMatrix, daysInMonth, loadMatrix, calculateMatrix, exportMatrix,
          clearingResults, clearResults, syncingPersons, syncPersons,
          progressVisible, calcPercent, calcStatusText,
          handleCellClick, editDialogVisible, editTab, cellRecords, loadingCellRecords, savingEdit, editForm, saveEdit, goBack
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
