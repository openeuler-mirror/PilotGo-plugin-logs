/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package main

import (
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/conf"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/global"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/logger"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/logtools"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/resourcemanage"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/signal"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/webserver"
	sdklogger "gitee.com/openeuler/PilotGo/sdk/logger"
)

func main() {
	/*
		init config
	*/
	conf.InitConfig()

	/*
		init logger
	*/
	logger.InitLogger()

	/*
		init error control、resource release、goroutine end management
	*/
	ermanager, err := resourcemanage.CreateErrorReleaseManager(global.RootCtx, Close)
	if err != nil {
		sdklogger.Fatal(err.Error())
	}
	global.ERManager = ermanager

	/*
		日志采集组件管理
	*/
	logtools.CreateLogClientsManager()

	/*

	 */
	global.InitOSName()

	/*
		init web server
	*/
	webserver.InitWebserver()

	/*
		终止进程信号监听
	*/
	signal.SignalMonitoring()
}

func Close() {
	if logtools.LogCollector != nil {
		logtools.LogCollector.CloseAll()
	}
}
