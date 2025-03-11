/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"gitee.com/openeuler/PilotGo/sdk/logger"
)

type WebsocketTcpHijackProxy struct{}

func InitWebServerTcpHijackProxy() {
	logger.Info("logs websocket proxy server started on %s", "10.41.107.29:9994")
	go func() {
		if err := http.ListenAndServe("10.41.107.29:9994", &WebsocketTcpHijackProxy{}); err != nil {
			logger.Fatal(err.Error())
		}
	}()
}

func (h *WebsocketTcpHijackProxy) ServeHTTP(_w http.ResponseWriter, _r *http.Request) {
	if _r.Method == http.MethodConnect {
		h.websocketConnectProxyHandle(_w, _r)
	}
}

// websocket代理：通过劫持客户端tcp连接及与目标服务器建立tcp连接的方式转发
func (h *WebsocketTcpHijackProxy) websocketConnectProxyHandle(_w http.ResponseWriter, _r *http.Request) {
	target_netconn, err := net.Dial("tcp", _r.Host)
	if err != nil {
		logger.Error("%d, Unable to connect to target", http.StatusServiceUnavailable)
		http.Error(_w, "Unable to connect to target", http.StatusServiceUnavailable)
		return
	}
	defer target_netconn.Close()

	_w.WriteHeader(http.StatusOK)

	hijacker, ok := _w.(http.Hijacker)
	if !ok {
		logger.Error("%d, Hijacking not supported", http.StatusInternalServerError)
		http.Error(_w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_netconn, _, err := hijacker.Hijack()
	if err != nil {
		logger.Error("%d, Hijack failed", http.StatusServiceUnavailable)
		client_netconn.Write([]byte(fmt.Sprintf("Unable to connect to target, code: %d", http.StatusServiceUnavailable)))
		return
	}
	defer client_netconn.Close()

	go io.Copy(target_netconn, client_netconn)
	io.Copy(client_netconn, target_netconn)
}

// 实现两个tcp net.conn间的数据传输，暂时弃用
func (h *WebsocketTcpHijackProxy) TransferMessages(_srcConn, _destConn net.Conn, _err_ch chan error) {
	for {
		buf := make([]byte, 4096)
		n, err := _srcConn.Read(buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			_err_ch <- err
			return
		}
		if n > 0 {
			_, err = _destConn.Write(buf[:n])
			if err != nil {
				_err_ch <- err
				return
			}
		}
	}
}
