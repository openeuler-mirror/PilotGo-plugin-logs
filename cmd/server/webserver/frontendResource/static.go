//go:build !production
// +build !production

/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2.
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */

package frontendResource

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func StaticRouter(router *gin.Engine) {
	router.Static("/plugin/logs/assets", "../../web/dist/assets")
	router.StaticFile("/plugin/logs", "../../web/dist/index.html")

	// 解决页面刷新404的问题
	router.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.RequestURI, "/plugin/logs/api") {
			c.File("../../web/dist/index.html")
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	})

}
