/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn> 
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package webserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/conf"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/global"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/pluginclient"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/webserver/frontendResource"
	"gitee.com/openeuler/PilotGo-plugin-logs/cmd/server/webserver/middleware"
	"gitee.com/openeuler/PilotGo/sdk/logger"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func InitWebServer() {
	if pluginclient.Global_Client == nil {
		logger.Fatal("Global_Client is nil")
	}

	engine := gin.New()
	engine.Use(gin.Recovery(), middleware.Logger([]string{
		"/plugin_manage/bind",
		"/",
	}))
	gin.SetMode(gin.ReleaseMode)
	pluginclient.Global_Client.RegisterHandlers(engine)
	pluginRouter(engine)
	proxyRouter(engine)
	frontendResource.StaticRouter(engine)

	web := &http.Server{
		Addr:    conf.Global_Config.Logs.Addr,
		Handler: engine,
	}

	global.ERManager.Wg.Add(1)
	go func() {
		global.ERManager.ErrorTransmit("webserver", "info", errors.Errorf("logs server started on %s", conf.Global_Config.Logs.Addr), false, false)
		if conf.Global_Config.Logs.Https_enabled {
			if err := web.ListenAndServeTLS(conf.Global_Config.Logs.CertFile, conf.Global_Config.Logs.KeyFile); err != nil {
				if strings.Contains(err.Error(), "Server closed") {
					err = errors.New(err.Error())
					global.ERManager.ErrorTransmit("webserver", "info", err, false, false)
					return
				}
				err = errors.Errorf("%s, addr: %s", err.Error(), conf.Global_Config.Logs.Addr)
				global.ERManager.ErrorTransmit("webserver", "error", err, true, true)
			}
		}
		if err := web.ListenAndServe(); err != nil {
			if strings.Contains(err.Error(), "Server closed") {
				err = errors.New(err.Error())
				global.ERManager.ErrorTransmit("webserver", "info", err, false, false)
				return
			}
			err = errors.New(err.Error())
			global.ERManager.ErrorTransmit("webserver", "error", err, true, true)
		}
	}()

	go func() {
		defer global.ERManager.Wg.Done()

		<-global.ERManager.GoCancelCtx.Done()

		global.ERManager.ErrorTransmit("webserver", "info", errors.New("shutting down web server..."), false, false)

		ctx, cancel := context.WithTimeout(global.RootCtx, 1*time.Second)
		defer cancel()

		if err := web.Shutdown(ctx); err != nil {
			global.ERManager.ErrorTransmit("webserver", "error", errors.Errorf("web server shutdown error: %s", err.Error()), false, false)
		} else {
			global.ERManager.ErrorTransmit("webserver", "info", errors.New("web server stopped"), false, false)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	pluginclient.Global_Client.Wait4Bind()
}

func pluginRouter(_engine *gin.Engine) {
	pilotgoApi := _engine.Group("/plugin/logs/api")
	{
		pilotgoApi.GET("/ip_list", GetIpListHandle)

		pilotgoApi.POST("/runcommand", RunCommandHandle)
	}
}

func proxyRouter(_engine *gin.Engine) {
	_engine.GET("/ws/proxy", WebsocketProxyHandle)
}

// func testRouter(_engine *gin.Engine) {
// 	_engine.PUT("/files/:filename", func(_ctx *gin.Context) {
// 		filename, ok := _ctx.Params.Get("filename")
// 		if !ok {
// 			_ctx.JSON(http.StatusBadRequest, "path param error")
// 			return
// 		}

// 		file, err := os.Create(fmt.Sprintf("/home/wjq/%s", filename))
// 		if err != nil {
// 			_ctx.JSON(http.StatusBadRequest, fmt.Sprintf("fail to create file: %s, %s", filename, err.Error()))
// 			return
// 		}
// 		defer file.Close()

// 		_, err = io.Copy(file, _ctx.Request.Body)
// 		if err != nil {
// 			_ctx.JSON(http.StatusBadRequest, fmt.Sprintf("io.copy error: %s", err.Error()))
// 			return
// 		}

// 		_ctx.JSON(http.StatusCreated, "")
// 	})
// }
