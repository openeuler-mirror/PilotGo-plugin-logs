/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package global

import (
	"context"
	"time"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/resourcemanage"
)

var (
	RootCtx = context.Background()

	HeartbeatPeriod = 5 * time.Second // 日志采集组件状态检测周期s
)

var ERManager *resourcemanage.ErrorReleaseManagement

func init() {

}
