/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package journald

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/global"
	"github.com/pkg/errors"
)

var JournaldCtx, JournaldCancel = context.WithCancel(global.RootCtx)

var _ sort.Interface = PageEntryBuffSortByTimestamp{}

type PageEntryBuffSortByTimestamp []string

func (peb PageEntryBuffSortByTimestamp) Len() int {
	return len(peb)
}

func (peb PageEntryBuffSortByTimestamp) Swap(i, j int) {
	peb[i], peb[j] = peb[j], peb[i]
}

func (peb PageEntryBuffSortByTimestamp) Less(i, j int) bool {
	i_raw_entry := map[string]interface{}{}
	if err := json.Unmarshal([]byte(peb[i]), &i_raw_entry); err != nil {
		global.ERManager.ErrorTransmit("logtools", "error", errors.Errorf("fail to unmarshal Journald JSON: %s; raw data: %+v(%d) ***", err, peb[i], len(peb[i])), false, true)
		return true
	}
	i_timestamp_int64, err := strconv.ParseInt(i_raw_entry["__REALTIME_TIMESTAMP"].(string), 10, 64)
	if err != nil {
		global.ERManager.ErrorTransmit("logtools", "error", errors.Errorf("fail to parse timestamp %s: %s", i_raw_entry["__REALTIME_TIMESTAMP"].(string), err.Error()), false, true)
		return true
	}

	j_raw_entry := map[string]interface{}{}
	if err := json.Unmarshal([]byte(peb[j]), &j_raw_entry); err != nil {
		global.ERManager.ErrorTransmit("logtools", "error", errors.Errorf("fail to unmarshal Journald JSON: %s; raw data: %+v(%d) ***", err, peb[j], len(peb[j])), false, false)
		return true
	}
	j_timestamp_int64, err := strconv.ParseInt(j_raw_entry["__REALTIME_TIMESTAMP"].(string), 10, 64)
	if err != nil {
		global.ERManager.ErrorTransmit("logtools", "error", errors.Errorf("fail to parse timestamp %s: %s", j_raw_entry["__REALTIME_TIMESTAMP"].(string), err.Error()), false, false)
		return true
	}
	return int(i_timestamp_int64) < int(j_timestamp_int64)
}
