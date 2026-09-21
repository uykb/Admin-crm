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
      </el-tabs>
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
          if (tabName === 'attendance') loadAttendance();
          if (tabName === 'persons') loadPersons();
        };

        onMounted(() => {
          loadDoors();
        });

        return {
          activeTab, doors, loadingDoors, syncingDoors, controlling,
          attendance, loadingAtt, attQuery, loadAttendance,
          persons, loadingPersons, syncingPersons, personKeyword, personPage, personPageSize, loadPersons, syncPersons,
          loadDoors, controlDoor, handleTabChange
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
