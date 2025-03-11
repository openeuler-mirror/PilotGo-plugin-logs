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
	"strings"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/pluginclient"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/webserver/proxy"
	"gitee.com/openeuler/PilotGo/sdk/common"
	"gitee.com/openeuler/PilotGo/sdk/response"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var ResultOptMsg = []string{"安装成功", "卸载成功"}

const (
	CommandInstall_Cmd = "yum install -y PilotGo-plugin-logs-agent && (echo '安装成功'; systemctl start PilotGo-plugin-logs-agent) || echo '安装失败'"
	CommandRemove_Cmd  = "yum remove -y PilotGo-plugin-logs-agent && echo '卸载成功' || echo '卸载失败'"
)

// 运行远程命令安装、卸载exporter
func RunCommandHandle(_ctx *gin.Context) {
	d := &struct {
		MachineUUIDs []string `json:"uuids"`
	}{}
	if err := _ctx.ShouldBind(d); err != nil {
		response.Fail(_ctx, nil, "parameter error")
		global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("fail to bind batch param: %s", err.Error()), false, false)
		return
	}

	var command string
	command_type := _ctx.Query("type")
	if command_type == "install" {
		command = CommandInstall_Cmd
	} else if command_type == "uninstall" {
		command = CommandRemove_Cmd
	} else {
		response.Fail(_ctx, nil, "请重新检查命令参数type")
		global.ERManager.ErrorTransmit("webserver", "error", errors.New("fail to resolve query param"), false, false)
		return
	}

	run_result := func(result []*common.CmdResult) {
		for _, res := range result {
			switch command_type {
			case "install":
				if !strings.Contains(res.Stdout, "成功") && !strings.Contains(res.Stdout, "success") {
					global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("failed runcommand result(%s, %d): %s", res.MachineIP, res.RetCode, res.Stderr), false, false)
				}
			case "uninstall":
				if !strings.Contains(res.Stdout, "成功") && !strings.Contains(res.Stdout, "success") {
					global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("failed runcommand result(%s, %d): %s", res.MachineIP, res.RetCode, res.Stderr), false, false)
				}
			}
		}
	}
	dd := &common.Batch{
		MachineUUIDs: d.MachineUUIDs,
	}
	err := pluginclient.Global_Client.RunCommandAsync(dd, command, run_result)
	if err != nil {
		response.Fail(_ctx, nil, err.Error())
		global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("fail to %s PilotGo-plugin-logs-agent to %v: %s", command_type, dd.MachineUUIDs, err.Error()), false, false)
		return
	}
	response.Success(_ctx, nil, "指令下发完成")
}

func GetIpListHandle(_ctx *gin.Context) {
	if pluginclient.Global_Client == nil {
		err := errors.New("Global_Client is nil")
		response.Fail(_ctx, nil, err.Error())
		global.ERManager.ErrorTransmit("webserver", "error", err, true, false)
		return
	}

	// ttcode
	// if cookie, err := _ctx.Request.Cookie("ws_session_id"); err != nil {
	// 	fmt.Printf(">>>cookie: %+v\n", err.Error())
	// } else {
	// 	fmt.Printf(">>>cookie: %+v\n", cookie)
	// }

	machine_list, err := pluginclient.Global_Client.MachineList()
	if err != nil {
		err = errors.New(err.Error())
		response.Fail(_ctx, nil, err.Error())
		global.ERManager.ErrorTransmit("agentmanager", "error", err, false, false)
	}

	machine_ip_list := []string{}
	for _, m := range machine_list {
		if global.IsIPandPORTValid(m.IP, "9995") {
			machine_ip_list = append(machine_ip_list, m.IP)
		} else {
			continue
		}
	}
	response.Success(_ctx, machine_ip_list, "")
}

func WebsocketProxyHandle(_ctx *gin.Context) {
	wsproxy := proxy.NewWebsocketForwardProxy()
	wsproxy.ID = _ctx.Request.Header.Get("clientId")
	wsproxy.Active = true
	if proxy.WebsocketProxyManager == nil {
		global.ERManager.ErrorTransmit("webserver", "error", errors.New("WebsocketProxyManager is nil"), true, false)
		_ctx.JSON(http.StatusInternalServerError, "WebsocketProxyManager is nil")
		return
	}
	proxy.WebsocketProxyManager.Add(wsproxy.ID, wsproxy)
	wsproxy.ServeHTTP(_ctx.Writer, _ctx.Request)
}
