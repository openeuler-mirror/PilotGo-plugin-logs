/* 
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
import { ref } from 'vue'
import { defineStore } from 'pinia'
interface LogSearchList {
    ip:string,
    timeRange:[Date,Date];
    level:string;
    service:{label:string,value:string};
    realTime:boolean;
}
export const useLogStore = defineStore('log', () => {
  const search_list = ref([] as LogSearchList[]);
  const ws_isOpen = ref(false);
  const clientId = ref(parseInt(Math.random() * 100000+'')); // websocket标识id，初始化为随机数
  const updateLogList = (param:any) => {
    let ip_index = search_list.value.findIndex(item => item.ip === param.ip);
    ip_index !== -1 ? search_list.value[ip_index] = param : search_list.value.push(param);
  }

  const $reset = () => {
    search_list.value = [];
    ws_isOpen.value = false;
  }
  return {clientId,ws_isOpen,search_list,updateLogList,$reset}
})