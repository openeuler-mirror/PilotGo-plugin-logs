/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package pluginclient

import "gitee.com/openeuler/PilotGo/sdk/plugin/client"

const Version = "1.0.1"

var PluginInfo = &client.PluginInfo{
	Name:        "logs",
	Version:     Version,
	Description: "logs plugin for PilotGo",
	Author:      "wangjunqi",
	Email:       "wangjunqi@kylinos.cn",
	Url:         "", // 客户端建立连接的插件服务端地址，非插件配置文件中web服务器的监听地址
	Icon:        "Reading",
	MenuName:    "主机日志",
	PluginType:  "micro-app",
}
