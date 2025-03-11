/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package global

import (
	"strings"

	"github.com/pkg/errors"
)

func InitOSName() {
	contents, err := FileReadString("/etc/system-release")
	if err != nil {
		ERManager.ErrorTransmit("global", "error", errors.Errorf("fail to init os name: %s", err.Error()), true, false)
	}
	OsName = strings.Split(contents, " ")[0]
	if OsName != "openEuler" && OsName != "Kylin" {
		ERManager.ErrorTransmit("global", "error", errors.Errorf("unsupport os version: %s", OsName), true, false)
	}
}
