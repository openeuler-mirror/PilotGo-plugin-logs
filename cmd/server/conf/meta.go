/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package conf

type LogsConf struct {
	Https_enabled bool   `yaml:"https_enabled"`
	CertFile      string `yaml:"cert_file"`
	KeyFile       string `yaml:"key_file"`
	Addr          string `yaml:"server_listen_addr"`
	Addr_target   string `yaml:"server_target_addr"`
}

type PilotGoConf struct {
	Addr string `yaml:"addr"`
}
