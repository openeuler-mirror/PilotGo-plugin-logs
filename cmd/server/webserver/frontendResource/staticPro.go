/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
//go:build production
// +build production

package frontendResource

import (
	"embed"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"strings"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"github.com/gin-gonic/gin"
)

//go:embed assets index.html
var StaticFiles embed.FS

func StaticRouter(router *gin.Engine) {
	sf, err := fs.Sub(StaticFiles, "assets")
	if err != nil {
		global.ERManager.ErrorTransmit("webserver", "warn", errors.New(err.Error()), false, false)
		return
	}

	mime.AddExtensionType(".js", "application/javascript")
	static := router.Group("/plugin/logs")
	{
		static.StaticFS("/assets", http.FS(sf))
		static.GET("/", func(c *gin.Context) {
			c.FileFromFS("/", http.FS(StaticFiles))
		})

	}

	// 解决页面刷新404的问题
	router.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.RequestURI, "/plugin/logs/api") {
			c.FileFromFS("/", http.FS(StaticFiles))
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	})

}
