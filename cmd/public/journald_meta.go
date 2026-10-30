/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package public

// JournalctlOptions 描述一次 journalctl 日志查询的过滤条件与分页参数，由前端通过
// WebSocket 下发给 logs agent，agent 据此拼装 journalctl 命令行参数。各字段含义：
//   - Since / Until：起止时间，映射为 --since / --until（仅在 Notail 为 true 时生效）
//   - Unit：指定 systemd 单元，映射为 --unit
//   - Identifier：syslog 标识符，映射为 --identifier
//   - Severity：日志优先级，映射为 --priority
//   - Transport：日志传输来源，映射为匹配字段 _TRANSPORT=<value>
//   - Notail：为 true 时附加 --no-tail 仅返回结果，否则 --follow 实时跟踪
//   - User：按用户过滤，格式为 name:uid（如 root:0），映射为 _UID=<uid>
//   - From / Size：分页起始下标（从 0 开始）与单页返回条数
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

// JMessage 是客户端与 logs agent 之间 WebSocket 通信的消息封装。
// Type 为消息类型（取值见下方常量），JOptions 为按需携带的 journalctl 查询参数，
// Data 为消息负载数据。
type JMessage struct {
	Type     int                `json:"type"`
	JOptions *JournalctlOptions `json:"joptions"`
	Data     interface{}        `json:"data"`
}

// 客户端与logs agent之间websocket通信的消息类型，按声明顺序依次为：
// UpdateOptionsMsg 下发/更新 journalctl 查询参数；AgentAddrMsg 上报 agent 地址；
// UnitListMsg 请求 systemd 服务单元列表；ConnectedMsg 连接建立通知；
// DataMsg 日志数据推送；UpdatePageMsg 翻页请求；DialFailedMsg 建立连接失败通知。
const (
	UpdateOptionsMsg int = iota
	AgentAddrMsg
	UnitListMsg
	ConnectedMsg
	DataMsg
	UpdatePageMsg
	DialFailedMsg
)

// StdoutDataType 标识 shell 命令 stdout 回传数据的类型。
type StdoutDataType int

// StdoutData 是 agent 将 shell 命令输出回传给客户端的数据封装，
// Type 为数据类型（取值见下方常量），Data 为实际数据内容。
type StdoutData struct {
	Type StdoutDataType `json:"type"`
	Data interface{}    `json:"data"`
}

// shell命令stdout数据类型：LogEntryData 为日志条目数据，UnitData 为 systemd 服务单元列表数据。
const (
	LogEntryData StdoutDataType = iota
	UnitData
)

// PageData 是分页查询 journalctl 日志的返回结果，
// Total 为日志总条数，Hits 为当前页的日志条目。
type PageData struct {
	Total int                      `json:"total"`
	Hits  []map[string]interface{} `json:"hits"`
}
