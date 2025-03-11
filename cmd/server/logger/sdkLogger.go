/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package logger

import (
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/conf"
	"gitee.com/openeuler/PilotGo/sdk/logger"
)

func InitLogger() {
	err := logger.Init(conf.Global_Config.Logopts)
	if err != nil {
		logger.Fatal(err.Error())
	}
}
