/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package public

import (
	"fmt"
	"runtime"
)

func DebugStackTrace() {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	fmt.Printf("StackTrace (length: %d)\n%s\n", n, buf[:n])
}
