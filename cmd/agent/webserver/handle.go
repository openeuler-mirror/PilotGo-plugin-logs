/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package webserver

import (
	"fmt"
	"net/http"
	"strings"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/global"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/logtools"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/logtools/journald"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许来自任何来源的连接
	},
}

func entryHandle(_w http.ResponseWriter, _r *http.Request) {
	global.ERManager.ErrorTransmit("webserver", "debug", errors.Errorf(
		"r.method: %v, r.proto: %v, r.host: %v, r.header: %+v, r.URL.scheme: %v, r.URL.host: %v, r.URL.path: %v",
		_r.Method, _r.Proto, _r.Host, _r.Header, _r.URL.Scheme, _r.URL.Host, _r.URL.Path),
		false, false)

	conn, err := upgrader.Upgrade(_w, _r, nil)
	if err != nil {
		global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("Error while upgrading connection: %s", err.Error()), false, false)
		_w.Write([]byte(fmt.Sprintf("Error while upgrading connection: %s", err.Error())))
		return
	}
	defer conn.Close()

	global.ERManager.ErrorTransmit("webserver", "info", errors.Errorf("connected to ws client: %s", strings.Split(_r.Header.Get("X-Forwarded-For"), ",")[0]), false, false)

	jclient := journald.CreateJournaldClient(conn, global.ReadCmdStderrTimeout)
	jclient.ID = _r.Header.Get("clientId")

	jclient.Active = true
	if logtools.LogCollector == nil {
		global.ERManager.ErrorTransmit("webserver", "error", errors.New("logcollector is nil"), false, false)
		_w.Write([]byte("logcollector is nil"))
		return
	}
	if err := logtools.LogCollector.Add(logtools.JournaldLogClientType, jclient.ID, jclient); err != nil {
		global.ERManager.ErrorTransmit("webserver", "error", errors.New(err.Error()), false, false)
		_w.Write([]byte(err.Error()))
		return
	}
	jclient.ReadMessageFromClient()
}
