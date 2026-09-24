package api

const HikUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>海康互联 (Hik-Connect) 门禁与考勤控制台</title>
  <!-- Element Plus CSS & Vue 3 -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.css" />
  <script src="https://cdn.jsdelivr.net/npm/vue@3.4.27/dist/vue.global.prod.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/element-plus@2.7.5/dist/index.full.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/@element-plus/icons-vue@2.3.1/dist/index.iife.min.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f7fa; margin: 0; padding: 20px; color: #303133; }
    .header { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); margin-bottom: 20px; display: flex; justify-content: space-between; align-items: center; }
    .header h2 { margin: 0; font-size: 20px; color: #1f2937; display: flex; align-items: center; gap: 10px; }
    .card-box { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); }
    .door-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 20px; margin-top: 15px; }
    .door-card { border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; background: #fafafa; transition: all 0.2s; }
    .door-card:hover { border-color: #4f46e5; box-shadow: 0 4px 12px rgba(79, 70, 229, 0.1); background: #fff; }
    .door-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
    .door-title { font-weight: 600; font-size: 16px; color: #111827; }
    .door-code { font-size: 12px; color: #6b7280; font-family: monospace; }
    .door-actions { display: flex; gap: 8px; margin-top: 15px; }
    .status-badge { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 6px; }
    .status-online { background: #10b981; }
    .status-offline { background: #9ca3af; }
    .filter-bar { display: flex; gap: 12px; margin-bottom: 15px; align-items: center; }
    .shift-day { background-color: #d1f2e1 !important; color: #008a3d !important; font-weight: bold; cursor: pointer; text-align: center; }
    .shift-night { background-color: #e0e0ff !important; color: #4f46e5 !important; font-weight: bold; cursor: pointer; text-align: center; }
    .shift-exception { background-color: #ffe4e6 !important; color: #e11d48 !important; font-weight: bold; cursor: pointer; text-align: center; }
    .shift-blank { background-color: transparent !important; cursor: pointer; text-align: center; }
    .shift-leave { background-color: #fef3c7 !important; color: #d97706 !important; font-weight: bold; cursor: pointer; text-align: center; }
    .el-table .cell { padding: 0 4px !important; }
  </style>
</head>
<body>
  <div id="app">
    <div class="header">
      <h2>
        <el-icon color="#4f46e5"><video-camera /></el-icon>
        海康互联 (Hik-Connect) 物联网管控中心
      </h2>
      <el-tag type="success" effect="dark" round>MVP 插件模式运行中</el-tag>
    </div>

    <div class="card-box">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 标签页 1：门禁设备管控 -->
        <el-tab-pane label="门禁设备管控" name="doors">
          <div class="filter-bar">
            <el-button type="primary" :loading="syncingDoors" @click="syncDoors">
              <el-icon><refresh /></el-icon> 同步海康门禁点
            </el-button>
            <el-text type="info" style="font-size: 13px;">包含远程一键开门、关门控制，自动推送动作至设备点</el-text>
          </div>

          <div class="door-grid" v-loading="loadingDoors">
            <div v-for="d in doors" :key="d.door_index_code" class="door-card">
              <div class="door-header">
                <div class="door-title">{{ d.door_name }}</div>
                <el-tag :type="d.status === 1 ? 'success' : 'info'" size="small">
                  <span class="status-badge" :class="d.status === 1 ? 'status-online' : 'status-offline'"></span>
                  {{ d.status === 1 ? '设备在线' : '设备离线' }}
                </el-tag>
              </div>
              <div class="door-code">编号: {{ d.door_index_code }} (通道 {{ d.channel_no }})</div>

              <div class="door-actions">
                <el-button type="success" size="small" :loading="controlling[d.door_index_code + '_1']" @click="controlDoor(d.door_index_code, 1)">
                  远程开门
                </el-button>
                <el-button type="danger" size="small" :loading="controlling[d.door_index_code + '_0']" @click="controlDoor(d.door_index_code, 0)">
                  关门
                </el-button>
                <el-button type="warning" size="small" :loading="controlling[d.door_index_code + '_2']" @click="controlDoor(d.door_index_code, 2)">
                  常开模式
                </el-button>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 标签页 2：考勤刷卡记录 -->
        <el-tab-pane label="打卡考勤记录" name="attendance">
          <div class="filter-bar">
            <el-input v-model="attQuery.person_name" placeholder="检索姓名" style="width: 160px;" clearable></el-input>
            <el-date-picker v-model="attQuery.start_date" type="date" value-format="YYYY-MM-DD" placeholder="开始日期" style="width: 140px;"></el-date-picker>
            <el-date-picker v-model="attQuery.end_date" type="date" value-format="YYYY-MM-DD" placeholder="结束日期" style="width: 140px;"></el-date-picker>
            <el-button type="primary" @click="loadAttendance">查询记录</el-button>
          </div>

          <el-table :data="attendance" stripe v-loading="loadingAtt" style="width: 100%;">
            <el-table-column prop="job_no" label="工号" width="120"></el-table-column>
            <el-table-column prop="person_name" label="姓名" width="120"></el-table-column>
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
        </el-tab-pane>

        <!-- 标签页 3：人员与组织档案 -->
        <el-tab-pane label="人员组织结构" name="persons">
          <div class="filter-bar">
            <el-input v-model="personKeyword" placeholder="检索人员/工号/手机号" style="width: 220px;" clearable @keyup.enter="loadPersons"></el-input>
            <el-button type="primary" @click="loadPersons">查询人员</el-button>
            <el-button type="success" :loading="syncingPersons" @click="syncPersons">同步组织与人员数据</el-button>
          </div>

          <el-table :data="persons.slice((personPage - 1) * personPageSize, personPage * personPageSize)" stripe v-loading="loadingPersons" style="width: 100%;">
            <el-table-column prop="person_id" label="海康 PersonID" width="180"></el-table-column>
            <el-table-column prop="person_name" label="姓名" width="150"></el-table-column>
            <el-table-column prop="job_no" label="工号" width="150"></el-table-column>
            <el-table-column prop="phone_no" label="手机号码"></el-table-column>
          </el-table>

          <div style="margin-top: 15px; display: flex; justify-content: flex-end;">
            <el-pagination
              v-model:current-page="personPage"
              v-model:page-size="personPageSize"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              :total="persons.length">
            </el-pagination>
          </div>
        </el-tab-pane>

        <!-- 标签页 4：排班考勤汇总 -->
        <el-tab-pane label="排班考勤汇总" name="matrix">
          <div class="filter-bar">
            <el-date-picker
              v-model="matrixMonth"
              type="month"
              placeholder="选择月份"
              value-format="YYYY-MM"
              @change="loadMatrix">
            </el-date-picker>
            <el-button type="primary" :loading="calculatingMatrix" @click="calculateMatrix">
              <el-icon><cpu /></el-icon> 一键智能排班判定
            </el-button>
          </div>

          <el-table :data="matrixData" stripe v-loading="loadingMatrix" style="width: 100%; margin-top: 15px;" border>
            <el-table-column prop="person_name" label="姓名" width="90" fixed="left"></el-table-column>
            <el-table-column prop="job_no" label="工号" width="100" fixed="left"></el-table-column>
            <el-table-column label="考勤规则" width="90" fixed="left">
               <template #default>排班打卡</template>
            </el-table-column>
            
            <el-table-column v-for="day in daysInMonth" :key="day.num" width="50" align="center">
              <template #header>
                <div style="line-height: 1.2; font-size: 12px; color: #606266;">
                  <div>{{ day.week }}</div>
                  <div>{{ parseInt(day.num) }}</div>
                </div>
              </template>
              <template #default="scope">
                <div 
                  :class="getCellClass(scope.row.days[day.num])" 
                  @click="handleCellClick(scope.row, day.num)"
                  style="width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; min-height: 40px; font-size: 12px;">
                  {{ scope.row.days[day.num] || '' }}
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>

      <!-- 手工调整弹窗 -->
      <el-dialog v-model="editDialogVisible" title="考勤结果手工调整" width="400px">
        <el-form :model="editForm" label-width="80px">
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
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="editDialogVisible = false">取消</el-button>
            <el-button type="primary" :loading="savingEdit" @click="saveEdit">保存</el-button>
          </span>
        </template>
      </el-dialog>
    </div>
  </div>

  <script>
    const { createApp, ref, reactive, onMounted } = Vue;
    const app = createApp({
      setup() {
        const activeTab = ref('doors');
        const doors = ref([]);
        const loadingDoors = ref(false);
        const syncingDoors = ref(false);
        const controlling = reactive({});

        const attendance = ref([]);
        const loadingAtt = ref(false);
        const attQuery = reactive({ person_name: '', start_date: '', end_date: '' });

        const persons = ref([]);
        const loadingPersons = ref(false);
        const syncingPersons = ref(false);
        const personKeyword = ref('');
        const personPage = ref(1);
        const personPageSize = ref(20);

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

        const loadPersons = async () => {
          loadingPersons.value = true;
          try {
            const res = await fetch('/api/v1/hikiot/persons/search?keyword=' + encodeURIComponent(personKeyword.value), { headers: getAuthHeader() });
            const json = await res.json();
            if (json.code === 200) persons.value = json.data || [];
          } catch(e) {}
          loadingPersons.value = false;
        };

        const syncPersons = async () => {
          syncingPersons.value = true;
          try {
            await fetch('/api/v1/hikiot/sync/persons', { method: 'POST', headers: getAuthHeader() });
            ElementPlus.ElMessage.success('人员组织数据同步完成');
            await loadPersons();
          } catch(e) {}
          syncingPersons.value = false;
        };

        const handleTabChange = (tabName) => {
          if (tabName === 'doors') loadDoors();
          // if (tabName === 'attendance') loadAttendance(); // 用户要求点击查询再拉取
          if (tabName === 'persons') loadPersons();
        };

        onMounted(() => {
          loadDoors();
        });

        // --- 排班矩阵视图逻辑 ---
        const matrixMonth = ref(new Date().toISOString().slice(0, 7)); // 默认当前月
        const matrixData = ref([]);
        const loadingMatrix = ref(false);
        const calculatingMatrix = ref(false);
        const daysInMonth = ref([]);
        
        const editDialogVisible = ref(false);
        const savingEdit = ref(false);
        const editForm = reactive({ personId: '', personName: '', date: '', shiftType: '', remark: '' });

        const loadMatrix = async () => {
          if (!matrixMonth.value) return;
          loadingMatrix.value = true;
          try {
            // 计算这个月有多少天，生成动态列
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

        const calculateMatrix = async () => {
          if (!matrixMonth.value) return;
          calculatingMatrix.value = true;
          try {
            const res = await fetch('/api/v1/hikiot/attendance/calculate', {
              method: 'POST',
              headers: getAuthHeader(),
              body: JSON.stringify({ month: matrixMonth.value })
            });
            const json = await res.json();
            if (json.code === 200) {
              ElementPlus.ElMessage.success('计算完成');
              await loadMatrix();
            } else {
              ElementPlus.ElMessage.error(json.msg || '计算失败');
            }
          } catch(e) {}
          calculatingMatrix.value = false;
        };

        const getCellClass = (shiftType) => {
          if (shiftType === '白班') return 'shift-day';
          if (shiftType === '夜班') return 'shift-night';
          if (shiftType === '请假' || shiftType === '休息') return 'shift-leave';
          if (shiftType && shiftType !== '空白') return 'shift-exception';
          return 'shift-blank';
        };

        const handleCellClick = (row, day) => {
          editForm.personId = row.person_id;
          editForm.personName = row.person_name;
          editForm.date = matrixMonth.value + '-' + day;
          editForm.shiftType = row.days[day] || '空白';
          editForm.remark = row.details[day] ? row.details[day].remark : '';
          editDialogVisible.value = true;
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

        // --- 拦截 tab 切换加载矩阵 ---
        const originalHandleTabChange = handleTabChange;
        const newHandleTabChange = (name) => {
          if (name === 'matrix' && matrixData.value.length === 0) {
             loadMatrix();
          }
          originalHandleTabChange(name);
        };

        return {
          activeTab, doors, loadingDoors, syncingDoors, controlling,
          attendance, loadingAtt, attQuery, loadAttendance,
          persons, loadingPersons, syncingPersons, personKeyword, personPage, personPageSize, loadPersons, syncPersons,
          loadDoors, controlDoor, handleTabChange: newHandleTabChange,
          matrixMonth, matrixData, loadingMatrix, calculatingMatrix, daysInMonth, loadMatrix, calculateMatrix,
          getCellClass, handleCellClick, editDialogVisible, savingEdit, editForm, saveEdit
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
