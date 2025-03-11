/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package main

import (
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/conf"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/logger"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/pluginclient"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/resourcemanage"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/signal"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/webserver"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/webserver/proxy"
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
		sdklogger.Fatal("%s", err.Error())
	}
	global.ERManager = ermanager

	/*
		init plugin client
	*/
	pluginclient.InitPluginClient()

	/*
		websocket proxy management
	*/
	proxy.CreateWebsocketProxyManagement()

	/*
		init web server
	*/
	webserver.InitWebServer()
	// proxy.InitWebServerTcpHijackProxy()

	/*
		业务模块
	*/

	/*
		终止进程信号监听
	*/
	signal.SignalMonitoring()
}

func Close() {
	if proxy.WebsocketProxyManager != nil {
		proxy.WebsocketProxyManager.CloseAll()
	}
}
