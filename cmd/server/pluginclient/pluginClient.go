/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2.
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package pluginclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/conf"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"gitee.com/openeuler/PilotGo/sdk/common"
	"gitee.com/openeuler/PilotGo/sdk/plugin/client"
)

var Global_Client *client.Client

var Global_Context context.Context

func InitPluginClient() {
	if conf.Global_Config != nil && conf.Global_Config.Logs.Https_enabled {
		PluginInfo.Url = fmt.Sprintf("https://%s", conf.Global_Config.Logs.Addr_target)
	} else if conf.Global_Config != nil && !conf.Global_Config.Logs.Https_enabled {
		PluginInfo.Url = fmt.Sprintf("http://%s", conf.Global_Config.Logs.Addr_target)
	} else {
		global.ERManager.ErrorTransmit("pluginclient", "error", errors.New("Global_Config is nil"), true, false)
	}

	Global_Client = client.DefaultClient(PluginInfo)

	// 注册插件扩展点
	var ex []common.Extention
	me1 := &common.MachineExtention{
		Type:       common.ExtentionMachine,
		Name:       "安装日志agent",
		URL:        "/plugin/logs/api/runcommand?type=install",
		Permission: "plugin.logs.agent/install",
	}
	me2 := &common.MachineExtention{
		Type:       common.ExtentionMachine,
		Name:       "卸载日志agent",
		URL:        "/plugin/logs/api/runcommand?type=uninstall",
		Permission: "plugin.logs.agent/uninstall",
	}
	pe1 := &common.PageExtention{
		Type:       common.ExtentionPage,
		Name:       "日志查询",
		URL:        "/page",
		Permission: "plugin.logs.page/menu",
	}
	// be1 := &common.BatchExtention{
	// 	Type:       common.ExtentionBatch,
	// 	Name:       "批次扩展",
	// 	URL:        "/batch",
	// 	Permission: "plugin.logs/function",
	// }
	ex = append(ex, pe1, me1, me2)
	Global_Client.RegisterExtention(ex)

	tag_cb := func(uuids []string) []common.Tag {
		machines, err := Global_Client.MachineList()
		if err != nil {
			return nil
		}

		var mu sync.Mutex
		var wg sync.WaitGroup
		var tags []common.Tag
		for _, m := range machines {
			wg.Add(1)
			go func(_m *common.MachineNode) {
				if global.IsIPandPORTValid(_m.IP, "9995") {
					tag := common.Tag{
						UUID: _m.UUID,
						Type: common.TypeOk,
						Data: "日志",
					}
					mu.Lock()
					tags = append(tags, tag)
					mu.Unlock()
				} else {
					tag := common.Tag{
						UUID: _m.UUID,
						Type: common.TypeError,
						Data: "",
					}
					mu.Lock()
					tags = append(tags, tag)
					mu.Unlock()
				}
				wg.Done()
			}(m)
		}
		wg.Wait()
		return tags
	}
	Global_Client.OnGetTags(tag_cb)

	addPermissions()

	Global_Context = context.Background()
}

func addPermissions() {
	var pe []common.Permission
	p1 := common.Permission{
		Resource: "logs",
		Operate:  "menu",
	}

	p2 := common.Permission{
		Resource: "logs_operate",
		Operate:  "button",
	}

	p := append(pe, p1, p2)
	Global_Client.RegisterPermission(p)
}
