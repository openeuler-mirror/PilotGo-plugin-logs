/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package middleware

import (
	"time"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func Logger(_skipPaths []string) gin.HandlerFunc {
	var skip map[string]struct{}

	if len(_skipPaths) > 0 {
		skip = make(map[string]struct{}, len(_skipPaths))

		for _, path := range _skipPaths {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if _, ok := skip[path]; !ok {
			endTime := time.Now()
			latency := endTime.Sub(start)
			method := c.Request.Method
			statusCode := c.Writer.Status()
			clientIP := c.ClientIP()
			errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

			if raw != "" {
				path = path + "?" + raw
			}

			if latency > time.Minute {
				latency = latency.Truncate(time.Second)
			}

			global.ERManager.ErrorTransmit("gin", "debug", errors.Errorf("|%3d| %-13v | %-15s |%-7s %#v",
				statusCode,
				latency,
				clientIP,
				method,
				path),
				false, false,
			)
			if errorMessage != "" {
				global.ERManager.ErrorTransmit("gin", "error", errors.New(errorMessage), false, false)
			}
		}
	}
}
