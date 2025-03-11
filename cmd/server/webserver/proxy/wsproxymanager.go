/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package proxy

import (
	"sync"
	"time"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"github.com/pkg/errors"
)

var WebsocketProxyManager *WebsocketProxyManagement

type WebsocketProxyManagement struct {
	// key: web client id
	WebsocketProxyMap map[string]*WebsocketForwardProxy

	// 终止采集组件状态检测
	heartbeatDone chan struct{}

	once sync.Once
}

func CreateWebsocketProxyManagement() {
	WebsocketProxyManager = &WebsocketProxyManagement{
		WebsocketProxyMap: make(map[string]*WebsocketForwardProxy),
		heartbeatDone:     make(chan struct{}),
	}

	go WebsocketProxyManager.heartbeatDetect()
}

func (wpm *WebsocketProxyManagement) Add(_id string, _wsproxy *WebsocketForwardProxy) {
	wpm.WebsocketProxyMap[_id] = _wsproxy
}

func (wpm *WebsocketProxyManagement) Delete(_id string) {
	delete(wpm.WebsocketProxyMap, _id)
}

func (wpm *WebsocketProxyManagement) heartbeatDetect() {
	for {
		select {
		case <-wpm.heartbeatDone:
			return
		case <-time.After(global.HeartbeatPeriod):
			for _id, _jc := range wpm.WebsocketProxyMap {
				if !_jc.Active {
					global.ERManager.ErrorTransmit("webserver", "info", errors.Errorf("remove websocket proxy client: %s", _id), false, false)
					wpm.Delete(_id)
				}
			}
		}
	}
}

func (wpm *WebsocketProxyManagement) CloseAll() {
	wpm.once.Do(func() {
		close(wpm.heartbeatDone)
	})

	for id, wsproxy := range wpm.WebsocketProxyMap {
		global.ERManager.ErrorTransmit("webserver", "info", errors.Errorf("shutdown websocket proxy: %s", id), false, false)
		wsproxy.Close(true, false, false)
	}
}
