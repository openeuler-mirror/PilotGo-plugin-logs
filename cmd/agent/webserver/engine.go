/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package webserver

import (
	"net/http"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/conf"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/global"
	"github.com/pkg/errors"
)

func InitWebserver() {
	http.HandleFunc("/ws/entry", entryHandle)

	go func() {
		global.ERManager.ErrorTransmit("webserver", "info", errors.Errorf("WebSocket server started on %s", conf.Global_Config.Logs.Addr), false, false)
		if conf.Global_Config.Logs.Https_enabled {
			if err := http.ListenAndServeTLS(conf.Global_Config.Logs.Addr, conf.Global_Config.Logs.CertFile, conf.Global_Config.Logs.KeyFile, nil); err != nil {
				global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("Error starting server: %s", err), true, false)
				return
			}
		} else {
			if err := http.ListenAndServe(conf.Global_Config.Logs.Addr, nil); err != nil {
				global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("Error starting server: %s", err), true, false)
				return
			}
		}
	}()
}
