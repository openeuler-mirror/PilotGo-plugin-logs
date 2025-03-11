<!--
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
-->
<template>
  <div style="width: 98%; margin: 0 auto; height: 100vh; padding: 4px" v-loading="isloading">
    <div class="search">
      <div class="level">
        选择ip：<el-select v-model="ip_key" placeholder="请选择主机ip" style="width: 130px" @change="searchIp">
          <el-option v-for="item in ip_options" :key="item" :label="item" :value="item" />
        </el-select>
      </div>
      &emsp;
      <div class="level">
        选择等级：<el-select
          v-model="level_key"
          placeholder="请选择日志等级"
          style="width: 130px"
          @change="searchLevel"
        >
          <el-option v-for="item in level_options" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </div>
      &emsp;
      <div class="time">
        时间范围：
        <el-date-picker
          v-model="log_time"
          :disabled="realTime"
          type="datetimerange"
          range-separator="To"
          start-placeholder="Start date"
          end-placeholder="End date"
          @change="ChangeTimeRange"
          @clear="ChangeTimeRange"
        />
      </div>
      &emsp;
      <div class="service">
        选择服务：<el-select
          v-model="service_key"
          placeholder="请选择主机服务"
          style="width: 140px"
          @change="searchService"
        >
          <el-option-group v-for="group in service_options" :key="group.label" :label="group.label">
            <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
          </el-option-group>
        </el-select>
      </div>
      &emsp;
      <div class="level">
        日志模式：<el-select
          v-model="realTime"
          placeholder="请选择日志模式"
          style="width: 120px"
          @change="isResetLog = true"
        >
          <el-option label="实时" :value="true" />
          <el-option label="非实时" :value="false" />
        </el-select>
      </div>
      &emsp;&emsp;
      <el-button type="primary" @click="handleSearch()">查询</el-button>
    </div>
    <div class="log_list">
      <p class="head">
        <span style="width: 200px">时间</span>
        <span style="width: 140px">等级</span>
        <span style="width: calc(100% - 300px)">消息</span>
      </p>
      <ul
        id="ulC"
        v-infinite-scroll="load"
        :infinite-scroll-distance="1"
        :infinite-scroll-immediate="false"
        class="body"
      >
        <li v-for="(i, index) in log_stream" :title="'&emsp;' + (index + 1) + '.&nbsp;'" :name="index" :key="index">
          <div class="li_content">
            <span class="center" style="display: inline-block; width: 200px">{{
              formatDate(new Date(Number(i.timestamp)), "YYYY-MM-DD HH:ii:ss")
            }}</span>
            <span class="center" style="display: inline-block; width: 140px">{{
              levels.find((item) => item.value === i.level)?.label
            }}</span>
            <span style="display: inline-block; padding: 0 6px; width: calc(100% - 300px)">
              {{ i.message }}
            </span>
          </div>
        </li>
        <p v-if="loading" style="color: var(--el-color-primary)">
          <el-icon class="is-loading">
            <Loading />
          </el-icon>
          loading...
        </p>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from "vue";
import { levels } from "./utils.ts";
import { formatDate } from "./utils.ts";
import socket from "./socket";
import { getIpList } from "../api/log.ts";
import { ElMessage } from "element-plus";
import { useLogStore } from "../stores/log.ts";

onMounted(() => {
  getIpList().then((res) => {
    if (res.data.code === 200 && res.data.data) {
      ip_options.value = res.data.data;
    } else {
      ElMessage.error("获取ip列表失败，请刷新重试");
    }
  });
  // 开启websocket
  socket.init(receiveMessage, "");
});

interface logItem {
  timestamp: string;
  message: string;
  level: string;
  targetName: string;
}
const realTime = ref(false); // 是否实时监听日志变化
const isResetLog = ref(false); // 是否清空日志重新查询
const log_stream = ref([] as logItem[]);
const total_logs = ref(0);
const loading = ref(false);
const isloading = ref(false);

// ------- 数据量足够时，保证首次数据填充满屏幕，否则滚动监听失效 ------
const load_flag = ref(true);
// 计算li元素的高度，决定第一次渲染数据量
const computed_li_total_height = () => {
  let ul_height: number = document.getElementById("ulC")!.offsetHeight;
  let lis: any = document.getElementsByClassName("li_content");
  let totalHeight: number = 0;
  for (let i = 0; i < lis.length; i++) {
    totalHeight += lis[i].offsetHeight;
  }
  if (ul_height >= totalHeight) {
    load();
  } else {
    load_flag.value = false;
  }
};
watch(
  () => log_stream.value,
  (newV: logItem[]) => {
    if (newV.length > 0 && load_flag.value) {
      nextTick(() => {
        computed_li_total_height();
      });
    }
  }
);

// ip搜索功能
const ip_key = ref("");
let ip_options = ref([] as string[]);
const searchIp = (_ip: string) => {
  isResetLog.value = true;
  // 获取主机对应服务列表
};
watch(
  [() => useLogStore().ws_isOpen, () => ip_key.value],
  (newV) => {
    if (newV[0] && newV[1]) {
      socket.send({
        type: 1,
        joptions: null,
        data: ip_key.value + ":9995",
      });
    }
  },
  { immediate: true }
);
const receiveMessage = (message: any) => {
  let result = JSON.parse(message.data);
  switch (result.type) {
    case 3:
      // 与目标机器建立连接
      socket.send({ type: 2, joptions: null, data: null });
      break;

    default:
      if (!result.data.data) return;
      if (result.data.type === 0) {
        isloading.value = false;
        // 返回消息属于日志条目
        if (result.data.data) {
          if (noTail.value) {
            // 固定时间段
            log_stream.value = log_stream.value.concat(result.data.data.hits);
            total_logs.value = result.data.data.total;
          } else {
            log_stream.value.push(result.data.data);
          }
        }
      } else {
        // 返回消息属于主机服务列表
        let severiceOptios = [] as any[];
        Object.keys(result.data.data).forEach((i: string) => {
          let selectItem = { label: "", options: [] };
          selectItem["label"] = i;
          selectItem["options"] = result.data.data[i].map((j: string) => {
            let opt = { label: "", value: "" };
            opt["label"] = j;
            opt["value"] = j;
            return opt;
          });
          severiceOptios.push(selectItem);
        });
        service_options.value = JSON.parse(JSON.stringify(severiceOptios));
      }
      break;
  }
};
// 等级搜索功能
const level_key = ref("6");
let level_options = levels;
const searchLevel = (_level: string) => {
  isResetLog.value = true;
};

// 服务搜索功能
interface SelectGroupItem {
  label: string;
  options: [
    {
      value: string;
      label: string;
    }
  ];
}
const service_key = ref("");
let service_options = ref<SelectGroupItem[]>();
const searchService = (_service: string) => {
  isResetLog.value = true;
};

// 时间筛选功能
const log_time = ref<[Date, Date]>([new Date(new Date().getTime() - 2 * 60 * 60 * 1000), new Date()]);
const ChangeTimeRange = (value: any) => {
  if (value) {
    isResetLog.value = true;
    log_time.value[0] = value[0];
    log_time.value[1] = value[1];
  }
};

// 查询方法
const handleSearch = () => {
  // isloading.value = true;
  is_continue.value = true;
  getWsLogs({
    severity: level_key.value,
    service: service_key.value,
    timeRange: log_time.value,
    noTail: !realTime.value,
    from: 0,
    size: 20,
    isResetLog: isResetLog.value,
  });
};

// 加载更多日志
const log_size = ref(0);
let is_continue = ref(true);
const load = () => {
  if (total_logs.value == 0 || !is_continue.value || realTime.value) return;
  if (log_size.value >= total_logs.value) {
    log_size.value = total_logs.value;
    is_continue.value = false;
  } else {
    log_size.value = log_size.value + 20;
    is_continue.value = true;
  }
  loading.value = true;
  getWsLogs({
    severity: level_key.value,
    service: service_key.value,
    timeRange: log_time.value,
    noTail: !realTime.value,
    from: log_size.value,
    size: 20,
    type: 5,
  });
};

// 发送ws日志请求
const noTail = ref(false); // false:实时  true:固定时间段
const getWsLogs = (params: any) => {
  noTail.value = params.noTail;
  if (params.isResetLog) {
    log_stream.value = [];
    total_logs.value = 0;
  }
  let joptions = {
    severity: params.severity,
    since: params.timeRange[0] == "" ? "" : formatDate(params.timeRange[0], "YYYY-MM-DD HH:ii:ss"),
    until: params.timeRange[1] == "" ? "" : formatDate(params.timeRange[1], "YYYY-MM-DD HH:ii:ss"),
    unit: "",
    user: "",
    transport: "",
    notail: params.noTail,
    from: params.from,
    size: params.size ? params.size : null,
  } as any;
  if (!service_options.value) return;
  let selected_service = service_options.value.find((group: any) =>
    group.options.some((option: any) => option.label == params.service)
  );
  if (selected_service) {
    selected_service.label === "systemd"
      ? (joptions["unit"] = params.service)
      : (joptions[`${selected_service.label}`] = params.service);
  }
  socket.send({
    type: params.type ? params.type : 0,
    joptions,
    data: null,
  });
};

onBeforeUnmount(() => {
  // 离开页面关闭socket
  socket.close();
});
</script>

<style scoped lang="scss">
.search {
  height: 44px;
  display: flex;
  align-items: center;
}

.log_list {
  height: calc(100% - 60px);
  width: 100%;
  padding: 0;

  .head {
    margin: 0 1px;
    display: flex;
    align-items: center;
    justify-content: space-around;
    background: var(--el-color-primary-light-9);

    span {
      display: inline-block;
      text-align: center;
    }
  }

  .body {
    margin: 0 1px;
    list-style: none;
    overflow: auto;
    height: calc(100% - 70px);
    .li_content {
      display: flex;
      align-items: center;
    }
  }
}

.border-side {
  border-left: 1px solid var(--el-color-info-light-3);
  border-right: 1px solid var(--el-color-info-light-3);
}

.center {
  text-align: center;
}
</style>