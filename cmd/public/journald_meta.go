/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package public

type JournalctlOptions struct {
	Since      string `json:"since"`
	Until      string `json:"until"`
	Unit       string `json:"unit"`
	Identifier string `json:"identifier"`
	Severity   string `json:"severity"`
	Transport  string `json:"transport"`
	Notail     bool   `json:"notail"`
	User       string `json:"user"` // root:0
	From       int    `json:"from"`
	Size       int    `json:"size"`
}

type JMessage struct {
	Type     int                `json:"type"`
	JOptions *JournalctlOptions `json:"joptions"`
	Data     interface{}        `json:"data"`
}

// 客户端与logs agent之间websocket通信的消息类型
const (
	UpdateOptionsMsg int = iota
	AgentAddrMsg
	UnitListMsg
	ConnectedMsg
	DataMsg
	UpdatePageMsg
	DialFailedMsg
)

type StdoutDataType int

type StdoutData struct {
	Type StdoutDataType `json:"type"`
	Data interface{}    `json:"data"`
}

// shell命令stdout数据类型
const (
	LogEntryData StdoutDataType = iota
	UnitData
)

type PageData struct {
	Total int                      `json:"total"`
	Hits  []map[string]interface{} `json:"hits"`
}
