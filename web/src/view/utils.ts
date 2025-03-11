/* 
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
/* 
* 1.formatDate(new Date(1533686888 * 1000), "YYYY-MM-DD HH:ii:ss");// 2019-07-09 19:44:01
* 2.formatDate(new Date(1562672641 * 1000), "YYYY-MM-DD 周W");//2019-07-09 周二 
*/
//时间戳转年月
export const formatDate = (date: any, formatStr: String) => {
    let arrWeek = ['日', '一', '二', '三', '四', '五', '六'],
      str = formatStr.replace(/yyyy|YYYY/, date.getFullYear()).replace(/yy|YY/, $addZero(date.getFullYear() % 100,
        2)).replace(/mm|MM/, $addZero(date.getMonth() + 1, 2)).replace(/m|M/g, date.getMonth() + 1).replace(
          /dd|DD/, $addZero(date.getDate(), 2)).replace(/d|D/g, date.getDate()).replace(/hh|HH/, $addZero(date
            .getHours(), 2)).replace(/h|H/g, date.getHours()).replace(/ii|II/, $addZero(date.getMinutes(), 2))
        .replace(/i|I/g, date.getMinutes()).replace(/ss|SS/, $addZero(date.getSeconds(), 2)).replace(/s|S/g, date
          .getSeconds()).replace(/w/g, date.getDay()).replace(/W/g, arrWeek[date.getDay()]);
    return str
  
  }
  function $addZero(v: any, size: number) {
    for (var i = 0, len: number = size - (v + "").length; i < len; i++) {
      v = "0" + v
    }
    return v + ""
  }
  
  // 日志等级
  export let levels = [
    {
      value: '0',
      label: 'emergency',
    },
    {
      value: '1',
      label: 'alert',
    },
    {
      value: '2',
      label: 'critical',
    },
    {
      value: '3',
      label: 'error',
    },
    {
      value: '4',
      label: 'warning',
    },
    {
      value: '5',
      label: 'notice',
    },
    {
      value: '6',
      label: 'informational',
    },
    {
      value: '7',
      label: 'debug',
    },
  ]