/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package logtools

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/global"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/agent/logtools/journald"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/public"
	"github.com/pkg/errors"
)

var LogCollector *LogClientManagement

const (
	JournaldLogClientType int = iota
)

type LogClientManagement struct {
	// key: web client id
	journaldClients map[string]*journald.JournaldClient

	// 终止采集组件状态检测
	heartbeatDone chan struct{}

	// 终止周期性采集日志时间轴数据
	timelineDone chan struct{}

	once sync.Once
}

func CreateLogClientsManager() {
	LogCollector = &LogClientManagement{
		journaldClients: make(map[string]*journald.JournaldClient),
		heartbeatDone:   make(chan struct{}),
		timelineDone:    make(chan struct{}),
	}

	go LogCollector.heartbeatDetect()
	// go LogCollector.logTimeline()
}

func (lcm *LogClientManagement) Add(_type int, _id string, _c interface{}) error {
	switch _type {
	case JournaldLogClientType:
		jc, ok := _c.(*journald.JournaldClient)
		if !ok {
			return fmt.Errorf("fail to add log collect client: %+v", _c)
		}
		lcm.journaldClients[_id] = jc
	}
	return nil
}

func (lcm *LogClientManagement) Get(_type int, _id string) (interface{}, bool) {
	switch _type {
	case JournaldLogClientType:
		jc, ok := lcm.journaldClients[_id]
		if ok {
			return jc, ok
		}
	}
	return nil, false
}

func (lcm *LogClientManagement) Delete(_type int, _id string) {
	switch _type {
	case JournaldLogClientType:
		delete(lcm.journaldClients, _id)
	}
}

func (lcm *LogClientManagement) ReturnLogClients(_type int) interface{} {
	switch _type {
	case JournaldLogClientType:
		return lcm.journaldClients
	}
	return nil
}

func (lcm *LogClientManagement) heartbeatDetect() {
	for {
		select {
		case <-lcm.heartbeatDone:
			return
		case <-time.After(global.HeartbeatPeriod):
			for _id, _jc := range lcm.journaldClients {
				if !_jc.Active {
					global.ERManager.ErrorTransmit("logtools", "info", errors.Errorf("remove web client: %s", _id), false, false)
					lcm.Delete(JournaldLogClientType, _id)
				}
			}
		}
	}
}

func (lcm *LogClientManagement) LogTimeline() error {
	for {
		select {
		case <-lcm.timelineDone:
			return fmt.Errorf("timeline done channel closed")
		case t := <-time.After(60 * time.Second):
			if len(lcm.journaldClients) == 0 {
				return fmt.Errorf("no active journald clients")
			}

			tmpjclient := journald.CreateJournaldClient(nil, global.ReadCmdStderrTimeout)
			cmd := exec.Command("systemctl", journald.UnitListDefaultOptions...)
			tmpjclient.ProcessData(cmd, public.UnitData)
			go tmpjclient.WriteMessageToClient()

			entry_count := make(map[string]string)
			for _, _unit := range tmpjclient.UnitsMap["systemd"] {
				since := t.Local().Format("2024-01-02 15:04:05")
				until := t.Add(-60 * time.Second).Local().Format("2024-01-02 15:04:05")
				cmd = exec.Command("/bin/bash", "-c", "journalctl", "--quiet", "--no-pager", "--unit", _unit+".service", "--since", since, "--until", until, "|", "wc", "-l")
				outputBytes, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("log timeline cmd error: %s, output: %v", err.Error(), string(outputBytes))
				}
				entry_count[_unit] = string(outputBytes)
			}

			fmt.Printf("\033[32m>>>\033[0m entry count: %+v\n", entry_count)

			tmpjclient.Close(false, true, false)
		}
	}

}

func (lcm *LogClientManagement) CloseAll() {
	lcm.once.Do(func() {
		close(lcm.heartbeatDone)
	})

	// lcm.timelineDoneCh <- struct{}{}
	for id, jc := range lcm.journaldClients {
		global.ERManager.ErrorTransmit("logtools", "info", errors.Errorf("shutdown journald client: %s", id), false, false)
		jc.CloseReadMsgCh <- struct{}{}
		jc.Close(true, true, false)
	}
}
